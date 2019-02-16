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

		val, err := nvp.Type.decode(nvpr, uint32(nvp.Value_elem))
		if err != nil {
			return err
		}

		if v.Kind() == reflect.Map {
			v.SetMapIndex(reflect.ValueOf(name), reflect.ValueOf(val))
		}
		if v.Kind() == reflect.Struct {
			field, ok := structFieldByName[name]
			if !ok {
				return ErrInvalidValue
			}
			field.Set(reflect.ValueOf(val))
		}
	}
}
