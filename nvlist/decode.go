// Package nvlist implements encoding and decoding of ZFS-style nvlists with an interface similar to
// that of encoding/json. It supports "native" encoding and parts of XDR in both big and little endian.
package nvlist

import (
	"encoding/binary"
	"errors"
	"io"
	"reflect"
	"strings"
)

var (
	ErrInvalidEncoding  = errors.New("this nvlist is not in native encoding")
	ErrInvalidEndianess = errors.New("this nvlist is neither in big nor in little endian")
	ErrInvalidData      = errors.New("this nvlist contains invalid data")
	ErrInvalidValue     = errors.New("the value provided to unmarshal contains invalid types")
	ErrUnsupportedType  = errors.New("this nvlist contains an unsupported type (hrtime)")
	errEndOfData        = errors.New("end of data")
)

// Encoding represents the encoding used for serialization/deserialization
type Encoding uint8

const (
	// EncodingNative is used in syscalls and cache files
	EncodingNative Encoding = 0x00
	// EncodingXDR is used on-disk (and is not actually XDR)
	EncodingXDR  Encoding = 0x01
	bigEndian             = 0x00
	littleEndian          = 0x01
)

// Unmarshal parses a ZFS-style nvlist in native encoding and with any endianness
func Unmarshal(data []byte, val interface{}) error {
	s := nvlistReader{
		nvlist: data,
	}
	if err := s.readNvHeader(); err != nil {
		return err
	}
	return s.readPairs(reflect.ValueOf(val))
}

type nvlistReader struct {
	nvlist      []byte
	currentByte int
	endianness  binary.ByteOrder
	encoding    Encoding
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
	if r.currentByte+len(p) < len(r.nvlist) {
		n = len(p)
	} else {
		n = len(r.nvlist) - r.currentByte
		err = io.EOF
	}
	for i := 0; i < n; i++ {
		p[i] = r.nvlist[r.currentByte+i]
	}
	r.currentByte += n
	return
}

func (r *nvPairReader) ReadByte() (byte, error) {
	if r.currentByte <= r.startByte+r.sizeBytes {
		val := r.nvlist.nvlist[r.currentByte]
		r.currentByte++
		return val, nil
	}
	return 0x00, ErrInvalidData
}

