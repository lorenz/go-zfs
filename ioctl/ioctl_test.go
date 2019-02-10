package ioctl

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

func TestNvlistIoctl(t *testing.T) {
	zfsHandle, err := os.Open("/dev/zfs")
	if err != nil {
		t.Error(err)
	}
	res := new(interface{})
	if err := NvlistIoctl(zfsHandle.Fd(), ZFS_IOC_DATASET_LIST_NEXT, "test1", make(map[string]interface{}), res); err != nil {
		t.Error(err)
	}
	out, _ := json.MarshalIndent(res, "", "\t")
	fmt.Println(string(out))
}
