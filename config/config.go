package config

import (
	"io"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/util"
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
	Watch    string
	Match    Regexp
}

type CameraConf struct {
	Path    string
	Url     string
	Port    int
	Cameras map[string]CameraNodeConf
}

type CapsConf map[string][]string

type CurrentcostConf struct {
	Device string
}

type DeviceConf struct {
	Id       string   `json:"id"`
	Name     string   `json:"name"`
	Group    string   `json:"group"`
	Location string   `json:"location"`
	Caps     []string `json:"caps"`
	Aliases  []string `json:"aliases"`
	Source   string   `json:"source"`
	Cap      map[string]bool
}

func (d DeviceConf) IsSwitchable() bool {
	return d.Cap["switch"]
}

func (d DeviceConf) Prefix() string {
	ps := strings.SplitN(d.Id, ".", 2)
	return ps[0]
}

func (d DeviceConf) SourceId() string {
	i := strings.Index(d.Source, ".")
	if i != -1 {
		return d.Source[i+1:]
	}
	return ""
}

type DataloggerConf struct {
	Path string
}

type EarthConf struct {
	Latitude  float64
	Longitude float64
}

type EndpointsConf struct {
	Mqtt struct {
		Broker string
	}
	Api string
}

type EspeakConf struct {
	Args string
	Port int
}

type GeneralEmailConf struct {
	Admin  string
	From   string
	Server string
}

type GeneralConf struct {
	Email   GeneralEmailConf
	Scripts string
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
	Device     string
	Zones      map[string]ZoneConf
	Slop       float64
	Minimum    float64
	Unoccupied float64
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
	Devices   map[string]Duration
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
	Endpoints    EndpointsConf
	Bill         BillConf
	Camera       CameraConf
	Caps         CapsConf
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

	Sources map[string]string // source -> device id
}

// Custom types

type Duration struct {
	time.Duration
}

func (self *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var value string
	if err := unmarshal(&value); err != nil {
		return err
	}

	val, err := time.ParseDuration(value)
	if err != nil {
		return err
	}
	self.Duration = val
	return nil
}

type Regexp struct {
	*regexp.Regexp
}

func (self *Regexp) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var expr string
	if err := unmarshal(&expr); err != nil {
		return err
	}

	r, err := regexp.Compile(expr)
	if err != nil {
		return err
	}
	self.Regexp = r
	return nil
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
	config := &Config{}
	config.Sources = map[string]string{}
	err := yaml.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}

	for id, device := range config.Devices {
		device.Id = id
		// prepend 'inherited' caps by type prefix
		if cs, ok := config.Caps[device.Prefix()]; ok {
			device.Caps = append(cs, device.Caps...)
		}
		if len(device.Caps) == 0 {
			// default to having a cap of device prefix
			device.Caps = []string{device.Prefix()}
		}

		device.Cap = map[string]bool{}
		for _, c := range device.Caps {
			device.Cap[c] = true
		}
		config.Devices[id] = device
		if device.Source != "" {
			config.Sources[device.Source] = device.Id
		}
	}

	return config, nil
}

func (self *Config) AddDeviceToEvent(ev *pubsub.Event) {
	if device, _ := self.LookupSource(ev.Source()); device != "" {
		ev.SetField("device", device)
		ev.SetRetained(true) // retain device events
	}
}

func (self *Config) LookupSource(source string) (string, bool) {
	s, ok := self.Sources[source]
	return s, ok
}

// Find the identifier for a given protocol by device name
func (self *Config) LookupDeviceProtocol(device string, protocol string) (string, bool) {
	if d, ok := self.Devices[device]; ok {
		if strings.HasPrefix(d.Source, protocol+".") {
			// return just the protocol identifier part
			return d.Source[len(protocol)+1:], true
		}
	}
	return "", false
}

func (self *Config) DevicesByProtocol(protocol string) []DeviceConf {
	var ret []DeviceConf
	protocol += "."
	for _, d := range self.Devices {
		if strings.HasPrefix(d.Source, protocol) {
			ret = append(ret, d)
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

func Must(c *Config, err error) *Config {
	if err != nil {
		panic(err)
	}
	return c
}
