// +build windows

package hid

import (
	"fmt"
	"os"
	"syscall"

	"github.com/tajtiattila/hid/platform"
)

// IsAccess checks if the err is an access error, meaning
// the device is currently unavailable because of system
// permissions or the device was opened with exclusive access.
func IsAccess(err error) bool {
	if os.IsPermission(err) {
		return true
	}
	if errc, _ := err.(syscall.Errno); errc == 32 {
		// ERROR_SHARING_VIOLATION
		return true
	}
	return false
}

func (d *Device) DeviceInfo() (*DeviceInfo, error) {
	i := &DeviceInfo{Name: d.Name()}
	err := statHandle(syscall.Handle(d.Fd()), i)
	if err != nil {
		return nil, err
	}
	return i, nil
}

func statHandle(h syscall.Handle, d *DeviceInfo) error {

	var attr platform.HIDD_ATTRIBUTES
	if err := platform.HidD_GetAttributes(h, &attr); err != nil {
		return err
	}

	d.Attr = &Attr{
		attr.VendorID,
		attr.ProductID,
		attr.VersionNumber,
	}

	var prepd uintptr
	if err := platform.HidD_GetParsedData(h, &prepd); err != nil {
		return err
	}
	defer platform.HidD_FreePreparsedData(prepd)

	var caps platform.HIDP_CAPS
	if errc := platform.HidP_GetCaps(prepd, &caps); errc != platform.HIDP_STATUS_SUCCESS {
		return fmt.Errorf("hid.GetCaps() failed with error code %#x", errc)
	}

	d.Caps = &Caps{
		Usage:     caps.Usage,
		UsagePage: caps.UsagePage,

		InputLen:   int(caps.InputReportByteLength),
		OutputLen:  int(caps.OutputReportByteLength),
		FeatureLen: int(caps.FeatureReportByteLength),

		NumLinkCollectionNodes: int(caps.NumberLinkCollectionNodes),
		NumInputButtonCaps:     int(caps.NumberInputButtonCaps),
		NumInputValueCaps:      int(caps.NumberInputValueCaps),
		NumInputDataIndices:    int(caps.NumberInputDataIndices),
		NumOutputButtonCaps:    int(caps.NumberOutputButtonCaps),
		NumOutputValueCaps:     int(caps.NumberOutputValueCaps),
		NumOutputDataIndices:   int(caps.NumberOutputDataIndices),
		NumFeatureButtonCaps:   int(caps.NumberFeatureButtonCaps),
		NumFeatureValueCaps:    int(caps.NumberFeatureValueCaps),
		NumFeatureDataIndices:  int(caps.NumberFeatureDataIndices),
	}

	return nil
}
