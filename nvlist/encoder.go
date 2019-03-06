package nvlist

import (
	"encoding/binary"
	"math"
	"reflect"
	"strings"
)

// Marshal serializes the given data into a ZFS-style nvlist
func Marshal(val interface{}) ([]byte, error) {
	writer := nvlistWriter{
		flags: uniqueNameFlag,
	}
	if err := writer.writeNvHeader(); err != nil {
		return nil, err
	}
	if err := writer.writeNvPairs(reflect.ValueOf(val)); err != nil {
		return nil, err
	}
	return writer.nvlist, nil
}

type nvlistWriter struct {
	nvlist                 []byte
	nvpairStartByte        int
	modeWriteFromStartByte bool
	endianness             binary.ByteOrder
	flags                  uint32
	version                int32
}

func (w *nvlistWriter) WriteByte(c byte) error {
	w.nvlist = append(w.nvlist, c)
	return nil
}

func (w *nvlistWriter) skipN(len int) {
	for i := 0; i < len; i++ {
		w.nvlist = append(w.nvlist, 0x00)
	}
}

func (w *nvlistWriter) skipToAlign() {
	var padSize int
	if (len(w.nvlist)-w.nvpairStartByte)%8 != 0 {
		padSize = 8 - ((len(w.nvlist) - w.nvpairStartByte) % 8)
	}
	for i := 0; i < padSize; i++ {
		w.WriteByte(0x0)
	}
}

func (w *nvlistWriter) writeInt(val interface{}) error {
	return binary.Write(w, w.endianness, val)
}

func (w *nvlistWriter) writeString(str string) error {
	for i := 0; i < len(str); i++ {
		if str[i] == 0x00 {
			return ErrInvalidValue
		}
		w.nvlist = append(w.nvlist, str[i])
	}
	w.nvlist = append(w.nvlist, 0x00) // Null byte
	return nil
}

func (w *nvlistWriter) Write(buf []byte) (int, error) {
	if w.modeWriteFromStartByte {
		for i, b := range buf {
			w.nvlist[w.nvpairStartByte+i] = b
		}
		w.nvpairStartByte += len(buf)
	} else {
		for _, b := range buf {
			w.nvlist = append(w.nvlist, b)
		}
	}
	return len(buf), nil
}

func (w *nvlistWriter) writeNvHeader() error {
	if err := w.WriteByte(byte(EncodingNative)); err != nil {
		return err
	}
	// TODO: Actually deal with BE
	if err := w.WriteByte(littleEndian); err != nil {
		return err
	}
	w.endianness = binary.LittleEndian

	w.skipN(2) // reserved

	w.writeInt(w.version)
	w.writeInt(w.flags)

	return nil
}

func unpackVal(v reflect.Value) reflect.Value {
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return reflect.Value{}
		}
		v = v.Elem()
	}
	if v.Kind() == reflect.Interface {
		if v.IsNil() {
			return reflect.Value{}
		}
		v = v.Elem()
	}
	return v
}

func unpackType(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() == reflect.Interface {
		t = t.Elem()
	}
	return t
}

func (w *nvlistWriter) startNvPair() {
	w.nvpairStartByte = len(w.nvlist)
	for i := 0; i < nvlistHeaderSize; i++ {
		w.nvlist = append(w.nvlist, 0x00)
	}
}

func (w *nvlistWriter) endNvPair(nvp nvpair) {
	if nvp.Type == typeUnknown {
		panic("Unknown type hit")
	}
	if w.nvpairStartByte == 0 {
		panic("nvPair was not started")
	}
	w.skipToAlign()
	nvp.Size = int32(len(w.nvlist) - w.nvpairStartByte)
	w.modeWriteFromStartByte = true
	w.writeInt(nvp.Size)
	w.writeInt(nvp.Name_sz)
	w.writeInt(nvp.Reserve)
	w.writeInt(nvp.Value_elem)
	w.writeInt(nvp.Type)
	w.modeWriteFromStartByte = false
	w.nvpairStartByte = 0
}

func (w *nvlistWriter) writeNvlistHeader() {
	nvl := nvlist{
		Nvflag: uniqueNameFlag,
	}
	w.writeInt(nvl.Version)
	w.writeInt(nvl.Nvflag)
	w.writeInt(nvl.Priv)
	w.writeInt(nvl.Flag)
	w.writeInt(nvl.Pad)
}

