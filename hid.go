package hid

type Device struct {
	Path string
	Desc string

	Attr *Attr
	Caps *Caps
}

type Attr struct {
	VendorId  uint16
	ProductId uint16
	Version   uint16
}

type Caps struct {
	Usage                   uint16
	UsagePage               uint16
	InputReportByteLength   uint16
	OutputReportByteLength  uint16
	FeatureReportByteLength uint16

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

func FindDevices() ([]Device, error) {
	return findDevices()
}
