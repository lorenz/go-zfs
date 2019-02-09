package nvlist

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
	"reflect"
)

var (
	ErrInvalidEncoding  = errors.New("this nvlist is not in native encoding")
	ErrInvalidEndianess = errors.New("this nvlist is neither in big nor in little endian")
	ErrInvalidData      = errors.New("this nvlist contains invalid data")
	ErrInvalidValue     = errors.New("the value provided to unmarshal contains invalid types")
	ErrUnsupportedType  = errors.New("this nvlist contains an unsupported type (hrtime)")
	errEndOfData        = errors.New("end of data")
)

const (
	NATIVE_ENCODING = 0x00
	BIG_ENDIAN      = 0x00
	LITTLE_ENDIAN   = 0x01
)

// Marshal serializes the given data into a ZFS-style nvlist
func Marshal(val interface{}) ([]byte, error) {
	writer := nvlistWriter{}
	if err := writer.writeNvHeader(); err != nil {
		return nil, err
	}
	if err := writer.writeNvPairs(reflect.ValueOf(val)); err != nil {
		return nil, err
	}
	return writer.buf.Bytes(), nil
}

// Unmarshal parses a ZFS-style nvlist in native encoding and with any endianness
func Unmarshal(data []byte, val interface{}) error {
	s := nvlistReader{
		nvlist: data,
	}
	if err := s.readNvHeader(); err != nil {
		return err
	}
	return s.readPairs(val)
}

type nvlistWriter struct {
	buf        bytes.Buffer
	nvlist     []byte
	endianness binary.ByteOrder
	flags      uint32
	version    int32
}

func (w *nvlistWriter) WriteByte(c byte) error {
	return w.buf.WriteByte(c)
}

func (w *nvlistWriter) skipN(len int) {
	for i := 0; i < len; i++ {
		w.buf.WriteByte(0x0)
	}
}

func (w *nvlistWriter) skipToAlign() {
	var len int
	if w.buf.Len()%8 != 0 {
		len = 8 - (w.buf.Len() % 8)
	}
	for i := 0; i < len; i++ {
		w.buf.WriteByte(0x0)
	}
}

func (w *nvlistWriter) writeInt(val interface{}) error {
	return binary.Write(&w.buf, w.endianness, val)
}

func (w *nvlistWriter) writeString(str string) error {
	// TODO: Filter for null bytes
	_, err := w.buf.WriteString(str)
	return err
}

func (w *nvlistWriter) writeNvHeader() error {
	if err := w.WriteByte(NATIVE_ENCODING); err != nil {
		return err
	}
	// TODO: Actually deal with BE
	if err := w.WriteByte(LITTLE_ENDIAN); err != nil {
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
		v = v.Elem()
	}
	if v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	return v
}

