// +build ignore

package ioctl

//go:generate go tool cgo -godefs types_gen.go

/*
#include "types.h"
*/
import "C"

type Cmd C.struct_zfs_cmd
type DMUObjectType C.enum_dmu_objset_type
type DMUObjectSetStats C.struct_dmu_objset_stats
type ZInjectRecord C.struct_zinject_record
type Share C.struct_zfs_share
type Stat C.struct_zfs_stat
type DRRBegin C.struct_drr_begin
type Ioctl C.enum_zfs_ioc
