package main

import (
	"errors"
	"io"
	"log"

	"github.com/tajtiattila/hid/ds4/ds4util"
	"github.com/tajtiattila/vjoy"
)

func main() {
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

	connh := new(connHandler)
	connh.vjd.dev = vjd

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
