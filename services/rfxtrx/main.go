// Service to communicate with an rfxcom USB transceiver. This can both receive
// and transmit events.
package rfxtrx

import (
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"

	"github.com/barnybug/gorfxtrx"
)

// Read events from the rfxtrx
func readEvents(dev *gorfxtrx.Device) {
	for {
		packet, err := dev.Read()
		if err != nil {
			log.Println("Error reading:", err)
			continue
		}
		if packet == nil {
			continue
		}

		ev := translatePacket(packet)

		if ev != nil {
			services.Publisher.Emit(ev)
		}

	}
}

const Origin = "rfxtrx"

func deviceName(s string) string {
	return strings.Replace(strings.ToLower(s), " ", "", -1)
}

func translatePacket(packet gorfxtrx.Packet) *pubsub.Event {
	var ev *pubsub.Event
	switch p := packet.(type) {
	case *gorfxtrx.Status:
		// no event emitted
		protocols := strings.Join(p.Protocols(), ", ")
		log.Printf("Status: type: %s transceiver: %d firmware: %d protocols: %s", p.TypeString(), p.TransceiverType, p.FirmwareVersion, protocols)
	case *gorfxtrx.LightingX10:
		fields := map[string]interface{}{
			"source":  p.Id(),
			"group":   p.Id()[:1],
			"command": p.Command(),
		}
		ev = pubsub.NewEvent("x10", fields)

	case *gorfxtrx.LightingHE:
		id := fmt.Sprintf("%07X%1X", p.HouseCode, p.UnitCode)
		fields := map[string]interface{}{
			"source":  id,
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
		source := strings.ToLower(p.Type())
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
		source := fmt.Sprintf("%04x", p.SensorId)
		fields := map[string]interface{}{
			"source":  source,
			"power":   p.Power,
			"total":   p.Total,
			"battery": p.Battery,
			"signal":  p.Signal,
		}
		ev = pubsub.NewEvent("power", fields)

	default:
		log.Printf("Ignored unhandled packet: %T: %s\n", packet, packet)
	}

	return ev
}

// Translate command messages into rfxtrx packets
func translateCommands(ev *pubsub.Event) (gorfxtrx.OutPacket, error) {
	device := ev.Device()
	command := ev.Command()
	pids := services.Config.LookupDeviceProtocol(device)
	if len(pids) == 0 {
		log.Println("Device not found for:", device)
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

var RepeatInterval, _ = time.ParseDuration("3s")

var repeats map[string]*time.Timer = map[string]*time.Timer{}

func repeatSend(dev *gorfxtrx.Device, device string, pkt gorfxtrx.OutPacket, repeat int64) error {
	// cancel any existing timer
	if t, ok := repeats[device]; ok {
		delete(repeats, device)
		t.Stop()
	}

	err := dev.Send(pkt)
	if err != nil {
		return err
	}

	if repeat > 1 {
		repeats[device] = time.AfterFunc(RepeatInterval, func() {
			repeatSend(dev, device, pkt, repeat-1)
		})
	}
	return nil
}

func transmitCommands(dev *gorfxtrx.Device) {
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
		err = repeatSend(dev, ev.Device(), pkt, repeat)
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

// Service rfxtrx
type Service struct{}

func (self *Service) ID() string {
	return "rfxtrx"
}

func (self *Service) Run() error {
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

		go transmitCommands(dev)
		readEvents(dev)

		log.Println("Disconnected")
		dev.Close()
	}
	return nil
}