func (r *nvPairReader) ReadBytes(delim byte) ([]byte, error) {
	startByte := r.currentByte
	for ; r.currentByte <= r.startByte+r.sizeBytes; r.currentByte++ {
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
		err = io.EOF
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
	var alignment int
	switch r.nvlist.encoding {
	case EncodingNative:
		alignment = 8
	case EncodingXDR:
		alignment = 4
	default:
		panic("Invalid encoding inside parser")
	}
	if (r.currentByte-r.startByte)%alignment != 0 {
		r.currentByte += alignment - ((r.currentByte - r.startByte) % alignment)
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
	switch Encoding(encoding) {
	case EncodingNative:
		r.encoding = EncodingNative
	case EncodingXDR:
		r.encoding = EncodingXDR
	default:
		return ErrInvalidEncoding
	}

	endiness, err := r.ReadByte()
	if err != nil {
		return err
	}

	switch endiness {
	case bigEndian:
		r.endianness = binary.BigEndian
	case littleEndian:
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

func (r *nvlistReader) readPairs(v reflect.Value) error {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() == reflect.Interface && v.NumMethod() == 0 {
		val := make(map[string]interface{})
		v.Set(reflect.ValueOf(val))
		v = v.Elem()
	}
	structFieldByName := make(map[string]reflect.Value)
	if v.Kind() == reflect.Struct {
		t := v.Type()
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			tags := strings.Split(field.Tag.Get("nvlist"), ",")
			name := field.Name
			if tags[0] != "" {
				name = tags[0]
			}
			structFieldByName[name] = v.Field(i)
		}
	} else if v.Kind() == reflect.Map {
		// Noop, but valid
	} else {
		return ErrInvalidData
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
		if nvp.Size == 0 { // End indicated by zero size
			return nil
		}
		if int(nvp.Size)+r.currentByte > len(r.nvlist) {
			return ErrInvalidData
		}
		nvpr.sizeBytes = int(nvp.Size)
		r.skipN(int(nvp.Size) - 4) // Skip to next nvPair, subtract 4 already read size bytes

		if r.encoding == EncodingXDR {
			r.skipN(4) // Skip decoded size, it's irrelevant for us
		}

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

		if nvp.Value_elem < 0 {
			return ErrInvalidData
		}
		if nvp.Value_elem > 65535 { // 64K entries are enough
			return ErrInvalidData
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

		setPrimitive := func(value interface{}) {
			rValue := reflect.ValueOf(value)
			if rValue.Kind() == reflect.Ptr {
				rValue = rValue.Elem()
			}
			if v.Kind() == reflect.Struct {
				field := structFieldByName[name]
				if field.CanSet() {
					field.Set(rValue)
				}
			} else if v.Kind() == reflect.Map {
				v.SetMapIndex(reflect.ValueOf(name), rValue)
			}
		}

		switch nvp.Type {
		case typeUnknown:
			return ErrInvalidData
		case typeBoolean:
			setPrimitive(true)
		case typeInt16, typeUint16, typeInt32, typeUint32, typeInt64, typeUint64, typeInt8, typeUint8: // Integer-style types
			var val interface{}
			switch nvp.Type {
			case typeInt16:
				val = new(int16)
			case typeUint16:
				val = new(uint16)
			case typeInt32:
				val = new(int32)
			case typeUint32:
				val = new(uint32)
			case typeInt64:
				val = new(int64)
			case typeUint64:
				val = new(uint64)
			case typeInt8:
				val = new(int8)
			case typeUint8:
				val = new(uint8)
			default:
				panic("Primitive type with no handler (illegal state), check all primitive types are handled")
			}

			err := nvpr.readInt(val)
			if err != nil {
				return err
			}
			setPrimitive(val)
		case typeByte:
			b, err := nvpr.ReadByte()
			if err != nil {
				return err
			}
			setPrimitive(b)
		case typeString:
			data, err := nvpr.ReadBytes(0x00)
			if err != nil {
				return err
			}
			setPrimitive(string(data[:len(data)-1]))
		case typeBooleanValue:
			var tmp int32
			err := nvpr.readInt(&tmp)
			if err != nil {
				return err
			}
			var val bool
			switch tmp {
			case 0:
				val = false
			case 1:
				val = true
			default:
				return ErrInvalidData
			}
			setPrimitive(val)
		// Array handling
		case typeInt16Array, typeUint16Array, typeInt32Array, typeUint32Array, typeInt64Array, typeUint64Array, typeInt8Array, typeUint8Array:
			var val interface{}
			switch nvp.Type {
			case typeInt16Array:
				val = make([]int16, nvp.Value_elem)
			case typeUint16Array:
				val = make([]uint16, nvp.Value_elem)
			case typeInt32Array:
				val = make([]int32, nvp.Value_elem)
			case typeUint32Array:
				val = make([]uint32, nvp.Value_elem)
			case typeInt64Array:
				val = make([]int64, nvp.Value_elem)
			case typeUint64Array:
				val = make([]uint64, nvp.Value_elem)
			case typeInt8Array:
				val = make([]int8, nvp.Value_elem)
			case typeUint8Array:
				val = make([]uint8, nvp.Value_elem)
			default:
				panic("Array type with no handler (illegal state), check all primitive types are handled")
			}
			if err := binary.Read(&nvpr, nvpr.nvlist.endianness, val); err != nil {
				return err
			}
			setPrimitive(val)

		case typeByteArray:
			val, err := nvpr.readN(int(nvp.Value_elem))
			if err != nil {
				return err
			}
			setPrimitive(val)
		case typeStringArray:
			val := make([]string, nvp.Value_elem)
			nvpr.skipN(int(8 * nvp.Value_elem)) // Skip pointers
			// Pointers are always aligned
			for i := uint32(0); i < uint32(nvp.Value_elem); i++ {
				data, err := nvpr.ReadBytes(0x00)
				if err != nil {
					return err
				}
				val[i] = string(data[:len(data)-1])
			}
			setPrimitive(val)
		case typeBooleanArray:
			var tmp int32
			val := make([]bool, nvp.Value_elem)
			for i := uint32(0); i < uint32(nvp.Value_elem); i++ {
				if err := nvpr.readInt(&tmp); err != nil {
					return err
				}
				switch tmp {
				case 0:
					val[i] = false
				case 1:
					val[i] = true
				default:
					return ErrInvalidData
				}
			}
			setPrimitive(val)
		// Nvlist handling
		case typeNvlist:
			if v.Kind() == reflect.Struct {
				field := structFieldByName[name]
				if field.CanSet() {
					if err := nvpr.nvlist.readPairs(field); err != nil {
						return err
					}
				}
			} else if v.Kind() == reflect.Map {
				valueType := v.Type().Elem()
				var val reflect.Value
				if valueType.Kind() == reflect.Interface {
					val = reflect.ValueOf(make(map[string]interface{}))
				} else if valueType.Kind() == reflect.Struct {
					val = reflect.New(valueType)
				} else if valueType.Kind() == reflect.Map {
					val = reflect.MakeMap(reflect.MapOf(reflect.TypeOf(""), valueType.Elem()))
				} else {
					panic("Cannot currently handle complex hybrid types")
				}
				if err := nvpr.nvlist.readPairs(val); err != nil {
					return err
				}
				if val.Kind() == reflect.Ptr {
					v.SetMapIndex(reflect.ValueOf(name), val.Elem())
				} else {
					v.SetMapIndex(reflect.ValueOf(name), val)
				}
			} else {
				panic("Invalid pair type (not map or struct)")
			}
		case typeNvlistArray:
			var val reflect.Value
			if v.Kind() == reflect.Struct {
				panic("Deserializing NVListArrays into structs currently unsupported")
			} else if v.Kind() == reflect.Map {
				val = reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(map[string]interface{}{})), int(nvp.Value_elem), int(nvp.Value_elem))
				// Drop unused data (nvlist header @ 8 bytes + 64 bit pointer @ 8 bytes)
				nvpr.skipN(int((8 + 8) * nvp.Value_elem))
				for i := 0; i < int(nvp.Value_elem); i++ { // arraySize is <2^16
					val.Index(i).Set(reflect.MakeMap(val.Type().Elem()))
					err := nvpr.nvlist.readPairs(val.Index(i))
					if err != nil {
						return err
					}
				}
				v.SetMapIndex(reflect.ValueOf(name), val)
			} else {
				panic("Invalid pair type (not map or struct)")
			}

		}
	}
}
