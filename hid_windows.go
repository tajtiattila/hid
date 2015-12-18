// needed for endianness in DeviceInterfatceDetailData
// +build windows,386 windows,amd64

package hid

import (
	"runtime"
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

func findDevices() ([]Device, error) {
	dis, err := SetupDiGetClassDevs(&hidClassGuid, nil, 0, DIGCF_PRESENT|DIGCF_DEVICEINTERFACE)
	if err != nil {
		panic(1)
		return nil, err
	}

	var idata SP_DEVINFO_DATA
	idata.cbSize = uint32(unsafe.Sizeof(idata))

	var edata SP_DEVICE_INTERFACE_DATA
	edata.cbSize = uint32(unsafe.Sizeof(edata))

	var v []Device
	for i := uint32(0); SetupDiEnumDeviceInfo(dis, i, &idata); i++ {
		for j := uint32(0); SetupDiEnumDeviceInterfaces(dis, &idata, &hidClassGuid, j, &edata); j++ {

			p, err := GetDeviceInterfaceDetail(dis, &edata, nil)
			if err != nil {
				return nil, err
			}
			v = append(v, Device{Path: p})
		}
	}
	return v, nil
}

func GetDeviceInterfaceDetail(
	dis HDEVINFO,
	edata *SP_DEVICE_INTERFACE_DATA,
	devInfData *SP_DEVINFO_DATA) (detail string, err error) {

	var bufsize uint32

	// this call seems to return an insufficient buffer error while querying the buffer size
	if err := SetupDiGetDeviceInterfaceDetail(dis, edata, nil, 0, &bufsize, nil); err != nil && bufsize == 0 {
		return "", err
	}

	buf := make([]uint16, cbSizeGetDeviceInterfaceDetail/2+bufsize)
	buf[0] = uint16(cbSizeGetDeviceInterfaceDetail)

	if err := SetupDiGetDeviceInterfaceDetail(dis, edata, &buf[0], bufsize, nil, devInfData); err != nil {
		return "", err
	}

	return string(utf16.Decode(buf[cbSizeGetDeviceInterfaceDetail/2:])), nil
}

type HWND uintptr
type HDEVINFO uintptr

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

//go:generate go run $GOROOT/src/syscall/mksyscall_windows.go -output zhid_windows.go hid_windows.go

const invalidHDEVINFO = ^HDEVINFO(0)

//sys SetupDiGetClassDevs(classGuid *GUID, enumerator *uint16, hwndParent HWND, flags uint32) (handle HDEVINFO, err error) [failretval==invalidHDEVINFO] = setupapi.SetupDiGetClassDevsW
//sys SetupDiEnumDeviceInfo(devInfoSet HDEVINFO, memberIndex uint32, devInfoData *SP_DEVINFO_DATA) (ok bool) = setupapi.SetupDiEnumDeviceInfo
//sys SetupDiEnumDeviceInterfaces(devInfoSet HDEVINFO, devInfoData *SP_DEVINFO_DATA, intfClassGuid *GUID, memberIndex uint32, devIntfData *SP_DEVICE_INTERFACE_DATA) (ok bool) = setupapi.SetupDiEnumDeviceInterfaces
//sys SetupDiDestroyDeviceInfoList(devInfoSet HDEVINFO) (err error) = setupapi.SetupDiDestroyDeviceInfoList
//sys SetupDiGetDeviceInterfaceDetail(devInfoSet HDEVINFO, dintfdata *SP_DEVICE_INTERFACE_DATA, detail *uint16, detailSize uint32, reqsize *uint32, devInfData *SP_DEVINFO_DATA) (err error) = setupapi.SetupDiGetDeviceInterfaceDetailW

//sys HidD_GetHidGuid(hidGuid *GUID) = hid.HidD_GetHidGuid
