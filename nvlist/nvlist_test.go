package nvlist

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"
)

func TestUnmarshal(t *testing.T) {
	data, err := ioutil.ReadFile("test-data.bin")
	if err != nil {
		t.Error(err)
	}
	test := new(interface{})
	if err := Unmarshal(data, test); err != nil {
		t.Error(err)
	}
	res, _ := json.MarshalIndent(test, "", "\t")
	fmt.Println(string(res))
}

func TestMarshal(t *testing.T) {
	data, err := ioutil.ReadFile("test-data.bin")
	if err != nil {
		t.Error(err)
	}
	test := new(interface{})
	if err := Unmarshal(data, test); err != nil {
		t.Error(err)
	}
	out, err := Marshal(test)
	if err != nil {
		t.Error(err)
	}
	ioutil.WriteFile("test-data-out.bin", out, 0644)
}
