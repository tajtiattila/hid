// Package ds4 accesses
// Sony® PlayStation® Dual Shock 4 controllers.
package ds4

import (
	"bytes"
	"fmt"
)

// constants for the Button field of D4State
const (
	Dpad = 0xf // 8: off, 0-7: clockwise from 12 o'clock

	DpadOff = 1 << 3
	DpadDir = 0x7

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
	Click = 1 << 17 // touchpad
)

// State represents the controller state.
type State struct {
	// sticks
	LX, LY, RX, RY byte

	// triggers
	L2, R2 byte

	// buttons
	Button uint32

	// accelerometer vector
	XAcc, YAcc, ZAcc int16

	// gyroscope vector
	// x axis points left
	// y axis points down
	// z axis points forward
	XGyro, YGyro, ZGyro int16

	// battery
	Battery byte // bit 5: charging, bits: 0-4 battery level percentage/10

	// Packet is a counter incremented whenever there is touch input
	Packet byte

	// Touch holds recognised touch events
	Touch [2]Touch
}

const (
	TouchInactive = 0x80
	TouchIdMask   = 0x7f
)

// Touch represents a single touch.
type Touch struct {
	// bit 7: zero when touch is active
	// bits 0-6: finger id incremented by every new touch
	Id byte

	// 12-bit touch positions
	X, Y int16
}

func (t Touch) Active() bool { return t.Id&TouchInactive == 0 }

var dpadStr = []string{
	"N ", "NE", "E ", "SE", "S ", "SW", "W ", "NW",
	"--", "--", "--", "--", "--", "--", "--", "--",
}

func (s *State) String() string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "L(%+4d %+4d) R(%+4d %+4d) ",
		int(s.LX)-128, int(s.LY)-128,
		int(s.RX)-128, int(s.RY)-128)
	fmt.Fprintf(&buf, " 2(%3d %3d)", s.L2, s.R2)
	buf.WriteString(dpadStr[s.Button&0xF])
	buf.WriteByte(' ')
	for i := uint(0); i < 14; i++ {
		if s.Button&(1<<(17-i)) != 0 {
			buf.WriteByte('X')
		} else {
			buf.WriteByte('.')
		}
	}
	x := int(int16(s.XGyro)) / 64
	y := int(int16(s.YGyro)) / 64
	z := int(int16(s.ZGyro)) / 64
	fmt.Fprintf(&buf, " G(%+4d %+4d %+4d)", x, y, z)
	fmt.Fprintf(&buf, " %02x", s.Battery)
	fmt.Fprintf(&buf, " %02x", s.Packet)
	for i := 0; i < 2; i++ {
		t := s.Touch[i]
		if t.Active() {
			fmt.Fprintf(&buf, " T(%02x %4d %4d)", t.Id&0x7f, t.X, t.Y)
		}
	}
	return buf.String()
}

func (s *State) Decode(p []byte) error {
	if len(p) == 0 {
		return fmt.Errorf("short packet")
	}
	switch p[0] {
	case 0x01:
		// pass
	case 0x11:
		p = p[2:]
	default:
		return fmt.Errorf("unrecognised packet")
	}

	if len(p) < 43 {
		return fmt.Errorf("short packet")
	}

	s.LX, s.LY = p[1], p[2]
	s.RX, s.RY = p[3], p[4]
	s.Button = uint32(p[5]) | uint32(p[6])<<8 | uint32(p[7])<<16
	s.L2, s.R2 = p[8], p[9]

	s.XAcc, s.YAcc, s.ZAcc = u16triplet(p[14:20])
	s.XGyro, s.YGyro, s.ZGyro = u16triplet(p[20:26])

	s.Battery = p[30]

	s.Packet = p[34]
	decodeTouch(p[35:39], &s.Touch[0])
	decodeTouch(p[39:43], &s.Touch[1])

	return nil
}

func (s *State) GyroRoll() float64 {
	return GyroRoll(gyroVec(s.XGyro, s.YGyro, s.ZGyro))
}

func (s *State) GyroPitch() float64 {
	return GyroPitch(gyroVec(s.XGyro, s.YGyro, s.ZGyro))
}

func (s *State) GyroRollPitch() (roll, pitch float64) {
	return GyroRollPitch(gyroVec(s.XGyro, s.YGyro, s.ZGyro))
}

func (s *State) GyroVec() (x, y, z float64) {
	return gyroVec(s.XGyro, s.YGyro, s.ZGyro)
}

func (s *State) Finger(fid byte) *Touch {
	if s.Touch[0].Id == fid {
		return &s.Touch[0]
	}
	if s.Touch[1].Id == fid {
		return &s.Touch[1]
	}
	return nil
}

func u16triplet(p []byte) (x, y, z int16) {
	x = int16(p[0])<<8 | int16(p[1])
	y = int16(p[2])<<8 | int16(p[3])
	z = int16(p[4])<<8 | int16(p[5])
	return
}

func decodeTouch(p []byte, t *Touch) {
	t.Id = p[0]
	t.X = int16(p[2]&0x0f)<<8 | int16(p[1])
	t.Y = int16(p[3])<<4 | (int16(p[2])&0xf0)>>4
}
