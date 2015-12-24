package hid

import (
	"github.com/tajtiattila/hid/asyncio"
	"github.com/tajtiattila/hid/platform"
)

// Names lists the names of all available HID devices.
func Names() ([]string, error) {
	return platform.FindDevices()
}

// VendorDevices finds accessible devices having the specified vendor and product IDs.
func VendorDevices(vendor uint16, products ...uint16) ([]*DeviceInfo, error) {
	v, err := Names()
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

// SerialNo finds accessible devices having the specified serial number.
func SerialNo(sno string) ([]*DeviceInfo, error) {
	v, err := Names()
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
		if i.Attr.SerialNo == sno {
			vv = append(vv, i)
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
	f, err := asyncio.Open(name)
	if err != nil {
		return nil, newErr("ds4.Open", name, err)
	}
	return &Device{f}, nil
}

// Device is a HID device that statisfies io.ReadWriteCloser.
type Device struct {
	*asyncio.File
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
	SerialNo  string
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

type Error struct {
	Func string
	Path string
	Err  error
}

func (e *Error) Error() string {
	return e.Func + " " + e.Path + ": " + e.Err.Error()
}

func newErr(f, p string, err error) error {
	return &Error{f, p, err}
}