func (w *nvlistWriter) writeNvPairs(v reflect.Value) error {
	v = unpackVal(v)

	var names []string
	var vals []reflect.Value

	switch v.Kind() {
	case reflect.Map:
		keys := v.MapKeys()
		for _, key := range keys {
			if key.Kind() != reflect.String {
				return ErrInvalidValue
			}
			names = append(names, key.String())
			vals = append(vals, unpackVal(v.MapIndex(key)))
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			names = append(names, v.Type().Field(i).Name)
			vals = append(vals, unpackVal(v.Field(i)))
		}
	default:
		return ErrInvalidValue
	}
	for i := 0; i < len(names); i++ {
		nameLen := len(names[i]) + 1
		if nameLen > math.MaxInt16 {
			return ErrInvalidValue
		}
		nvp := nvpair{
			Size:       0,
			Name_sz:    int16(nameLen),
			Value_elem: 0,
			Type:       0,
		}

		w.writeInt(nvp.Size)
		w.writeInt(nvp.Name_sz)
		w.writeInt(nvp.Value_elem)
		w.writeInt(nvp.Type)

		w.buf.WriteString(names[i])
		w.WriteByte(0x00)
		w.skipToAlign()

		t := vals[i].Kind()
		switch t {
		case reflect.Int8, reflect.Uint8, reflect.Int16, reflect.Uint16, reflect.Int32, reflect.Uint32, reflect.Int64, reflect.Uint64, reflect.Float64:
			if err := w.writeInt(vals[i].Interface()); err != nil {
				return err
			}
		case reflect.Bool:
			var val int32
			if vals[i].Bool() {
				val = 1
			}
			if err := w.writeInt(val); err != nil {
				return err
			}
		case reflect.Map, reflect.Struct:
			// TODO: Write header
			w.skipN(8 + 8) // 8 bytes header + 8 bytes pointer
			if err := w.writeNvPairs(vals[i]); err != nil {
				return nil
			}
		case reflect.String:
			w.writeString(vals[i].String())
		case reflect.Array, reflect.Slice:
			nvp.Value_elem = int32(vals[i].Len()) // TODO: Might be a downcast
			switch vals[i].Type().Elem().Kind() { // TODO: Enhanced type unpacking
			case reflect.Int8, reflect.Uint8, reflect.Int16, reflect.Uint16, reflect.Int32, reflect.Uint32, reflect.Int64, reflect.Uint64:
				for j := 0; j < vals[i].Len(); j++ {
					if err := w.writeInt(vals[i].Index(j).Interface()); err != nil {
						return err
					}
				}
			case reflect.Bool:
				for j := 0; j < vals[i].Len(); j++ {
					var val int32
					if vals[i].Index(j).Bool() {
						val = 1
					}
					if err := w.writeInt(val); err != nil {
						return err
					}
				}
			case reflect.String:
				for j := 0; j < vals[i].Len(); j++ {
					w.writeString(vals[i].Index(j).String())
				}
			case reflect.Struct, reflect.Map:
				w.skipN((8 + 8) * vals[i].Len()) // TODO: First 8 bytes are technically nvlist header
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
		w.skipToAlign()
	}
	w.skipN(4) // 4 byte trailer
	return nil
}

type nvlistReader struct {
	nvlist      []byte
	currentByte int
	endianness  binary.ByteOrder
	flags       uint32
	version     int32
}

type nvPairReader struct {
	nvlist      *nvlistReader
	startByte   int
	sizeBytes   int
	currentByte int
}

func (r *nvlistReader) ReadByte() (byte, error) {
	if r.currentByte < len(r.nvlist) {
		val := r.nvlist[r.currentByte]
		r.currentByte++
		return val, nil
	}
	return 0x00, ErrInvalidData
}

func (r *nvlistReader) Read(p []byte) (n int, err error) {
	if r.currentByte+len(p) <= len(r.nvlist) {
		n = len(p)
	} else {
		n = len(r.nvlist) - r.currentByte
	}
	for i := 0; i < n; i++ {
		p[i] = r.nvlist[r.currentByte+i]
	}
	r.currentByte += n
	return
}

func (r *nvPairReader) ReadByte() (byte, error) {
	if r.currentByte < r.startByte+r.sizeBytes {
		val := r.nvlist.nvlist[r.currentByte]
		r.currentByte++
		return val, nil
	}
	return 0x00, ErrInvalidData
}

func (r *nvPairReader) ReadBytes(delim byte) ([]byte, error) {
	startByte := r.currentByte
	for ; r.currentByte < r.startByte+r.sizeBytes; r.currentByte++ {
		if r.nvlist.nvlist[r.currentByte] == delim {
			val := r.nvlist.nvlist[startByte : r.currentByte+1]
			r.currentByte++ // consume delimiter
			return val, nil
		}
	}
	return []byte{}, ErrInvalidData
}

func (r *nvPairReader) readN(n int) (val []byte, err error) {
	if r.currentByte+n <= r.startByte+r.sizeBytes {
		val = r.nvlist.nvlist[r.currentByte : r.currentByte+n]
		r.currentByte += n
		return
	}
	err = ErrInvalidData
	return
}

func (r *nvPairReader) Read(p []byte) (n int, err error) {
	if r.currentByte+len(p) <= r.startByte+r.sizeBytes {
		n = len(p)
	} else {
		n = r.startByte + r.sizeBytes - r.currentByte
	}
	for i := 0; i < n; i++ {
		p[i] = r.nvlist.nvlist[r.currentByte+i]
	}
	r.currentByte += n
	return
}

func (r *nvPairReader) readInt(val interface{}) error {
	return binary.Read(r, r.nvlist.endianness, val)
}

// Skips to next 8-byte aligned address inside nvPair
func (r *nvPairReader) skipToAlign() {
	if (r.currentByte-r.startByte)%8 != 0 {
		r.currentByte += 8 - ((r.currentByte - r.startByte) % 8)
	}
}

func (r *nvPairReader) skipN(n int) {
	r.currentByte += n
}

func (r *nvlistReader) skipN(n int) {
	r.currentByte += n
}

func (r *nvlistReader) readInt(data interface{}) error {
	return binary.Read(r, r.endianness, data)
}

func (r *nvlistReader) readNvHeader() error {
	encoding, err := r.ReadByte()
	if err != nil {
		return err
	}
	if encoding != NATIVE_ENCODING {
		return ErrInvalidEncoding
	}
	endiness, err := r.ReadByte()
	if err != nil {
		return err
	}

	switch endiness {
	case BIG_ENDIAN:
		r.endianness = binary.BigEndian
	case LITTLE_ENDIAN:
		r.endianness = binary.LittleEndian
	default:
		return ErrInvalidEndianess
	}

	r.skipN(2) // reserved

	if err := r.readInt(&r.version); err != nil {
		return err
	}
	if err := r.readInt(&r.flags); err != nil {
		return err
	}

	return nil
}

func (r *nvlistReader) readPairs(data interface{}) error {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
		if v.Kind() == reflect.Interface && v.NumMethod() == 0 {
			val := make(map[string]interface{})
			v.Set(reflect.ValueOf(val))
			v = v.Elem()
		}
	}
	for {
		var nvp nvpair
		nvpr := nvPairReader{
			nvlist:      r,
			currentByte: r.currentByte + 4, // Size (4 bytes)
			startByte:   r.currentByte,
		}
		if err := r.readInt(&nvp.Size); err != nil {
			return err
		}
		if nvp.Size < 0 {
			return ErrInvalidData
		}
		if nvp.Size == 0 {
			return nil
		}
		nvpr.sizeBytes = int(nvp.Size)
		r.skipN(int(nvp.Size) - 4) // Skip to next nvPair, subtract 4 already read size bytes

		if err := nvpr.readInt(&nvp.Name_sz); err != nil {
			return err
		}
		if nvp.Name_sz <= 0 { // Null terminated, so at least size 1 is required
			return ErrInvalidData
		}
		if err := nvpr.readInt(&nvp.Reserve); err != nil {
			return err
		}
		if err := nvpr.readInt(&nvp.Value_elem); err != nil {
			return err
		}
		if err := nvpr.readInt(&nvp.Type); err != nil {
			return err
		}

		nameRaw, err := nvpr.readN(int(nvp.Name_sz)) // Upcast: always OK
		if err != nil {
			return err
		}
		name := string(nameRaw[:len(nameRaw)-1]) // Remove null termination

		nvpr.skipToAlign()

		val, err := nvp.Type.decode(nvpr, uint32(nvp.Value_elem))
		if err != nil {
			return err
		}

		if v.Kind() == reflect.Map {
			v.SetMapIndex(reflect.ValueOf(name), reflect.ValueOf(val))
		}
		if v.Kind() == reflect.Struct {
			field := v.FieldByName(name)
			field.Set(reflect.ValueOf(val))
		}
	}
}
