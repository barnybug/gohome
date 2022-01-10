// Service to communicate with orvibo sockets. This can both receive and
// transmit events.
package orvibo

import (
	"fmt"
	"log"
	"time"

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
)

func handleCommand(ev *pubsub.Event) {
	dev := ev.Device()
	command := ev.Command()
	ident, ok := services.Config.LookupDeviceProtocol(dev, "orvibo")
	if !ok {
		return // command not for us
	}
	if command != "off" && command != "on" {
		log.Println("Command not recognised:", command)
		return
	}
	if device, ok := devices[ident]; ok {
		log.Printf("Setting device %s to %s\n", dev, command)
		SetState(device, command == "on")
	} else {
		log.Println("Device not recognised:", device)
	}
}

func handleStateChange(msg *StateChangedMessage) {
	log.Printf("Device %s changed to %t\n", msg.Device.MACAddress, msg.State)

	source := fmt.Sprintf("orvibo.%s", msg.Device.MACAddress)
	command := "off"
	if msg.State {
		command = "on"
	}
	fields := map[string]interface{}{
		"source":  source,
		"command": command,
	}
	ev := pubsub.NewEvent("orvibo", fields)
	services.Config.AddDeviceToEvent(ev)
	services.Publisher.Emit(ev)
}

// Service orvibo
type Service struct{}

func (self *Service) ID() string {
	return "orvibo"
}

func (self *Service) Run() error {
	devices := map[string]*Device{}
	commandChannel := services.Subscriber.Subscribe(pubsub.Prefix("command"))

	if services.Config.Orvibo.Broadcast != "" {
		BroadcastAddress = services.Config.Orvibo.Broadcast
	}

	err := Start()
	if err != nil {
		return err
	}
	Discover()

	// Look for new devices every minute
	autoDiscover := time.Tick(5 * time.Minute)
	// Resubscription should happen every 5 minutes, but we make it 3, just to be on the safe side
	resubscribe := time.Tick(3 * time.Minute)

	for {
		select {
		case msg := <-Events:
			switch msg := msg.(type) {
			case *ReadyMessage:
				log.Println("UDP connection ready")
			case *NewDeviceMessage:
				devices[msg.Device.MACAddress] = msg.Device
				log.Printf("Socket found: %s\n", msg.Device.MACAddress)
				time.AfterFunc(300*time.Millisecond, func() {
					Subscribe(msg.Device)
				})
			case *SubscribeAckMessage:
				// noop
			case *StateChangedMessage:
				handleStateChange(msg)
			}

		case <-autoDiscover:
			Discover()

		case <-resubscribe:
			for _, device := range devices {
				Subscribe(device)
			}

		case command := <-commandChannel:
			handleCommand(command)
		}
	}
}
