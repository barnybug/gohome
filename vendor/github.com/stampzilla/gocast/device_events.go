package gocast

import "github.com/stampzilla/gocast/events"

func (d *Device) OnEvent(callback func(event events.Event)) {
	d.eventListeners = append(d.eventListeners, callback)
}

func (d *Device) Dispatch(event events.Event) {
	for _, callback := range d.eventListeners {
		go callback(event)
	}
}
