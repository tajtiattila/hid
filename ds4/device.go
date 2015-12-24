package ds4

import (
	"time"

	"github.com/tajtiattila/hid"
)

type Device struct {
	*hid.Device

	// bluetooth
	bt bool

	ibuf []byte
	obuf []byte
}

type Error struct {
	Msg string

	// error cause
	Err error
}

func (e *Error) Error() string { return e.Msg + ": " + e.Err.Error() }

const (
	BT_OUTPUT_REPORT_LENGTH = 78
	BT_INPUT_REPORT_LENGTH  = 547
)

func Open(name string) (*Device, error) {
	d, err := hid.Open(name)
	if err != nil {
		return nil, &Error{"ds4.Open", err}
	}
	di, err := d.DeviceInfo()
	if err != nil {
		d.Close()
		return nil, &Error{"ds4.DeviceInfo", err}
	}
	x := &Device{
		Device: d,

		bt:   di.Caps.InputLen > 64,
		ibuf: make([]byte, di.Caps.InputLen),
		obuf: make([]byte, di.Caps.OutputLen),
	}
	if err = x.SetOutput(&Output{}); err != nil {
		d.Close()
		return nil, &Error{"ds4.SetOutput", err}
	}
	return x, nil
}

func (d *Device) ReadState(s *State) error {
	_, err := d.Device.Read(d.ibuf)
	if err != nil {
		return err
	}
	return s.Decode(d.ibuf)
}

func (d *Device) SetColor(c Color) error {
	return d.SetOutput(&Output{Led: c})
}

func (d *Device) SetFlashColor(c Color, on, off time.Duration) error {
	return d.SetOutput(&Output{Led: c, On: on, Off: off})
}

func (d *Device) SetOutput(o *Output) (err error) {
	if d.bt {
		d.obuf[0] = 0x11
		d.obuf[1] = 0x80
		d.obuf[3] = 0xff
		d.obuf[6] = o.Light     //fast motor
		d.obuf[7] = o.Heavy     //slow motor
		d.obuf[8] = o.Led.R     //red
		d.obuf[9] = o.Led.G     //green
		d.obuf[10] = o.Led.B    //blue
		d.obuf[11] = dur(o.On)  //flash on duration
		d.obuf[12] = dur(o.Off) //flash off duration

		err = d.SetOutputReport(d.obuf)
	} else {
		d.obuf[0] = 0x05
		d.obuf[1] = 0xff
		d.obuf[4] = o.Light     //fast motor
		d.obuf[5] = o.Heavy     //slow  motor
		d.obuf[6] = o.Led.R     //red
		d.obuf[7] = o.Led.G     //green
		d.obuf[8] = o.Led.B     //blue
		d.obuf[9] = dur(o.On)   //flash on duration
		d.obuf[10] = dur(o.Off) //flash off duration

		_, err = d.Write(d.obuf)
	}
	return err
}

type Output struct {
	// Rumble motors
	Light, Heavy byte

	// Led color
	Led Color

	// Flash durations
	On, Off time.Duration
}

type Color struct {
	R, G, B byte
}

func dur(d time.Duration) byte {
	if d <= 0 {
		return 0
	}
	n := int64(d/time.Millisecond) / 10
	if n == 0 {
		return 1
	}
	if n <= 255 {
		return byte(n)
	}
	return 255
}
