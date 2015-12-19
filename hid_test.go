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
		t.Logf("%v %q %q %v %v\n", i, di.Name, di.Attr.SerialNo, di.Caps.InputLen, di.Caps.OutputLen)
	}
}

func TestRead(t *testing.T) {
	v, err := FindVendorDevices(0x54C, 0x5C4) // DualShock 4
	if err != nil {
		t.Fatal(err)
	}
	if len(v) == 0 {
		return
	}
	d, err := Open(v[0].Name)
	if err != nil {
		t.Fatal(err)
	}
	di, err := d.DeviceInfo()
	if err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, di.Caps.InputLen)
	for i := 0; i < 10; i++ {
		n, err := d.Read(buf)
		if err != nil {
			t.Error(err)
			continue
		}
		t.Logf("%v %v %+v\n", i, n, buf[:n])
	}
}