func (w *nvlistWriter) writeNvPairs(v reflect.Value) error {
	v = unpackVal(v)

	if !v.IsValid() {
		// Null pointer
		return nil
	}

	var names []string
	var vals []reflect.Value

	switch v.Kind() {
	case reflect.Map:
		keys := v.MapKeys()
		for _, key := range keys {
			if key.Kind() != reflect.String {
				return ErrInvalidValue
			}
			val := unpackVal(v.MapIndex(key))
			if val.IsValid() {
				names = append(names, key.String())
				vals = append(vals, val)
			}
		}
	case reflect.Struct:
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			tags := strings.Split(t.Field(i).Tag.Get("nvlist"), ",")
			name := tags[0]
			val := unpackVal(v.Field(i))
			if len(tags) > 1 {
				switch tags[1] {
				case "omitempty":
					if isEmptyValue(val) {
						continue
					}
				case "ro": // Never marshal
					continue
				}
			}
			if val.IsValid() {
				if name == "" {
					names = append(names, t.Field(i).Name)
				} else {
					names = append(names, name)
				}
				vals = append(vals, val)
			}
		}
	default:
		return ErrInvalidValue
	}

	for i := 0; i < len(names); i++ {
		nameLen := len(names[i]) + 1
		if nameLen >= math.MaxInt16 {
			return ErrInvalidValue
		}
		nvp := nvpair{
			Size:       0,
			Name_sz:    int16(nameLen),
			Value_elem: 1,
			Type:       0,
		}

		t := vals[i].Kind()

		if t == reflect.Bool && !vals[i].Bool() {
			continue
		}

		w.startNvPair()

		w.writeString(names[i])
		w.skipToAlign()

		switch t {
		case reflect.Int8, reflect.Uint8, reflect.Int16, reflect.Uint16, reflect.Int32, reflect.Uint32, reflect.Int64, reflect.Uint64, reflect.Float64:
			nvp.Type = nvtypeFromKind(t)
			if err := w.writeInt(vals[i].Interface()); err != nil {
				return err
			}
			w.endNvPair(nvp)
		case reflect.Bool:
			nvp.Type = typeBoolean
			nvp.Value_elem = 0
			w.endNvPair(nvp)
		case reflect.Map, reflect.Struct:
			nvp.Type = typeNvlist
			w.writeNvlistHeader()
			w.endNvPair(nvp)
			if err := w.writeNvPairs(vals[i]); err != nil {
				return nil
			}
		case reflect.String:
			nvp.Type = typeString
			w.writeString(vals[i].String())
			w.endNvPair(nvp)
		case reflect.Array, reflect.Slice:
			if vals[i].Len() >= math.MaxInt32 {
				return ErrInvalidValue
			}
			nvp.Value_elem = int32(vals[i].Len())
			elemKind := unpackType(vals[i].Type().Elem()).Kind()
			switch elemKind {
			case reflect.Int8, reflect.Uint8, reflect.Int16, reflect.Uint16, reflect.Int32, reflect.Uint32, reflect.Int64, reflect.Uint64:
				nvp.Type = nvtypeFromArrayKind(elemKind)
				for j := 0; j < vals[i].Len(); j++ {
					if err := w.writeInt(vals[i].Index(j).Interface()); err != nil {
						return err
					}
				}
				w.endNvPair(nvp)
			case reflect.Bool:
				nvp.Type = typeBooleanArray
				for j := 0; j < vals[i].Len(); j++ {
					var val int32
					if unpackVal(vals[i].Index(j)).Bool() {
						val = 1
					}
					if err := w.writeInt(val); err != nil {
						return err
					}
				}
				w.endNvPair(nvp)
			case reflect.String:
				nvp.Type = typeStringArray
				w.skipN(8 * vals[i].Len()) // Skip pointers
				for j := 0; j < vals[i].Len(); j++ {
					w.writeString(unpackVal(vals[i].Index(j)).String())
				}
				w.endNvPair(nvp)
			case reflect.Struct, reflect.Map:
				nvp.Type = typeNvlistArray
				w.skipN(8 * vals[i].Len()) // Skip pointers
				for j := 0; j < vals[i].Len(); j++ {
					w.writeNvlistHeader()
				}
				w.endNvPair(nvp)
				for j := 0; j < vals[i].Len(); j++ {
					if err := w.writeNvPairs(vals[i].Index(j)); err != nil {
						return err
					}
				}
			default:
				return ErrInvalidValue
			}
		default:
			return ErrInvalidValue
		}
	}
	w.skipN(4) // 4 byte trailer
	return nil
}

func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return false
}
