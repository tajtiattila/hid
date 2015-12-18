package hid

type Device struct {
	Path string
	Desc string
}

func FindDevices() ([]Device, error) {
	return findDevices()
}
