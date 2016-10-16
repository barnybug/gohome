// Service to communicate with energenie sockets. This can both receive and
// transmit events.
package energenie

import (
	"fmt"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/barnybug/ener314"
	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
)

func handleCommand(ev *pubsub.Event) {
	dev := ev.Device()
	command := ev.Command()
	pids := services.Config.LookupDeviceProtocol(dev)
	if pids["energenie"] == "" {
		return // command not for us
	}
	if command != "off" && command != "on" {
		log.Println("Command not recognised:", command)
		return
	}
}

type Action string

const (
	TargetTemperature Action = "TargetTemperature"
	Identify          Action = "Identify"
	Exercise          Action = "Exercise"
	Diagnostics       Action = "Diagnostics"
	Voltage           Action = "Voltage"
)

type SensorRequest struct {
	Action Action
	Value  float64
	Repeat int
}

type SensorRequestQueue []SensorRequest

func (q SensorRequestQueue) Append(s SensorRequest) SensorRequestQueue {
	// dedup
	var ret SensorRequestQueue
	for _, i := range q {
		if i.Action != s.Action {
			ret = append(ret, i)
		}
	}
	return append(ret, s)
}

func (q SensorRequestQueue) Len() int { return len(q) }

func (q SensorRequestQueue) Swap(i, j int) { q[i], q[j] = q[j], q[i] }

func (q SensorRequestQueue) Less(i, j int) bool {
	if q[j].Action == TargetTemperature {
		// i should come before j if j is TargetTemperature
		return true
	} else if q[i].Action == TargetTemperature {
		// j should come before i if i is TargetTemperature
		return false
	} else {
		// otherwise just use alphabetic order
		return strings.Compare(string(q[i].Action), string(q[j].Action)) == -1
	}
}

func (s SensorRequest) String() string {
	if s.Action == TargetTemperature {
		return fmt.Sprintf("Target temperature %.1f°C", s.Value)
	} else {
		return fmt.Sprint(s.Action)
	}
}

// Service energenie
type Service struct {
	dev     *ener314.Device
	targets map[string]float64
	queue   map[uint32]SensorRequestQueue
}

func (self *Service) ID() string {
	return "energenie"
}

func round(f float64, dp int) float64 {
	shift := math.Pow(10, float64(dp))
	return math.Floor(f*shift+.5) / shift
}

func emitTemp(msg *ener314.Message, record ener314.Temperature) {
	source := fmt.Sprintf("energenie.%06x", msg.SensorId)
	value := record.Value
	fields := map[string]interface{}{
		"source": source,
		"temp":   round(value, 1),
	}
	ev := pubsub.NewEvent("temp", fields)
	services.Publisher.Emit(ev)
}

func (self *Service) sendQueuedRequests(sensorId uint32) {
	if qu, ok := self.queue[sensorId]; ok {
		// sort the requests, so TargetTemps are last
		sort.Sort(qu)

		var requeue SensorRequestQueue
		for _, request := range qu {
			log.Printf("%06x Sending %s\n", sensorId, request)
			switch request.Action {
			case TargetTemperature:
				self.dev.TargetTemperature(sensorId, request.Value)
			case Identify:
				self.dev.Identify(sensorId)
			case Diagnostics:
				self.dev.Diagnostics(sensorId)
			case Exercise:
				self.dev.ExerciseValve(sensorId)
			case Voltage:
				self.dev.Voltage(sensorId)
			}

			if request.Repeat > 0 {
				// requeue any requests to repeat
				request.Repeat -= 1
				requeue = append(requeue, request)
			}
		}

		if len(requeue) > 0 {
			self.queue[sensorId] = requeue
		} else {
			delete(self.queue, sensorId)
		}
	}
}

func (self *Service) handleMessage(msg *ener314.Message) {
	record := msg.Records[0] // only examine first record
	switch t := record.(type) {
	case ener314.Join:
		log.Printf("%06x Join\n", msg.SensorId)
		self.dev.Join(msg.SensorId)
	case ener314.Temperature:
		log.Printf("%06x Temperature: %.1f°C\n", msg.SensorId, t.Value)
		emitTemp(msg, t)
		// the eTRV is listening - this is the opportunity to send any queued requests
		self.sendQueuedRequests(msg.SensorId)
	case ener314.Voltage:
		log.Printf("%06x Voltage: %.2fV\n", msg.SensorId, t.Value)
	case ener314.Diagnostics:
		log.Printf("%06x Diagnostics report: %s\n", msg.SensorId, t)
	}
}

func lookupSensorId(device string) uint32 {
	trv := strings.Replace(device, "thermostat.", "trv.", 1)
	matches := services.Config.LookupDeviceProtocol(trv)
	if sid, ok := matches["energenie"]; ok {
		id, _ := strconv.ParseUint(sid, 16, 32)
		return uint32(id)
	}
	return 0
}

func (self *Service) handleThermostat(ev *pubsub.Event) {
	var current float64
	var ok bool
	if current, ok = self.targets[ev.Device()]; !ok {
		current = -1 // target not set
	}

	target, ok := ev.Fields["target"].(float64)
	if !ok {
		log.Println("Error: thermostat event target field invalid:", ev)
		return
	}
	if current == target {
		return // nothing to do
	}

	// lookup sensorid
	sensorId := lookupSensorId(ev.Device())
	if sensorId != 0 {
		self.queueRequest(sensorId, SensorRequest{Action: TargetTemperature, Value: target, Repeat: 2})
	}
	self.targets[ev.Device()] = target
}

func (self *Service) queueRequest(sensorId uint32, request SensorRequest) {
	log.Printf("%06x Queueing %s\n", sensorId, request)
	self.queue[sensorId] = self.queue[sensorId].Append(request)
}

func (self *Service) handleCommand(ev *pubsub.Event) {
	sensorId := lookupSensorId(ev.Device())
	if sensorId == 0 {
		return // command not for us
	}
	switch ev.Command() {
	case "identify":
		self.queueRequest(sensorId, SensorRequest{Action: Identify})
	case "diagnostics":
		self.queueRequest(sensorId, SensorRequest{Action: Diagnostics})
	case "exercise":
		self.queueRequest(sensorId, SensorRequest{Action: Exercise})
	case "voltage":
		self.queueRequest(sensorId, SensorRequest{Action: Voltage})
	default:
		log.Println("Command not recognised:", ev.Command())
	}
}

func (self *Service) Run() error {
	self.targets = map[string]float64{}
	self.queue = map[uint32]SensorRequestQueue{}

	ener314.SetLevel(ener314.LOG_TRACE)
	dev := ener314.NewDevice()
	self.dev = dev
	err := dev.Start()
	if err != nil {
		return err
	}
	thermostatChannel := services.Subscriber.FilteredChannel("thermostat")
	commandChannel := services.Subscriber.FilteredChannel("command")

	receiveTimer := time.NewTimer(time.Millisecond)

	for {
		select {
		case <-receiveTimer.C:
			// poll receive
			for msg := dev.Receive(); msg != nil; {
				self.handleMessage(msg)
				// check for any more
				msg = dev.Receive()
			}
			// check again in 100ms
			receiveTimer.Reset(100 * time.Millisecond)
		case ev := <-thermostatChannel:
			self.handleThermostat(ev)
		case ev := <-commandChannel:
			self.handleCommand(ev)
		}
	}
}
