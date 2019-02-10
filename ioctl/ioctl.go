package ioctl

import (
	"errors"
	"runtime"
	"unsafe"

	"git.dolansoft.org/lorenz/go-zfs/nvlist"
	"golang.org/x/sys/unix"
)

func NvlistIoctl(fd uintptr, ioctl Ioctl, name string, request interface{}, response interface{}) error {
	src, err := nvlist.Marshal(request)
	if err != nil {
		return err
	}
	// WARNING: Here be dragons! This is completely outside of Go's safety net and uses various
	// criticial runtime workarounds to make sure that memory is safely handled
	dst := make([]byte, 4096)
	cmd := &Cmd{
		Nvlist_src: uint64(uintptr(unsafe.Pointer(&src[0]))), Nvlist_src_size: uint64(len(src)),
		Nvlist_dst: uint64(uintptr(unsafe.Pointer(&dst[0]))), Nvlist_dst_size: uint64(len(dst)),
	}
	if len(name) > 4095 {
		return errors.New("Name is too big")
	}
	for i := 0; i < len(name); i++ {
		cmd.Name[i] = name[i]
	}
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, fd, uintptr(ioctl), uintptr(unsafe.Pointer(cmd)))
	runtime.KeepAlive(src)
	runtime.KeepAlive(dst)
	if errno != 0 {
		return errno
	}
	return nvlist.Unmarshal(dst, response)
}
