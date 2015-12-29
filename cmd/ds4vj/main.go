package main

import (
	"errors"
	"flag"
	"io"
	"log"
	"time"

	"github.com/tajtiattila/hid/ds4/ds4util"
	"github.com/tajtiattila/vjoy"
)

func main() {
	swipe := flag.Bool("swipehat", false, "use touchpad swipes as extra hat")
	slider := flag.Bool("slider", false, "use touchpad as slider")
	throttle := flag.Bool("throttle", false, "use touchpad as throttle")
	bumper := flag.Bool("bumper", false, "special bumper shift logic")
	gyro := flag.Bool("gyro", false, "feed gyro roll/pitch to second device")
	alpha := flag.Float64("alpha", 1, "alpha value for 1st order gyro smoothing filter")
	movavg := flag.Duration("movavg", 0, "moving average filter duration for gyro")
	flag.Parse()

	if !vjoy.Available() {
		Fatal(errors.New("vjoy.dll missing?"))
	}

	vjd, err := vjoy.Acquire(1)
	if err != nil {
		Fatal(err)
	}
	defer vjd.Relinquish()
	vjd.Reset()
	vjd.Update()

	var vjd2 *vjoy.Device
	if *gyro {
		vjd2, err = vjoy.Acquire(2)
		if err != nil {
			Fatal(err)
		}
		defer vjd2.Relinquish()
		vjd2.Reset()
		vjd2.Update()
	}

	connh := new(connHandler)
	connh.vjd = VJD{dev: vjd, dev2: vjd2}

	switch {
	case *swipe:
		connh.touchlogic = TouchHat
	case *slider:
		connh.touchlogic = TouchSlider
	case *throttle:
		connh.touchlogic = TouchThrottle
	}

	if *bumper {
		connh.logic = BumpShiftLogic
	}

	connh.ngf = func() ds4util.Filter {
		return newFilter(*alpha, *movavg)
	}

	var dm *ds4util.DeviceManager

	guimain(func(w io.Writer, ch chan<- ds4util.Event) {
		l := log.New(w, "", log.Ltime)
		dm = ds4util.NewDeviceManager(connh, l)
		go func() {
			for e := range dm.Event() {
				ch <- e
			}
		}()
	})

	if dm != nil {
		dm.Close()
	}
}

func newFilter(a float64, d time.Duration) ds4util.Filter {
	var af, mf ds4util.Filter
	if 0 < a && a < 1 {
		af = ds4util.NewAlphaFilter(3, a)
	}
	if d > 0 {
		// make room for 3 readings per Millisecond
		n := 3 * int(d/time.Millisecond)
		mf = ds4util.NewMovAvg(3, n, d)
	}

	if af != nil {
		if mf == nil {
			return af
		}
		return ds4util.Combine(af, mf)
	}
	if mf != nil {
		return mf
	}
	return ds4util.Input
}
