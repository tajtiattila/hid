package hid

import "testing"

func TestFindDevices(t *testing.T) {
	v, err := FindDevices()
	if err != nil {
		t.Fatal(err)
	}
	for i, s := range v {
		t.Logf("%v %q", i, s)
	}
}

func TestFindVendorDevices(t *testing.T) {
	v, err := FindVendorDevices(0x54C, 0x5C4) // DualShock 4
	if err != nil {
		t.Fatal(err)
	}
	for i, di := range v {
		t.Logf("%v %q %v %v\n", i, di.Name, di.Caps.InputLen, di.Caps.OutputLen)
	}
}
