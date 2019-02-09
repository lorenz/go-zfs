// +build ignore

package nvlist

//go:generate go tool cgo -godefs types_gen.go

//#include "types.h"
import "C"

type Nvpair C.struct_nvpair
type Nvlist C.struct_nvlist
type NvHeader C.struct_nvs_header
