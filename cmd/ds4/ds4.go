package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"time"

	"github.com/tajtiattila/hid"
	"github.com/tajtiattila/hid/ds4"
	"github.com/tajtiattila/hid/ds4/ds4util"
)

var (
	serialno string
	touch    bool
	verbose  bool

	alpha  float64
	movavg time.Duration
)

func main() {
	flag.StringVar(&serialno, "sno", "", "Device serial number to use")
	flag.BoolVar(&touch, "touch", false, "Touch test")
	flag.BoolVar(&verbose, "v", false, "Verbose output")
	flag.Float64Var(&alpha, "alpha", 1, "Gyro low pass filter alpha")
	flag.DurationVar(&movavg, "movavg", 0, "Moving average duration")
	flag.Parse()

	var dlist []*hid.DeviceInfo
	var err error
	if serialno != "" {
		dlist, err = hid.SerialNo(serialno)
	} else {
		dlist, err = hid.VendorDevices(0x54C, 0x5C4) // DualShock 4
	}
	if err != nil {
		log.Println("device list:", err)
		return
	}
	if len(dlist) == 0 {
		log.Println("device not found")
		return
	}
	fmt.Println(len(dlist), "device(s) found")
	for i, di := range dlist {
		fmt.Println(i, di.Name, di.Attr.SerialNo)
	}

	di := *dlist[0]
	for _, xdi := range dlist[1:] {
		if xdi.Caps.InputLen > di.Caps.InputLen {
			di = *xdi
		}
	}

	fmt.Print("i/o report length: ", di.Caps.InputLen, "/", di.Caps.OutputLen, "\n")

	ibuf := make([]byte, di.Caps.InputLen)

	d, err := ds4.Open(di.Name)
	if err != nil {
		log.Println("opening device:", err)
		return
	}
	defer d.Close()

	d.SetColor(ds4.Color{0xff, 0x88, 0x00})
	//d.SetFlashColor(ds4.Color{255, 0, 0}, time.Second, time.Second)

	n, err := d.Read(ibuf)
	if n != 0 {
		dumpbytes(ibuf[:n], 0)
		fmt.Println()
	}
	if err != nil {
		log.Println("initial read:", err)
		return
	}

	var s ds4.State
	if err := s.Decode(ibuf); err == nil {
		var charging string
		if s.Battery&0xf0 != 0 {
			charging = " (charging)"
		}
		fmt.Printf("Battery: %v%%%s\n", 10*(s.Battery&0xf), charging)
	}

	ch := make(chan struct{})
	go func() {
		buf := make([]byte, 1)
		os.Stdin.Read(buf)
		close(ch)
	}()

	var f func(p []byte, s *ds4.State)
	if touch {
		tt := NewTouchTest()
		f = func(p []byte, s *ds4.State) {
			tt.Run(s)
		}
	} else {
		f = InputTest
		if alpha != 1 {
			filter = ds4util.NewAlphaFilter(3, alpha)
		}
		if movavg != 0 {
			n := int(movavg/time.Millisecond) * 3
			a := ds4util.NewMovAvg(3, n, movavg)
			if filter != ds4util.Input {
				filter = ds4util.Combine(filter, a)
			} else {
				filter = a
			}
		}
	}

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

		if err := s.Decode(ibuf); err != nil {
			continue
		}

		f(ibuf, &s)
	}
}

var filter ds4util.Filter = ds4util.Input

func InputTest(ibuf []byte, s *ds4.State) {
	fmt.Print("\r")
	const m = 100000
	v := filter.Filter([]int{
		int(s.XGyro) * m,
		int(s.YGyro) * m,
		int(s.ZGyro) * m,
	})
	x, y, z := float64(v[0]), float64(v[1]), float64(v[2])
	r, p := ds4.GyroRollPitch(x, y, z)
	mag := math.Sqrt(x*x + y*y + z*z)
	fmt.Printf("%5.2f %5.2f %5.2f %4.0f %4.0f", x/mag, y/mag, z/mag, r, p)
	/*

		gr, gp, ok := s.GyroRollPitch()
		ls := "ok  "
		if !ok {
			ls = "lock"
		}
		fmt.Printf(" GX(%5.1f %5.1f %s)", gr, gp, ls)
	*/

	//dumpbytes(ibuf, 40)

	// clear to end of buffer
	os.Stdout.Write([]byte{27, '[', 'J'})
}

type TouchTest struct {
	pad     *ds4util.SwipeLogic
	lastpkt byte
	buf     bytes.Buffer
}

func NewTouchTest() *TouchTest {
	t := new(TouchTest)
	t.pad = ds4util.NewSwipeLogic(t)
	return t
}

func (t *TouchTest) Run(s *ds4.State) {
	t.pad.HandleState(s)
	if s.Packet == t.lastpkt {
		if t.buf.Len() != 0 {
			t.buf.WriteTo(os.Stdout)
			t.buf.Reset()
		}
		return
	}
	t.lastpkt = s.Packet
	if verbose {
		fmt.Fprintln(&t.buf, time.Now().Format("15:04:05.000"), s.String())
	}
}

func (t *TouchTest) Swipe(dir, ntouch int) {
	fmt.Println("Swipe:", dir, ntouch)
}

func (t *TouchTest) Touch(x, y int) {
	fmt.Println("Touch:", x, y)
}

func (t *TouchTest) Click(x, y int) {
	fmt.Println("Click:", x, y)
}

func absgyro(raw int16) int {
	v := int(raw)
	if v < 0 {
		v = -v
	}
	return v
}

func flt(raw int16) float64 {
	return float64(raw)
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
