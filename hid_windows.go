// needed for endianness in DeviceInterfatceDetailData
// +build windows,386 windows,amd64

package hid

import (
	"fmt"
	"os"
	"runtime"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

var (
	hidClassGuid GUID

	cbSizeGetDeviceInterfaceDetail uint32
)

func init() {
	HidD_GetHidGuid(&hidClassGuid)

	if runtime.GOARCH == "386" {
		cbSizeGetDeviceInterfaceDetail = 6
	} else {
		cbSizeGetDeviceInterfaceDetail = 8
	}
}

func findDevices() ([]string, error) {
	dis, err := SetupDiGetClassDevs(&hidClassGuid, nil, 0, DIGCF_PRESENT|DIGCF_DEVICEINTERFACE)
	if err != nil {
		return nil, err
	}
	defer SetupDiDestroyDeviceInfoList(dis)

	var idata SP_DEVINFO_DATA
	idata.cbSize = uint32(unsafe.Sizeof(idata))

	var edata SP_DEVICE_INTERFACE_DATA
	edata.cbSize = uint32(unsafe.Sizeof(edata))

	var v []string
	for i := uint32(0); SetupDiEnumDeviceInfo(dis, i, &idata) == nil; i++ {
		for j := uint32(0); SetupDiEnumDeviceInterfaces(dis, &idata, &hidClassGuid, j, &edata) == nil; j++ {

			p, err := getDevicePath(dis, &edata, nil)
			if err != nil {
				return nil, err
			}

			v = append(v, p)
		}
	}
	return v, nil
}

func getDevicePath(dis HDEVINFO, edata *SP_DEVICE_INTERFACE_DATA,
	devInfData *SP_DEVINFO_DATA) (detail string, err error) {

	var bufsize uint32

	// this call seems to return an insufficient buffer error while querying the buffer size
	if err := SetupDiGetDeviceInterfaceDetail(dis, edata, nil, 0, &bufsize, nil); err != nil && bufsize == 0 {
		return "", err
	}

	buf := make([]uint16, bufsize+4)
	buf[0] = uint16(cbSizeGetDeviceInterfaceDetail)

	if err := SetupDiGetDeviceInterfaceDetail(dis, edata, &buf[0], bufsize, nil, devInfData); err != nil {
		return "", err
	}

	const firstChar = 2
	l := firstChar
	for l < len(buf) && buf[l] != 0 {
		l++
	}

	return string(utf16.Decode(buf[firstChar:l])), nil
}

func getBusReportedDeviceDescription(dis HDEVINFO, devInfoData *SP_DEVINFO_DATA) (string, error) {
	var propt, size uint32

	buf := make([]byte, 1024)

	run := true
	for run {
		err := SetupDiGetDeviceProperty(dis, devInfoData, &busReportedDeviceDesc,
			&propt, &buf[0], uint32(len(buf)), &size, 0)
		switch {
		case size > uint32(len(buf)):
			buf = make([]byte, size+16)
		case err != nil:
			return "", err
		default:
			run = false
		}
	}

	return utf16BytesToString(buf), nil
}

func getRegistryDeviceDescription(dis HDEVINFO, devInfoData *SP_DEVINFO_DATA) (string, error) {
	var propt, size uint32

	buf := make([]byte, 1024)

	run := true
	for run {
		err := SetupDiGetDeviceRegistryProperty(dis, devInfoData, SPDRP_DEVICEDESC,
			&propt, &buf[0], uint32(len(buf)), &size)
		switch {
		case size > uint32(len(buf)):
			buf = make([]byte, size+16)
		case err != nil:
			return "", err
		default:
			run = false
		}
	}
	return utf16BytesToString(buf), nil
}

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

	var attr HIDD_ATTRIBUTES
	attr.size = uint32(unsafe.Sizeof(attr))
	if err := HidD_GetAttributes(h, &attr); err != nil {
		return err
	}

	d.Attr = &Attr{
		attr.VendorID,
		attr.ProductID,
		attr.VersionNumber,
	}

	var prepd uintptr
	if err := HidD_GetParsedData(h, &prepd); err != nil {
		return err
	}
	defer HidD_FreePreparsedData(prepd)

	var caps HIDP_CAPS
	if errc := HidP_GetCaps(prepd, &caps); errc != HIDP_STATUS_SUCCESS {
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

func utf16BytesToString(p []byte) string {
	u := make([]uint16, len(p)/2)
	l := 0
	for i := range u {
		c := uint16(p[i*2]) + uint16(p[i*2+1])<<8
		u[i] = c
		if c != 0 {
			l = i + 1
		}
	}
	return string(utf16.Decode(u[:l]))
}

type HWND uintptr
type HDEVINFO uintptr

const (
	invalidHDEVINFO = ^HDEVINFO(0)
)

const (
	DIGCF_PRESENT         = 0x2
	DIGCF_ALLCLASSES      = 0x4
	DIGCF_DEVICEINTERFACE = 0x10
)

type GUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

var busReportedDeviceDesc = DEVPROPKEY{
	GUID{0x540b947e, 0x8b40, 0x45bc, [8]byte{0xa8, 0xa2, 0x6a, 0x0b, 0x89, 0x4c, 0xbd, 0xa2}},
	4,
}

type SP_DEVINFO_DATA struct {
	cbSize    uint32
	ClassGuid GUID
	DevInst   uint32
	Reserved  uintptr
}

type SP_DEVICE_INTERFACE_DATA struct {
	cbSize             uint32
	InterfaceClassGuid GUID
	Flags              uint32
	Reserved           uintptr
}

type SP_DEVICE_INTERFACE_DETAIL_DATA struct {
	cbSize     uint32 // should be set to 6 on 386, 8 on amd64
	DevicePath [256]uint16
}

type DEVPROPKEY struct {
	fmtid GUID
	pid   uint32
}

type HIDD_ATTRIBUTES struct {
	size          uint32
	VendorID      uint16
	ProductID     uint16
	VersionNumber uint16
}

type HIDP_CAPS struct {
	Usage                     uint16
	UsagePage                 uint16
	InputReportByteLength     uint16
	OutputReportByteLength    uint16
	FeatureReportByteLength   uint16
	Reserved                  [17]uint16
	NumberLinkCollectionNodes uint16
	NumberInputButtonCaps     uint16
	NumberInputValueCaps      uint16
	NumberInputDataIndices    uint16
	NumberOutputButtonCaps    uint16
	NumberOutputValueCaps     uint16
	NumberOutputDataIndices   uint16
	NumberFeatureButtonCaps   uint16
	NumberFeatureValueCaps    uint16
	NumberFeatureDataIndices  uint16
}

const (
	SPDRP_DEVICEDESC = 0

	HIDP_STATUS_SUCCESS = 0x110000
)

//go:generate go run $GOROOT/src/syscall/mksyscall_windows.go -output zhid_windows.go hid_windows.go

//sys SetupDiGetClassDevs(classGuid *GUID, enumerator *uint16, hwndParent HWND, flags uint32) (handle HDEVINFO, err error) [failretval==invalidHDEVINFO] = setupapi.SetupDiGetClassDevsW
//sys SetupDiEnumDeviceInfo(devInfoSet HDEVINFO, memberIndex uint32, devInfoData *SP_DEVINFO_DATA) (err error) = setupapi.SetupDiEnumDeviceInfo
//sys SetupDiEnumDeviceInterfaces(devInfoSet HDEVINFO, devInfoData *SP_DEVINFO_DATA, intfClassGuid *GUID, memberIndex uint32, devIntfData *SP_DEVICE_INTERFACE_DATA) (err error) = setupapi.SetupDiEnumDeviceInterfaces
//sys SetupDiDestroyDeviceInfoList(devInfoSet HDEVINFO) (err error) = setupapi.SetupDiDestroyDeviceInfoList
//sys SetupDiGetDeviceInterfaceDetail(devInfoSet HDEVINFO, dintfdata *SP_DEVICE_INTERFACE_DATA, detail *uint16, detailSize uint32, reqsize *uint32, devInfData *SP_DEVINFO_DATA) (err error) = setupapi.SetupDiGetDeviceInterfaceDetailW
//sys SetupDiGetDeviceProperty(devInfoSet HDEVINFO, devInfoData *SP_DEVINFO_DATA, propKey *DEVPROPKEY, propType *uint32, propBuf *byte, propBufSize uint32, reqsize *uint32, flags uint32) (err error) = setupapi.SetupDiGetDevicePropertyW
//sys SetupDiGetDeviceRegistryProperty(devInfoSet HDEVINFO, devInfoData *SP_DEVINFO_DATA, prop uint32, propRegDataType *uint32, propBuf *byte, propBufSize uint32, reqsize *uint32) (err error) = setupapi.SetupDiGetDeviceRegistryPropertyW

//sys HidD_GetHidGuid(hidGuid *GUID) = hid.HidD_GetHidGuid
//sys HidD_GetAttributes(h syscall.Handle, a *HIDD_ATTRIBUTES) (err error) = hid.HidD_GetAttributes
//sys HidD_GetParsedData(h syscall.Handle, preparsedData *uintptr) (err error) = hid.HidD_GetPreparsedData
//sys HidD_FreePreparsedData(preparsedData uintptr) (err error) = hid.HidD_FreePreparsedData
//sys HidP_GetCaps(preparsedData uintptr, caps *HIDP_CAPS) (errCode uint32) = hid.HidP_GetCaps
