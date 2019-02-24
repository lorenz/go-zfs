package ioctl

import (
	"runtime"
	"unsafe"

	"git.dolansoft.org/lorenz/go-zfs/nvlist"
	"golang.org/x/sys/unix"
)

func NvlistIoctl(fd uintptr, ioctl Ioctl, name string, cmd *Cmd, request interface{}, response interface{}, config interface{}) error {
	var src []byte
	var configRaw []byte
	var err error
	if request != nil {
		if src, err = nvlist.Marshal(request); err != nil {
			return err
		}
		//ioutil.WriteFile("ioctl-req.bin", src, 0644)
	}
	// WARNING: Here be dragons! This is completely outside of Go's safety net and uses various
	// criticial runtime workarounds to make sure that memory is safely handled
	dst := make([]byte, 4096)
	if response != nil {
		cmd.Nvlist_dst = uint64(uintptr(unsafe.Pointer(&dst[0])))
		cmd.Nvlist_dst_size = uint64(len(dst))
	}
	if request != nil {
		cmd.Nvlist_src = uint64(uintptr(unsafe.Pointer(&src[0])))
		cmd.Nvlist_src_size = uint64(len(src))
	}
	if config != nil {
		if configRaw, err = nvlist.Marshal(config); err != nil {
			return err
		}
		cmd.Nvlist_conf = uint64(uintptr(unsafe.Pointer(&configRaw[0])))
		cmd.Nvlist_conf_size = uint64(len(configRaw))
	}
	stringToDelimitedBuf(name, cmd.Name[:])
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, fd, uintptr(ioctl), uintptr(unsafe.Pointer(cmd)))
	runtime.KeepAlive(src)
	runtime.KeepAlive(dst)
	runtime.KeepAlive(cmd)
	runtime.KeepAlive(configRaw)
	if errno != 0 {
		return errno
	}
	//ioutil.WriteFile("ioctl-res.bin", dst, 0644)
	if response != nil {
		return nvlist.Unmarshal(dst, response)
	}
	return nil
}
