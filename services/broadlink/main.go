// Service to communicate with bradlink wifi devices.
package broadlink

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"

	broadlink "github.com/barnybug/go-broadlink"
)

// Service broadlink
type Service struct {
	discovered map[string]*broadlink.Device
	state      map[string]*broadlink.BGState
	deviceMap  map[string]*broadlink.Device
}

func (self *Service) ID() string {
	return "broadlink"
}

func (self *Service) handleCommand(ev *pubsub.Event) {
	id, ok := services.Config.LookupDeviceProtocol(ev.Device(), "broadlink")
	if !ok {
		return // command not for us
	}
	device, ok := self.deviceMap["broadlink."+id]
	if !ok {
		log.Printf("Device %s not found", id)
		return
	}
	command := ev.Command()
	if command != "off" && command != "on" {
		log.Println("Command not recognised:", command)
		return
	}
	log.Printf("Setting device %s to %s\n", ev.Device(), command)
	socket, _ := strconv.Atoi(id[len(id)-1:])
	var state *broadlink.BGState
	if socket == 1 {
		if command == "on" {
			state = broadlink.StatePwr1On
		} else {
			state = broadlink.StatePwr1Off
		}
	} else {
		if command == "on" {
			state = broadlink.StatePwr2On
		} else {
			state = broadlink.StatePwr2Off
		}
	}
	state, err := device.SetState(state)
	if err != nil {
		log.Printf("Failure setting state: %s", err)
	} else {
		log.Printf("Set state to: %s", state)
		self.checkState(device, state)
	}
}

func ack(device, command string) {
	log.Printf("State %s: %s", device, command)
	fields := pubsub.Fields{
		"device":  device,
		"command": command,
	}
	ack := pubsub.NewEvent("ack", fields)
	services.Publisher.Emit(ack)

}

func deviceId(device *broadlink.Device) string {
	return device.MacString()
}

func socketSource(device *broadlink.Device, n int) string {
	return fmt.Sprintf("broadlink.%s%d", device.MacString(), n)
}

func deviceName(device *broadlink.Device, n int) string {
	return fmt.Sprintf("BG Electrical Smart Socket %d", n)
}

func announce(source, name string) {
	fields := pubsub.Fields{"source": source, "name": name}
	ev := pubsub.NewEvent("announce", fields)
	services.Config.AddDeviceToEvent(ev)
	services.Publisher.Emit(ev)
}

func command(i uint8) string {
	if i > 0 {
		return "on"
	}
	return "off"
}

func (self *Service) checkState(device *broadlink.Device, state *broadlink.BGState) {
	did := deviceId(device)
	previous := self.state[did]
	if previous == nil || *previous.Pwr1 != *state.Pwr1 {
		if id, ok := services.Config.LookupSource(socketSource(device, 1)); ok {
			ack(id, command(*state.Pwr1))
		}
	}
	if previous == nil || *previous.Pwr2 != *state.Pwr2 {
		if id, ok := services.Config.LookupSource(socketSource(device, 2)); ok {
			ack(id, command(*state.Pwr2))
		}
	}
	self.state[did] = state
}

func (self *Service) handleDiscovery(device *broadlink.Device) {
	id := deviceId(device)
	if dev, exists := self.discovered[id]; !exists {
		log.Printf("Discovered %s", id)
		self.discovered[id] = device
		if err := device.Auth(); err != nil {
			log.Printf("Device auth failed: %s", err)
			return
		}
		log.Printf("Authenticated successfully with %s", id)
		// announce two sockets per device
		for i := 1; i <= 2; i++ {
			source := socketSource(device, i)
			if _, exists := self.deviceMap[source]; !exists {
				self.deviceMap[source] = device
			}
			announce(source, deviceName(device, i))
		}
	} else {
		device = dev
	}

	state, err := device.GetState()
	if err != nil {
		log.Printf("Getting state failed: %s", err)
	} else {
		self.checkState(device, state)
	}
}

func (self *Service) Init() error {
	self.discovered = map[string]*broadlink.Device{}
	self.state = map[string]*broadlink.BGState{}
	self.deviceMap = map[string]*broadlink.Device{}
	services.WaitForConfig()
	return nil
}

func (self *Service) Run() error {
	manager := broadlink.NewManager(false)
	commandChannel := services.Subscriber.Subscribe(pubsub.Prefix("command"))

	go func() {
		for {
			manager.Discover(5 * time.Second)
			time.Sleep(time.Minute)
		}
	}()

	for {
		select {
		case command := <-commandChannel:
			self.handleCommand(command)
		case device := <-manager.Discovered:
			self.handleDiscovery(device)
		}
	}
	return nil
}
