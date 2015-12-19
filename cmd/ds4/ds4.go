package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/tajtiattila/hid"
)

var serialno string

func main() {
	flag.StringVar(&serialno, "-sno", "", "Device serial number to use")
	flag.Parse()

	var dlist []*hid.DeviceInfo
	var err error
	if serialno != "" {
		dlist, err = hid.SerialNo(serialno)
	} else {
		dlist, err = hid.VendorDevices(0x54C, 0x5C4) // DualShock 4
	}
	if err != nil {
		log.Println(err)
		return
	}
	if len(dlist) == 0 {
		log.Println("device not found")
		return
	}
	fmt.Println(len(dlist), "device(s) found")
	di := dlist[0]
	fmt.Print("i/o report length: ", di.Caps.InputLen, "/", di.Caps.OutputLen, "\n")

	ibuf := make([]byte, di.Caps.InputLen)
	obuf := make([]byte, di.Caps.OutputLen)

	d, err := hid.Open(di.Name)
	if err != nil {
		log.Println(err)
		return
	}
	defer d.Close()

	obuf[0] = 0x11
	obuf[1] = 0x80
	obuf[3] = 0xff
	obuf[6] = 0    //fast motor
	obuf[7] = 0    //slow motor
	obuf[8] = 0xff //red
	obuf[9] = 0x88 //green
	obuf[10] = 0   //blue
	obuf[11] = 0   //flash on duration
	obuf[12] = 0   //flash off duration
	err = d.SetOutputReport(obuf)
	if err != nil {
		log.Println(err)
		return
	}

	n, err := d.Read(ibuf)
	if n != 0 {
		dumpbytes(ibuf[:n], 0)
		fmt.Println()
	}
	if err != nil {
		log.Println(err)
		return
	}

	for {
		_, err := d.Read(ibuf)
		if err != nil {
			log.Println(err)
			return
		}

		var s RawState
		if err := s.Decode(ibuf); err != nil {
			continue
		}
		fmt.Print("\r", s.String())
		dumpbytes(ibuf[28:], 32)
	}
}

var hexbytes = []byte("0123456789abcdef")

func dumpbytes(p []byte, max int) {
	if max != 0 && len(p) > max {
		p = p[:max]
	}
	var buf bytes.Buffer
	for i, b := range p {
		if i%4 == 0 {
			if i != 0 {
				buf.WriteByte('-')
			}
		} else {
			buf.WriteByte(' ')
		}
		buf.WriteByte(hexbytes[b>>4])
		buf.WriteByte(hexbytes[b&15])
	}
	buf.WriteByte(27)
	buf.WriteByte('[')
	buf.WriteByte('J')
	os.Stdout.Write(buf.Bytes())
}

// constants for the Button field of D4State
const (
	Dpad = 0xf // 8: off, 0-7: clockwise from 12 o'clock

	Square   = 1 << 4
	Cross    = 1 << 5
	Circle   = 1 << 6
	Triangle = 1 << 7

	L1      = 1 << 8
	R1      = 1 << 9
	L2      = 1 << 10
	R2      = 1 << 11
	Options = 1 << 12
	Share   = 1 << 13
	L3      = 1 << 14
	R3      = 1 << 15

	PS    = 1 << 16
	Touch = 1 << 17
)

type RawState struct {
	// sticks
	LX, LY, RX, RY byte

	// triggers
	L2, R2 byte

	// buttons
	Button uint32

	// accelerometer
	XAcc, YAcc, ZAcc uint16

	// gyro
	XGyro, YGyro, ZGyro uint16
}

var dpadStr = []string{
	"N ", "NE", "E ", "SE", "S ", "SW", "W ", "NW",
	"--", "--", "--", "--", "--", "--", "--", "--",
}

func (s *RawState) String() string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "L(%+4d %+4d) R(%+4d %+4d) ",
		int(s.LX)-128, int(s.LY)-128,
		int(s.RX)-128, int(s.RY)-128)
	buf.WriteString(dpadStr[s.Button&0xF])
	buf.WriteByte(' ')
	for i := uint(0); i < 14; i++ {
		if s.Button&(1<<(17-i)) != 0 {
			buf.WriteByte('X')
		} else {
			buf.WriteByte('.')
		}
	}
	buf.WriteByte(' ')
	x := int(int16(s.XGyro)) / 64
	y := int(int16(s.YGyro)) / 64
	z := int(int16(s.ZGyro)) / 64
	fmt.Fprintf(&buf, "G(%+4d %+4d %+4d)", x, y, z)
	return buf.String()
}

func (s *RawState) Decode(p []byte) error {
	if p[0] != 0x11 {
		// should we support 0x01?
		return fmt.Errorf("Data packet %#x is not supported", p[0])
	}

	p = p[2:]
	if len(p) < 26 {
		return fmt.Errorf("short packet")
	}

	s.LX, s.LY = p[1], p[2]
	s.RX, s.RY = p[3], p[4]
	s.Button = uint32(p[5]) | uint32(p[6])<<8 | uint32(p[7])<<16
	s.L2, s.R2 = p[8], p[9]

	s.XAcc, s.YAcc, s.ZAcc = u16triplet(p[14:20])
	s.XGyro, s.YGyro, s.ZGyro = u16triplet(p[20:26])
	return nil
}

func u16triplet(p []byte) (x, y, z uint16) {
	x = uint16(p[0])<<8 | uint16(p[1])
	y = uint16(p[2])<<8 | uint16(p[3])
	z = uint16(p[4])<<8 | uint16(p[5])
	return
}
