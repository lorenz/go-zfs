package ioctl

import (
	"errors"
	"io"
	"os"
	"testing"
)

func TestSequence(t *testing.T) {
	fileLocation := "/dev/shm/test.img"
	f, err := os.Create(fileLocation)
	if err != nil {
		t.Fatal(err)
	}

	if err := f.Truncate(1e9); err != nil { // 1GiB
		t.Fatal(err)
	}
	f.Close()

	defer PoolDestroy("tp1")
	defer os.Remove(fileLocation)

	err = PoolCreate("tp1", map[string]uint64{}, VDev{
		Type: "root",
		Children: []VDev{
			VDev{
				Type: "file",
				Path: fileLocation,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := Create("tp1/test5", ObjectTypeZFS, nil); err != nil {
		t.Fatal(err)
	}
	if err := Create("tp1/test7", ObjectTypeZFS, nil); err != nil {
		t.Error(err)
	}
	if err := Rename("tp1/test7", "tp1/test6", false); err != nil {
		t.Error(err)
	}
	if err := Snapshot([]string{"tp1/test5@snap1", "tp1/test6@snap1"}, "tp1", nil); err != nil {
		t.Error(err)
	}
	n, err := SendSpace("tp1/test5@snap1", SendSpaceOptions{Compress: true})
	if err != nil {
		t.Error(err)
	}
	if n == 0 {
		t.Error(errors.New("size of snaphsot is 0"))
	}
	if err := Snapshot([]string{"tp1/test5@snap2"}, "tp1", nil); err != nil {
		t.Error(err)
	}
	n, err = SendSpace("tp1/test5@snap2", SendSpaceOptions{From: "tp1/test5@snap1"})
	if err != nil {
		t.Error(err)
	}
	if n == 0 {
		t.Error(errors.New("size of snaphsot is 0"))
	}

	r, err := Send("tp1/test5@snap2", SendOptions{From: "tp1/test5@snap1"})
	if err != nil {
		t.Error(err)
	}
	defer r.Close()
	f, err = os.Create("send.bin")
	if err != nil {
		t.Error(err)
	}
	defer f.Close()
	io.Copy(f, r)

	r, err = Send("tp1/test5@nonexistent", SendOptions{})
	if err == nil {
		t.Error("Nonexistent send should immediately return an error")
	}

	if err := Destroy("tp1/test5@snap1", ObjectTypeAny, false); err != nil {
		t.Error(err)
	}
	if err := Destroy("tp1/test5@snap2", ObjectTypeAny, false); err != nil {
		t.Error(err)
	}
	if err := Destroy("tp1/test6@snap2", ObjectTypeAny, false); err != nil {
		t.Error(err)
	}
	if err := PoolDestroy("tp1"); err != nil {
		t.Error(err)
	}
}
