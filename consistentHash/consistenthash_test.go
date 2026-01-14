package consistenthash

import (
	"strconv"
	"testing"
)

func TestConsistenthash(t *testing.T) {
	m := New(3, func(key []byte) uint32 {
		num, error := strconv.Atoi(string(key))
		if error != nil {
			return uint32(1)
		}
		return uint32(num)
	})
	// 虚拟节点：2、4、6、12、14、16、22、24、26
	m.Add("6", "4", "2")
	testCases := map[string]string{
		"2":  "2",
		"11": "2",
		"23": "4",
		"27": "2",
	}

	for k, v := range testCases {
		if m.Get(k) != v {
			t.Errorf("Asking for %s, should have yielded %s", k, v)
		}
	}

	// Adds 8, 18, 28
	m.Add("8")

	// 27 should now map to 8.
	testCases["27"] = "8"

	for k, v := range testCases {
		if m.Get(k) != v {
			t.Errorf("Asking for %s, should have yielded %s", k, v)
		}
	}
}
