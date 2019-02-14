// +build gofuzz

package nvlist

func Fuzz(data []byte) int {
	out := new(interface{})
	err := Unmarshal(data, &out)
	if err == nil {
		return 1
	}
	return 0
}
