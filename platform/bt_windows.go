package platform

import (
	"errors"
	"syscall"
	"unsafe"
)

func DisconnectBluetooth(sno string) error {
	if sno == "" {
		return errors.New("hid/platform: empty serial number")
	}

	addr := serialNoToBtMacAddr(sno)

	var btfrp BLUETOOTH_FIND_RADIO_PARAMS
	btfrp.Size = uint32(unsafe.Sizeof(btfrp))

	var radio syscall.Handle
	search, err := BluetoothFindFirstRadio(&btfrp, &radio)
	if err != nil {
		return err
	}
	defer BluetoothFindRadioClose(search)

	for radio != 0 {
		const IOCTL_BTH_DISCONNECT_DEVICE = 0x41000c
		var bytesReturned uint32
		err := syscall.DeviceIoControl(radio, IOCTL_BTH_DISCONNECT_DEVICE,
			&addr[0], uint32(len(addr)), nil, 0, &bytesReturned, nil)
		if err == nil {
			// success
			return nil
		}
		if err := BluetoothFindNextRadio(search, &radio); err != nil {
			const ERROR_NO_MORE_ITEMS = 259
			if errc, _ := err.(syscall.Errno); errc == ERROR_NO_MORE_ITEMS {
				break
			}
			return err
		}
	}
	return errors.New("hid/platform: device not found")
}

func serialNoToBtMacAddr(s string) []byte {
	buf := make([]byte, 8)
	i, p := 0, 6
	var val byte
	for _, r := range s {
		if i == 2 {
			i = 0
			// skip separator, if any
			if r == ':' {
				continue
			}
		}

		var digit byte
		switch {
		case '0' <= r && r <= '9':
			digit = byte(r - '0')
		case 'a' <= r && r <= 'f':
			digit = byte(r-'a') + 10
		case 'A' <= r && r <= 'F':
			digit = byte(r-'A') + 10
		}
		val = val<<4 | digit
		i++

		if i == 2 {
			p--
			buf[p] = val
			if p == 0 {
				break
			}
			val = 0
		}
	}
	return buf
}

type BLUETOOTH_FIND_RADIO_PARAMS struct {
	Size uint32
}

var (
	modbt = syscall.NewLazyDLL("bthprops.cpl")

	procBluetoothFindFirstRadio = modbt.NewProc("BluetoothFindFirstRadio")
	procBluetoothFindRadioClose = modbt.NewProc("BluetoothFindRadioClose")
	procBluetoothFindNextRadio  = modbt.NewProc("BluetoothFindNextRadio")
)

func BluetoothFindFirstRadio(btfrp *BLUETOOTH_FIND_RADIO_PARAMS, radio *syscall.Handle) (handle syscall.Handle, err error) {
	r0, _, e1 := syscall.Syscall(procBluetoothFindFirstRadio.Addr(), 2, uintptr(unsafe.Pointer(btfrp)), uintptr(unsafe.Pointer(radio)), 0)
	handle = syscall.Handle(r0)
	if handle == 0 {
		if e1 != 0 {
			err = error(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func BluetoothFindRadioClose(handle syscall.Handle) (err error) {
	r0, _, e1 := syscall.Syscall(procBluetoothFindRadioClose.Addr(), 1, uintptr(handle), 0, 0)
	if r0 == 0 {
		if e1 != 0 {
			err = error(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func BluetoothFindNextRadio(handle syscall.Handle, radio *syscall.Handle) (err error) {
	r0, _, e1 := syscall.Syscall(procBluetoothFindNextRadio.Addr(), 2, uintptr(handle), uintptr(unsafe.Pointer(radio)), 0)
	if r0 == 0 {
		if e1 != 0 {
			err = error(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}
