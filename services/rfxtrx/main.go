// Service to communicate with an rfxcom USB transceiver. This can both receive
// and transmit events.
package rfxtrx

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"path/filepath"
	"strings"
	"time"

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"

	"github.com/barnybug/gorfxtrx"
)

// Service rfxtrx
type Service struct {
	inflight chan *pubsub.Event
}

func deviceName(s string) string {
	return strings.Replace(strings.ToLower(s), " ", "", -1)
}

// Read events from the rfxtrx
func (self *Service) readEvents(dev *gorfxtrx.Device) {
	for {
		packet, err := dev.Read()
		if err != nil {
			log.Println("Error reading:", err)
			continue
		}
		if packet == nil {
			continue
		}

		ev := self.translatePacket(packet)

		if ev != nil {
			services.Publisher.Emit(ev)
		}

	}
}

func (self *Service) translatePacket(packet gorfxtrx.Packet) *pubsub.Event {
	var ev *pubsub.Event
	switch p := packet.(type) {
	case *gorfxtrx.Status:
		// no event emitted
		protocols := strings.Join(p.Protocols(), ", ")
		log.Printf("Status: type: %s transceiver: %d firmware: %d protocols: %s", p.TypeString(), p.TransceiverType, p.FirmwareVersion, protocols)
	case *gorfxtrx.LightingX10:
		source := fmt.Sprintf("x10.%s", p.Id())
		fields := map[string]interface{}{
			"source":  source,
			"group":   p.Id()[:1],
			"command": p.Command(),
		}
		ev = pubsub.NewEvent("x10", fields)

	case *gorfxtrx.LightingHE:
		source := fmt.Sprintf("homeeasy.%07X%1X", p.HouseCode, p.UnitCode)
		fields := map[string]interface{}{
			"source":  source,
			"command": p.Command(),
		}
		ev = pubsub.NewEvent("homeeasy", fields)

	case *gorfxtrx.Temp:
		source := fmt.Sprintf("thn132n.%s", p.Id()[0:2])
		fields := map[string]interface{}{
			"source":  source,
			"temp":    p.Temp,
			"battery": p.Battery,
		}
		ev = pubsub.NewEvent("temp", fields)

	case *gorfxtrx.TempHumid:
		major := strings.ToLower(strings.Split(p.Type(), ",")[0])
		source := fmt.Sprintf("%s.%s", major, p.Id()[0:2])
		fields := map[string]interface{}{
			"source":   source,
			"temp":     p.Temp,
			"humidity": p.Humidity,
			"battery":  p.Battery,
		}
		ev = pubsub.NewEvent("temp", fields)

	case *gorfxtrx.Wind:
		source := strings.ToLower(p.Type()) + ".0"
		fields := map[string]interface{}{
			"source":   source,
			"speed":    p.Gust,
			"avgspeed": p.AverageSpeed,
			"dir":      p.Direction,
			"battery":  p.Battery,
		}
		ev = pubsub.NewEvent("wind", fields)

	case *gorfxtrx.Rain:
		device := strings.ToLower(p.Type())
		source := fmt.Sprintf("%s.%s", device, p.Id())
		fields := map[string]interface{}{
			"source":    source,
			"rate":      p.RainRate,
			"all_total": p.RainTotal,
			"battery":   p.Battery,
		}
		ev = pubsub.NewEvent("rain", fields)

	case *gorfxtrx.Chime:
		device := deviceName(p.Type())
		source := fmt.Sprintf("%s.%s", device, strings.Replace(p.Id(), ":", "", 1))
		fields := map[string]interface{}{
			"source":  source,
			"chime":   p.Chime,
			"battery": p.Battery,
			"command": "on",
		}
		ev = pubsub.NewEvent("chime", fields)

	case *gorfxtrx.Power:
		source := fmt.Sprintf("owl.%04x", p.SensorId)
		fields := map[string]interface{}{
			"source":  source,
			"power":   p.Power,
			"battery": p.Battery,
			"signal":  p.Signal,
		}
		if p.Total != 0 {
			// owl reports zero values occasionally
			fields["total"] = p.Total
		}
		ev = pubsub.NewEvent("power", fields)

	case *gorfxtrx.TransmitAck:
		pending := <-self.inflight
		if p.OK() {
			fields := map[string]interface{}{
				"device":  pending.Device(),
				"command": pending.Command(),
			}
			ev = pubsub.NewEvent("ack", fields)
		} else {
			log.Printf("Transmit failed: %s for %s\n", packet, pending)
		}

	default:
		log.Printf("Ignored unhandled packet: %T: %s\n", packet, packet)
	}

	if ev != nil && ev.Device() == "" {
		services.Config.AddDeviceToEvent(ev)
	}

	return ev
}

