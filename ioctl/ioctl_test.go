package ioctl

import (
	"fmt"
	"os"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestNvlistIoctl(t *testing.T) {
	zfsHandle, err := os.Open("/dev/zfs")
	if err != nil {
		t.Error(err)
	}
	res := new(interface{})
	cmd := &Cmd{}
	if err := NvlistIoctl(zfsHandle.Fd(), ZFS_IOC_DATASET_LIST_NEXT, "test1", cmd, nil, res); err != nil {
		t.Error(err)
	}
	var outNameRaw []byte
	for i := 0; i < len(cmd.Name); i++ {
		if cmd.Name[i] == 0x00 {
			break
		}
		outNameRaw = append(outNameRaw, cmd.Name[i])
	}
	fmt.Println(string(outNameRaw))
	spew.Dump(res)
	res = new(interface{})
	cmd2 := &Cmd{Cookie: cmd.Cookie}
	if err := NvlistIoctl(zfsHandle.Fd(), ZFS_IOC_DATASET_LIST_NEXT, "test1", cmd2, nil, res); err != nil {
		t.Error(err)
	}
	var outNameRaw2 []byte
	for i := 0; i < len(cmd2.Name); i++ {
		if cmd2.Name[i] == 0x00 {
			break
		}
		outNameRaw2 = append(outNameRaw2, cmd2.Name[i])
	}
	fmt.Println(string(outNameRaw2))
	spew.Dump(res)
}
