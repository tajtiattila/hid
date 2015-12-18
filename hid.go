package hid

import (
	"os"

	"github.com/tajtiattila/hid/platform"
)

// FindDevices lists all available HID devices.
func FindDevices() ([]string, error) {
	return platform.FindDevices()
}

// FindVendorDevices finds accessible devices having the specified vendor and product IDs.
func FindVendorDevices(vendor uint16, products ...uint16) ([]*DeviceInfo, error) {
	v, err := FindDevices()
	if err != nil {
		return nil, err
	}
	var vv []*DeviceInfo
	for _, n := range v {
		i, err := Stat(n)
		if err != nil {
			if IsAccess(err) {
				continue
			}
			return nil, err
		}
		if i.Attr.VendorId != vendor {
			continue
		}
		for _, iv := range products {
			if iv == i.Attr.ProductId {
				vv = append(vv, i)
				break
			}
		}
	}
	return vv, nil
}

// Stat returns device info from the specified path.
func Stat(name string) (*DeviceInfo, error) {
	d, err := Open(name)
	if err != nil {
		return nil, err
	}
	defer d.Close()
	return d.DeviceInfo()
}

// Open opens the specified device.
func Open(name string) (*Device, error) {
	f, err := os.OpenFile(name, os.O_RDWR, 0777)
	if err != nil {
		return nil, err
	}
	return &Device{f}, nil
}

// Device is a HID device that statisfies io.ReadWriteCloser.
type Device struct {
	*os.File
}

type DeviceInfo struct {
	Name string

	Attr *Attr
	Caps *Caps
}

type Attr struct {
	VendorId  uint16
	ProductId uint16
	Version   uint16
}

type Caps struct {
	Usage     uint16
	UsagePage uint16

	// Report lengths
	InputLen   int
	OutputLen  int
	FeatureLen int

	NumLinkCollectionNodes int
	NumInputButtonCaps     int
	NumInputValueCaps      int
	NumInputDataIndices    int
	NumOutputButtonCaps    int
	NumOutputValueCaps     int
	NumOutputDataIndices   int
	NumFeatureButtonCaps   int
	NumFeatureValueCaps    int
	NumFeatureDataIndices  int
}
