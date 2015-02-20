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

	"gopkg.in/v1/yaml"
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
	Cameras map[string]CameraNodeConf
}

type CurrentcostConf struct {
	Device string
}

type DeviceConf struct {
	Id    string `json:"id"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	Group string `json:"group"`
}

type DataloggerConf struct {
	Path string
}

type EarthConf struct {
	Latitude  float64
	Longitude float64
}

type EndpointsConf struct {
	Zeromq struct {
		Pub string
		Sub string
	}
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
	Host string
}

type HeatingConf struct {
	Device   string
	Schedule map[string]map[string][]map[string]float64
	Sensors  []string
	Slop     float64
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
}

type JabberConf struct {
	Jid  string
	Pass string
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

type VoiceConf map[string]string

type WeatherConf struct {
	Outside struct {
		Rain string
		Temp string
		Wind string
	}
	Windy float64
}

type WatchdogConf struct {
	Devices map[string]string
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
	Processes    map[string]ProcessConf
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
	Rfid         RfidConf
	SMS          SMSConf
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
		device.Type = strings.Split(id, ".")[0]
		self.Devices[id] = device
	}

	for name, cf := range self.Processes {
		cf.Cmd = util.ExpandUser(cf.Cmd)
		cf.Path = util.ExpandUser(cf.Path)
		self.Processes[name] = cf
	}
	return self, nil
}

// Find the device name
func (self *Config) LookupDeviceName(ev *pubsub.Event) string {
	topic := ev.Topic
	source := ev.Source()
	// try: protocol.id
	if device, ok := self.Protocols[topic][source]; ok {
		return device
	}
	// fallback: topic.source
	// ignore dynamic topics (prefix _)
	if topic != "" && source != "" && !strings.HasPrefix(topic, "_") {
		return topic + "." + source
	}
	return ""
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
