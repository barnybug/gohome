package config

import (
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/util"

	"gopkg.in/yaml.v1"
)

type BillConf struct {
	Electricity struct {
		Primary_Rate    float64
		Standing_Charge float64
	}
	Vat      float64
	Currency string
}

type CameraNodeConf struct {
	Protocol string
	Url      string
	User     string
	Password string
}

type CameraConf struct {
	Path    string
	Url     string
	Port    int
	Cameras map[string]CameraNodeConf
}

type CurrentcostConf struct {
	Device string
}

type DeviceConf struct {
	Id       string   `json:"id"`
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Group    string   `json:"group"`
	Location string   `json:"location"`
	Caps     []string `json:"caps"`
	Cap      map[string]bool
}

type DataloggerConf struct {
	Path string
}

type EarthConf struct {
	Latitude  float64
	Longitude float64
}

type EndpointsConf struct {
	Nanomsg struct {
		Pub string
		Sub string
	}
	Mqtt struct {
		Broker string
	}
	Api string
}

type EspeakConf struct {
	Args string
}

type GeneralEmailConf struct {
	Admin  string
	From   string
	Server string
}

type GeneralConf struct {
	Email GeneralEmailConf
}

type Duration struct {
	Duration time.Duration
}

func (self *Duration) SetYAML(tag string, value interface{}) bool {
	if value, ok := value.(string); ok {
		val, err := time.ParseDuration(value)
		if err == nil {
			self.Duration = val
			return true
		}
	}
	return false
}

type GraphiteConf struct {
	Url string
	Tcp string
}

type ScheduleConf map[string][]map[string]float64

type ZoneConf struct {
	Sensor   string
	Schedule ScheduleConf
}

type HeatingConf struct {
	Device  string
	Zones   map[string]ZoneConf
	Slop    float64
	Minimum float64
}

type IrrigationConf struct {
	At       *Duration
	Device   string
	Enabled  bool
	Factor   float64
	Interval *Duration
	Max_Temp float64
	Max_Time float64
	Min_Temp float64
	Min_Time float64
	Sensor   string
}

type JabberConf struct {
	Jid  string
	Pass string
}

type OrviboConf struct {
	Broadcast string
}

type PushbulletConf struct {
	Token string
}

type PresenceConf struct {
	Trigger string
	People  map[string][]string
}

type ProcessConf struct {
	Cmd  string
	Path string
}

type SMSConf struct {
	Device    string
	Telephone string
}

type TwitterConf struct {
	Auth struct {
		Consumer_key    string
		Consumer_secret string
		Token           string
		Token_secret    string
	}
}

type RfidConf struct {
	Device string
}

type SlackConf struct {
	Token string
}

type TelegramConf struct {
	Token   string
	Chat_id int64
}

type VoiceConf map[string]string

type WeatherConf struct {
	Sensors struct {
		Rain     string
		Temp     string
		Wind     string
		Pressure string
	}
	Windy float64
}

type WatchdogConf struct {
	Alert     string
	Devices   map[string]string
	Processes []string
	Pings     []string
}

type WundergroundConf struct {
	Id       string
	Password string
	Url      string
}

// Configuration structure
type Config struct {
	// yaml fields
	Devices      map[string]DeviceConf
	Protocols    map[string]map[string]string
	Endpoints    EndpointsConf
	Bill         BillConf
	Camera       CameraConf
	Currentcost  CurrentcostConf
	Datalogger   DataloggerConf
	Earth        EarthConf
	Espeak       EspeakConf
	General      GeneralConf
	Graphite     GraphiteConf
	Heating      HeatingConf
	Irrigation   IrrigationConf
	Jabber       JabberConf
	Orvibo       OrviboConf
	Presence     PresenceConf
	Pushbullet   PushbulletConf
	Rfid         RfidConf
	Slack        SlackConf
	SMS          SMSConf
	Telegram     TelegramConf
	Twitter      TwitterConf
	Voice        VoiceConf
	Watchdog     WatchdogConf
	Weather      WeatherConf
	Wunderground WundergroundConf
}

// Open configuration from disk.
func Open() (*Config, error) {
	file, err := os.Open(ConfigPath("gohome.yml"))
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return OpenReader(file)
}

// Open configuration from a reader.
func OpenReader(r io.Reader) (*Config, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return OpenRaw(data)
}

// Open configuration from []byte.
func OpenRaw(data []byte) (*Config, error) {
	self := &Config{}
	err := yaml.Unmarshal(data, self)
	if err != nil {
		return nil, err
	}

	for id, device := range self.Devices {
		device.Id = id
		if len(device.Caps) == 0 {
			major := strings.Split(id, ".")[0]
			device.Caps = []string{major}
		}
		device.Type = device.Caps[0]
		device.Cap = map[string]bool{}
		for _, c := range device.Caps {
			device.Cap[c] = true
		}
		self.Devices[id] = device
	}

	return self, nil
}

func (self *Config) AddDeviceToEvent(ev *pubsub.Event) {
	// split source into protocol.id
	ps := strings.SplitN(ev.Source(), ".", 2)
	protocol := ps[0]
	var id string
	if len(ps) > 1 {
		id = ps[1]
	}
	device := self.Protocols[protocol][id]
	if device != "" {
		ev.SetField("device", device)
	}
}

// Find the protocol and identifier for by device name
func (self *Config) LookupDeviceProtocol(matchName string) map[string]string {
	ret := map[string]string{}
	for protocol, value := range self.Protocols {
		for id, name := range value {
			if name == matchName {
				ret[protocol] = id
			}
		}
	}
	return ret
}

// helpers

// Resolve a configuration file under .config/gohome
func ConfigPath(p string) string {
	config := os.Getenv("XDG_CONFIG_HOME")
	if config == "" {
		config = path.Join(os.Getenv("HOME"), ".config")
	}
	return path.Join(config, "gohome", p)
}

// Get path to a log file
func LogPath(p string) string {
	return path.Join(util.ExpandUser("~/go/log"), p)
}