// Translate command messages into rfxtrx packets
func translateCommands(ev *pubsub.Event) (gorfxtrx.OutPacket, error) {
	device := ev.Device()
	command := ev.Command()
	pids := services.Config.LookupDeviceProtocol(device)
	if len(pids) == 0 {
		return nil, nil
	}

	if pids["homeeasy"] == "" && pids["x10"] == "" {
		// command not for us
		return nil, nil
	}

	if command != "off" && command != "on" {
		log.Println("Command not recognised:", command)
		return nil, nil
	}
	level := ev.IntField("level")
	if level > 255 {
		level = 255
	}

	switch {
	case pids["homeeasy"] != "":
		if level != 0 {
			command = "set level"
		}
		pkt, err := gorfxtrx.NewLightingHE(0x00, pids["homeeasy"], command)
		if level != 0 {
			pkt.Level = byte(level)
		}
		return pkt, err
	case pids["x10"] != "":
		return gorfxtrx.NewLightingX10(0x01, pids["x10"], command)
	}
	return nil, nil
}

var repeats map[string]*time.Timer = map[string]*time.Timer{}

func (self *Service) repeatSend(dev *gorfxtrx.Device, event *pubsub.Event, pkt gorfxtrx.OutPacket, repeat int64) error {
	// cancel any existing timer
	device := event.Device()
	if t, ok := repeats[device]; ok {
		delete(repeats, device)
		t.Stop()
	}

	err := dev.Send(pkt)
	if err != nil {
		return err
	}
	self.inflight <- event

	if repeat > 1 {
		duration := time.Duration((rand.Float64()*2 + 1) * float64(time.Second))
		repeats[device] = time.AfterFunc(duration, func() {
			self.repeatSend(dev, event, pkt, repeat-1)
		})
	}
	return nil
}

func (self *Service) transmitCommands(dev *gorfxtrx.Device) {
	for ev := range services.Subscriber.FilteredChannel("command") {
		pkt, err := translateCommands(ev)
		if err != nil {
			log.Println("Couldn't translate command:", err)
			continue
		}
		if pkt == nil {
			// command not translated
			continue
		}
		repeat := ev.IntField("repeat")
		err = self.repeatSend(dev, ev, pkt, repeat)
		if err != nil {
			log.Fatalln("Exiting after error sending:", err)
		}
	}
}

func getStatus(dev *gorfxtrx.Device) {
	log.Println("Setting mode")
	setmode := &gorfxtrx.SetMode{}
	err := dev.Send(setmode)
	if err != nil {
		log.Println("Error sending packet:", err)
	}

	log.Println("Getting status")
	status := &gorfxtrx.Status{}
	err = dev.Send(status)
	if err != nil {
		log.Println("Error sending packet:", err)
	}
}

func defaultDevName() string {
	matches, _ := filepath.Glob("/dev/serial/by-id/usb-RFXCOM_RFXtrx433_*-if00-port0")
	if len(matches) > 0 {
		return matches[0]
	}
	return ""
}

func (self *Service) ID() string {
	return "rfxtrx"
}

func (self *Service) emptyInflight() {
	more := true
	for more {
		select {
		case <-self.inflight:
			break
		default:
			more = false
			break
		}
	}
}

func (self *Service) Run() error {
	self.inflight = make(chan *pubsub.Event, 10)
	devname := defaultDevName()
	if devname == "" {
		return errors.New("rfxtrx device not found")
	}

	for {
		dev, err := gorfxtrx.Open(devname, false)
		if err != nil {
			log.Fatal("Error opening device: ", devname)
		}
		log.Println("Connected")

		// get device status 300ms after reset
		time.AfterFunc(300*time.Millisecond, func() { getStatus(dev) })

		go self.transmitCommands(dev)
		self.readEvents(dev)

		log.Println("Disconnected")
		dev.Close()

		self.emptyInflight()
	}
	return nil
}
