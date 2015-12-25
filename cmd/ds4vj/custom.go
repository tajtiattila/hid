package main

import (
	"github.com/tajtiattila/hid/ds4"
	"github.com/tajtiattila/vjoy"
)

// bumperLogic is a special logic where L1/R1 (bumpers) are shifts.
//
// L1 converts X to Z and RY to RZ axes.
//
// When either L1 or R1 is pressed, another dpad or button click is sent.
//
// The circle button is also special, a hat press together
type bumperLogic struct {
	special uint32
}

func (l *bumperLogic) SetState(vj *vjoy.Device, s *ds4.State) {
	const (
		buttonmask = 0xfffff

		circleSpec = 1 << 20
		shiftSpec  = 1 << 21
	)

	button := (s.Button & buttonmask) & ^uint32(ds4.L2|ds4.R2)

	// reduce trigger sensitivity
	const triggerMinPull = 20
	if s.L2 >= triggerMinPull {
		button |= ds4.L2
	}
	if s.R2 >= triggerMinPull {
		button |= ds4.R2
	}

	// special circle processing
	if l.special&circleSpec != 0 {
		// both circle and dpad released
		if button&ds4.Circle == 0 && button&ds4.DpadOff != 0 {
			l.special &^= circleSpec
		}
	} else {
		// circle and dpad held together
		if button&ds4.Circle != 0 && button&ds4.DpadOff == 0 {
			l.special |= circleSpec
		}
	}

	// special shift processing
	if l.special&shiftSpec != 0 {
		// no shiftable button is being pressed
		if !hasBumperShiftable(button) {
			l.special &^= shiftSpec
		}
	} else {
		// shiftable button together with shift pressed
		if button&(ds4.L1|ds4.R1) != 0 && hasBumperShiftable(button) {
			l.special |= shiftSpec
		}
	}

	button |= l.special

	for i, m := range []btnmask{
		{ds4.Cross, shiftSpec},
		{ds4.Circle, shiftSpec | circleSpec}, // circle and no dpad
		{ds4.Square, shiftSpec},
		{ds4.Triangle, shiftSpec},

		{ds4.L1, ds4.R1}, // L1 without R1
		{ds4.R1, ds4.L1}, // R1 without L1
		{ds4.L2, 0},
		{ds4.R2, 0},
		{ds4.L3, shiftSpec},
		{ds4.R3, shiftSpec},

		{ds4.Options, 0},
		{ds4.Share, 0},
		{ds4.PS, 0},
		{ds4.Click, 0},

		{ds4.L1 | ds4.R1, 0}, // L1 and R1 together

		{circleSpec, 0}, // special circle with dpad

		// shifted buttons
		{ds4.Cross | shiftSpec, 0},
		{ds4.Circle | shiftSpec, circleSpec},
		{ds4.Square | shiftSpec, 0},
		{ds4.Triangle | shiftSpec, 0},
		{ds4.L3 | shiftSpec, 0},
		{ds4.R3 | shiftSpec, 0},
	} {
		pushed := button&m.set == m.set && button&m.unset == 0
		vj.Button(uint(i)).Set(pushed)
	}

	switch {
	case l.special&circleSpec != 0:
		vj.Hat(0).SetDiscrete(vjoy.HatOff)
		vj.Hat(2).SetDiscrete(hat(button))
		vj.Hat(3).SetDiscrete(vjoy.HatOff)
	case l.special&shiftSpec != 0:
		//case button&(ds4.L1|ds4.R1) != 0:
		vj.Hat(0).SetDiscrete(vjoy.HatOff)
		vj.Hat(2).SetDiscrete(vjoy.HatOff)
		vj.Hat(3).SetDiscrete(hat(button))
	default:
		vj.Hat(0).SetDiscrete(hat(button))
		vj.Hat(2).SetDiscrete(vjoy.HatOff)
		vj.Hat(3).SetDiscrete(vjoy.HatOff)
	}

	lx, ly := axis(s.LX), axis(s.LY)
	rx, ry := axis(s.RX), axis(s.RY)

	if button&(ds4.L1|ds4.R1) == ds4.L1 {
		vj.Axis(vjoy.AxisX).Setf(0)
		vj.Axis(vjoy.AxisZ).Setf(lx)
		vj.Axis(vjoy.AxisRY).Setf(0)
		vj.Axis(vjoy.AxisRZ).Setf(ry)
	} else {
		vj.Axis(vjoy.AxisX).Setf(lx)
		vj.Axis(vjoy.AxisZ).Setf(0)
		vj.Axis(vjoy.AxisRY).Setf(ry)
		vj.Axis(vjoy.AxisRZ).Setf(0)
	}
	vj.Axis(vjoy.AxisY).Setf(ly)
	vj.Axis(vjoy.AxisRX).Setf(rx)
}

type btnmask struct {
	set, unset uint32
}

func hasBumperShiftable(btn uint32) bool {
	const (
		shiftable = ds4.Cross | ds4.Circle | ds4.Square | ds4.Triangle | ds4.L3 | ds4.R3 | ds4.DpadOff
	)
	return btn&shiftable != ds4.DpadOff
}
