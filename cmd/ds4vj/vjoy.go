package main

import (
	"math"
	"sync"
	"time"

	"github.com/tajtiattila/hid/ds4"
	"github.com/tajtiattila/hid/ds4/ds4util"
	"github.com/tajtiattila/vjoy"
)

type VJD struct {
	mtx sync.Mutex
	dev *vjoy.Device
}

type vjoyHandler struct {
	vjd *VJD
	d   *ds4.Device
	tl  TouchLogic
	sl  SetStater

	bw bool
}

const (
	TouchHat    = 1 // swipe → hat
	TouchSlider = 2 // touch position → slider

	NormalLogic    = 0
	BumpShiftLogic = 1
)

type connHandler struct {
	vjd VJD

	logic      int
	touchlogic int
}

func (ch *connHandler) Connect(d *ds4.Device, e ds4util.Entry) (ds4util.StateHandler, error) {
	h := &vjoyHandler{
		vjd: &ch.vjd,
		d:   d,
		bw:  batteryWarn(e.Battery),
	}

	switch ch.logic {
	case NormalLogic:
		h.sl = SetStaterFunc(setState)
	case BumpShiftLogic:
		h.sl = new(bumperLogic)
	}

	switch ch.touchlogic {
	case TouchHat:
		sh := new(swipeHat)
		sh.logic = ds4util.NewSwipeLogic(sh)
		h.tl = sh

	case TouchSlider:
		h.tl = new(touchSlider)

	default:
		h.tl = new(emptyLogic)
	}

	h.setColor()
	return h, nil
}

func (h *vjoyHandler) State(s *ds4.State) error {
	bw := s.Battery&0x10 == 0 && s.Battery&0xF < 2
	if h.bw != bw {
		h.bw = bw
		h.setColor()
	}

	h.tl.HandleState(s)

	vj := h.vjd.dev

	h.vjd.mtx.Lock()
	defer h.vjd.mtx.Unlock()
	vj.Hat(1).SetDiscrete(h.tl.HatState())
	vj.Axis(vjoy.Slider0).Setuf(h.tl.Slider())
	h.sl.SetState(vj, s)
	vj.Update()

	return nil
}

func (h *vjoyHandler) Close() error {
	h.vjd.mtx.Lock()
	defer h.vjd.mtx.Unlock()
	h.vjd.dev.Reset()
	h.vjd.dev.Update()
	return nil
}

func (h *vjoyHandler) setColor() {
	if h.bw {
		h.d.SetFlashColor(ds4.Color{0xff, 0x00, 0x00},
			time.Second/2, time.Second/2)
	} else {
		h.d.SetColor(ds4.Color{0xff, 0x88, 0x00})
	}
}

type SetStater interface {
	SetState(vj *vjoy.Device, s *ds4.State)
}

type SetStaterFunc func(vj *vjoy.Device, s *ds4.State)

func (f SetStaterFunc) SetState(vj *vjoy.Device, s *ds4.State) {
	f(vj, s)
}

func setState(vj *vjoy.Device, s *ds4.State) {
	button := s.Button & ^uint32(ds4.L2|ds4.R2)

	// reduce trigger sensitivity
	const triggerMinPull = 20
	if s.L2 >= triggerMinPull {
		button |= ds4.L2
	}
	if s.R2 >= triggerMinPull {
		button |= ds4.R2
	}

	for i, m := range []uint32{
		ds4.Cross,
		ds4.Circle,
		ds4.Square,
		ds4.Triangle,

		ds4.L1,
		ds4.R1,
		ds4.L2,
		ds4.R2,
		ds4.L3,
		ds4.R3,

		ds4.Options,
		ds4.Share,
		ds4.PS,
		ds4.Click,
	} {
		vj.Button(uint(i)).Set(button&m != 0)
	}

	vj.Hat(0).SetDiscrete(hat(button))

	vj.Axis(vjoy.AxisX).Setf(axis(s.LX))
	vj.Axis(vjoy.AxisY).Setf(axis(s.LY))
	vj.Axis(vjoy.AxisRX).Setf(axis(s.RX))
	vj.Axis(vjoy.AxisRY).Setf(axis(s.RY))

	vj.Axis(vjoy.AxisZ).Setf(gyroAxis(s.GyroRoll()))
}

