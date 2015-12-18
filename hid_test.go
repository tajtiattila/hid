package hid

import "testing"

func TestFindDevices(t *testing.T) {
	ds, err := FindDevices()
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range ds {
		t.Log(d.Path)
	}
}
