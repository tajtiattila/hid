package hid

import "testing"

func TestFindDevices(t *testing.T) {
	ds, err := FindDevices()
	if err != nil {
		t.Fatal(err)
	}
	for i, d := range ds {
		t.Logf("%v %q %q", i, d.Path, d.Desc)
	}
}