func batteryWarn(b byte) bool {
	return b&0x10 == 0 && b&0xF < 2
}

func axis(v byte) float32 {
	const deadzone = 0.05
	r0 := (float32(v)/255*2 - 1) * math.Sqrt2
	r := r0
	if r < 0 {
		r += deadzone
	} else {
		r -= deadzone
	}
	if r*r0 < 0 {
		return 0
	}
	if r < -1 {
		return -1
	}
	if 1 < r {
		return 1
	}
	return r
}

// 10°...45° -> 0..1
func gyroAxis(v float64) float32 {
	r := v
	if r < 0 {
		r += 10
	} else {
		r -= 10
	}
	if r*v < 0 {
		return 0
	}
	r /= 35
	if r < -1 {
		return -1
	}
	if 1 < r {
		return 1
	}
	return float32(r)
}

func hat(button uint32) vjoy.HatState {
	switch button & ds4.Dpad {
	case 0:
		return vjoy.HatN
	case 2:
		return vjoy.HatE
	case 4:
		return vjoy.HatS
	case 6:
		return vjoy.HatW
	}
	return vjoy.HatOff
}

type TouchLogic interface {
	HandleState(s *ds4.State)

	Slider() float32
	HatState() vjoy.HatState
}

type emptyLogic struct{}

func (*emptyLogic) HandleState(s *ds4.State) {}
func (*emptyLogic) Slider() float32          { return 0 }
func (*emptyLogic) HatState() vjoy.HatState  { return vjoy.HatOff }

type swipeHat struct {
	logic *ds4util.SwipeLogic

	mtx   sync.Mutex
	swipe [4]int
}

func (h *swipeHat) HandleState(s *ds4.State) {
	h.logic.HandleState(s)
}

func (h *swipeHat) Slider() float32 {
	return 0
}

func (h *swipeHat) HatState() vjoy.HatState {
	h.mtx.Lock()
	defer h.mtx.Unlock()
	switch {
	case h.swipe[0] > 0:
		return vjoy.HatN
	case h.swipe[1] > 0:
		return vjoy.HatE
	case h.swipe[2] > 0:
		return vjoy.HatS
	case h.swipe[3] > 0:
		return vjoy.HatW
	}
	return vjoy.HatOff
}

func (h *swipeHat) Touch(x, y int) {
}

func (h *swipeHat) Click(x, y int) {
}

func (h *swipeHat) Swipe(dir int, ntouch int) {
	if ntouch == 1 {
		var n int
		switch dir {
		case ds4util.SwipeUp:
			n = int(vjoy.HatN)
		case ds4util.SwipeRight:
			n = int(vjoy.HatE)
		case ds4util.SwipeLeft:
			n = int(vjoy.HatW)
		case ds4util.SwipeDown:
			n = int(vjoy.HatS)
		default:
			return
		}

		time.AfterFunc(100*time.Millisecond, func() {
			h.mtx.Lock()
			defer h.mtx.Unlock()
			h.swipe[n]--
		})

		h.mtx.Lock()
		defer h.mtx.Unlock()
		h.swipe[n]++
	}
}

type touchSlider struct {
	v float32
}

func (t *touchSlider) HandleState(s *ds4.State) {
	if !s.Touch[0].Active() {
		return
	}
	const (
		right  = 1920
		border = 200
	)
	x := int(s.Touch[0].X)
	// useful range is 0..1
	// clamping is not needed, vjoy.Axis.Setuf() will do just that
	t.v = float32(x-border) / (right - 2*border)
}

func (t *touchSlider) Slider() float32 { return t.v }

func (*touchSlider) HatState() vjoy.HatState { return vjoy.HatOff }
