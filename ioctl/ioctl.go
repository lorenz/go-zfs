// Package ioctl provides a pure-Go low-level wrapper around ZFS's ioctl interface and basic wrappers around common
// ioctls to make them usable from normal Go code.
package ioctl

import (
	"errors"
	"runtime"
	"unsafe"

	"git.dolansoft.org/lorenz/go-zfs/nvlist"
	"golang.org/x/sys/unix"
)

// NvlistIoctl issues a low-level ioctl syscall with only some common wrappers. All unsafety is contained in here.
func NvlistIoctl(fd uintptr, ioctl Ioctl, name string, cmd *Cmd, request interface{}, response interface{}, config interface{}) error {
	var src []byte
	var configRaw []byte
	var err error
	if request != nil {
		if src, err = nvlist.Marshal(request); err != nil {
			return err
		}
	}
	dst := make([]byte, 8*1024)
	for {
		// This is necessary as some ioctl handlers modify the command buffer even though they
		// later return ENOMEM and we retry the call.
		privateCmd := *cmd
		// WARNING: Here be dragons! This is completely outside of Go's safety net and uses various
		// criticial runtime workarounds to make sure that memory is safely handled
		if response != nil {
			privateCmd.Nvlist_dst = uint64(uintptr(unsafe.Pointer(&dst[0])))
			privateCmd.Nvlist_dst_size = uint64(len(dst))
		}
		if request != nil {
			privateCmd.Nvlist_src = uint64(uintptr(unsafe.Pointer(&src[0])))
			privateCmd.Nvlist_src_size = uint64(len(src))
		}
		if config != nil {
			if configRaw, err = nvlist.Marshal(config); err != nil {
				return err
			}
			privateCmd.Nvlist_conf = uint64(uintptr(unsafe.Pointer(&configRaw[0])))
			privateCmd.Nvlist_conf_size = uint64(len(configRaw))
		}
		stringToDelimitedBuf(name, privateCmd.Name[:])
		_, _, errno := unix.Syscall(unix.SYS_IOCTL, fd, uintptr(ioctl), uintptr(unsafe.Pointer(&privateCmd)))
		runtime.KeepAlive(src)
		runtime.KeepAlive(dst)
		runtime.KeepAlive(privateCmd)
		runtime.KeepAlive(configRaw)
		if errno == unix.ENOMEM {
			if len(dst) >= 16*1024*1024 {
				return errors.New("return buffer is bigger than 16MiB, something probably went wrong")
			}
			dst = make([]byte, len(dst)*8)
			continue
		}
		*cmd = privateCmd
		if errno != 0 {
			return errno
		}
		break
	}
	if response != nil {
		return nvlist.Unmarshal(dst, response)
	}
	return nil
}
