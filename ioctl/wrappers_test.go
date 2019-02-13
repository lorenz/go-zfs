package ioctl

import (
	"errors"
	"io"
	"os"
	"testing"
)

func TestSequence(t *testing.T) {
	// Clean everything out
	Destroy("test1/test5@snap1", ObjectTypeAny, false)
	Destroy("test1/test5@snap2", ObjectTypeAny, false)
	Destroy("test1/test6@snap1", ObjectTypeAny, false)
	Destroy("test1/test5", ObjectTypeAny, false)
	Destroy("test1/test6", ObjectTypeAny, false)
	Destroy("test1/test7", ObjectTypeAny, false)

	if err := Create("test1/test5", ObjectTypeZFS, nil); err != nil {
		t.Fatal(err)
	}
	if err := Create("test1/test7", ObjectTypeZFS, nil); err != nil {
		t.Error(err)
	}
	if err := Rename("test1/test7", "test1/test6", false); err != nil {
		t.Error(err)
	}
	if err := Snapshot([]string{"test1/test5@snap1", "test1/test6@snap1"}, "test1", nil); err != nil {
		t.Error(err)
	}
	n, err := SendSpace("test1/test5@snap1", SendSpaceOptions{Compress: true})
	if err != nil {
		t.Error(err)
	}
	if n == 0 {
		t.Error(errors.New("size of snaphsot is 0"))
	}
	if err := Snapshot([]string{"test1/test5@snap2"}, "test1", nil); err != nil {
		t.Error(err)
	}
	n, err = SendSpace("test1/test5@snap2", SendSpaceOptions{From: "test1/test5@snap1"})
	if err != nil {
		t.Error(err)
	}
	if n == 0 {
		t.Error(errors.New("size of snaphsot is 0"))
	}

	r, err := Send("test1/test5@snap2", SendOptions{From: "test1/test5@snap1"})
	if err != nil {
		t.Error(err)
	}
	defer r.Close()
	f, err := os.Create("send.bin")
	if err != nil {
		t.Error(err)
	}
	defer f.Close()
	io.Copy(f, r)

	r, err = Send("test1/test5@nonexistent", SendOptions{})
	if err == nil {
		t.Error("Nonexistent send should immediately return an error")
	}

	if err := Destroy("test1/test5@snap1", ObjectTypeAny, false); err != nil {
		t.Error(err)
	}
	if err := Destroy("test1/test5@snap2", ObjectTypeAny, false); err != nil {
		t.Error(err)
	}
	if err := Destroy("test1/test6@snap2", ObjectTypeAny, false); err != nil {
		t.Error(err)
	}
}

func TestObjsetStats(t *testing.T) {
	_, err := ObjsetStats("test1/test9")
	if err != nil {
		t.Error(err)
	}
}
