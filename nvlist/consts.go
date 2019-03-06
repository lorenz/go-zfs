package nvlist

import "reflect"

type nvtype uint32

const (
	typeUnknown nvtype = iota
	typeBoolean
	typeByte
	typeInt16
	typeUint16
	typeInt32
	typeUint32
	typeInt64
	typeUint64
	typeString
	typeByteArray
	typeInt16Array
	typeUint16Array
	typeInt32Array
	typeUint32Array
	typeInt64Array
	typeUint64Array
	typeStringArray
	typeHrtime
	typeNvlist
	typeNvlistArray
	typeBooleanValue
	typeInt8
	typeUint8
	typeBooleanArray
	typeInt8Array
	typeUint8Array
	typeDouble
)

const nvlistHeaderSize = 16
const uniqueNameFlag = 0x01

var nvtypeFromKindMap = map[reflect.Kind]nvtype{
	reflect.Bool:    typeBooleanValue,
	reflect.Int8:    typeInt8,
	reflect.Int16:   typeInt16,
	reflect.Int32:   typeInt32,
	reflect.Int64:   typeInt64,
	reflect.Uint8:   typeByte, // Special case, probably needs override
	reflect.Uint16:  typeUint16,
	reflect.Uint32:  typeUint32,
	reflect.Uint64:  typeUint64,
	reflect.Float64: typeDouble,
	reflect.Map:     typeNvlist,
	reflect.String:  typeString,
	reflect.Struct:  typeNvlist,
}

var nvtypeFromArrayKindMap = map[reflect.Kind]nvtype{
	reflect.Bool:   typeBooleanArray,
	reflect.Int8:   typeInt8Array,
	reflect.Int16:  typeInt16Array,
	reflect.Int32:  typeInt32Array,
	reflect.Int64:  typeInt64Array,
	reflect.Uint8:  typeByteArray, // Special case, probably needs override
	reflect.Uint16: typeUint16Array,
	reflect.Uint32: typeUint32Array,
	reflect.Uint64: typeUint64Array,
	reflect.Map:    typeNvlistArray,
	reflect.String: typeStringArray,
	reflect.Struct: typeNvlistArray,
}

// nvtypeFromKind gets the nvtype from the given reflect kind for non-compound types
func nvtypeFromKind(kind reflect.Kind) nvtype {
	t, ok := nvtypeFromKindMap[kind]
	if !ok {
		return typeUnknown
	}
	return t
}

// nvtypeFromArrayKind gets the nvtype for an array of the given reflect kind
func nvtypeFromArrayKind(kind reflect.Kind) nvtype {
	t, ok := nvtypeFromArrayKindMap[kind]
	if !ok {
		return typeUnknown
	}
	return t
}
