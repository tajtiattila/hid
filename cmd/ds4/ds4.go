package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"math"
	"os"

	"github.com/tajtiattila/hid"
	"github.com/tajtiattila/hid/ds4"
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

	ch := make(chan struct{})
	go func() {
		buf := make([]byte, 1)
		os.Stdin.Read(buf)
		close(ch)
	}()

	var max int
	for {
		select {
		case <-ch:
			fmt.Println()
			err := d.DisconnectRadio()
			if err != nil {
				log.Println(err)
			}
			return
		default:
		}

		_, err := d.Read(ibuf)
		if err != nil {
			log.Println(err)
			return
		}

		var s ds4.State
		if err := s.Decode(ibuf); err != nil {
			continue
		}
		fmt.Print("\r", s.String())

		x, y, z := absgyro(s.XGyro), absgyro(s.YGyro), absgyro(s.ZGyro)
		if max < x {
			max = x
		}
		if max < y {
			max = y
		}
		if max < z {
			max = z
		}

		xf, yf, zf := flt(s.XGyro), flt(s.YGyro), flt(s.ZGyro)

		//p := ibuf[2:]
		//dumpbytes(p[34:], 32)
		fmt.Printf(" %5d %8.2f", max, math.Sqrt(xf*xf+yf*yf+zf*zf))

		// clear to end of buffer
		os.Stdout.Write([]byte{27, '[', 'J'})
	}
}

func absgyro(raw uint16) int {
	v := int(int16(raw))
	if v < 0 {
		v = -v
	}
	return v
}

func flt(raw uint16) float64 {
	return float64(int16(raw))
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
	os.Stdout.Write(buf.Bytes())
}
