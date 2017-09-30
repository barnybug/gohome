# yeelight

Library to manipulate Xiaomi/Yeelight light products.

## ALPHA-ALPHA-ALPHA-ALPHA-ALPHA-ALPHA-ALPHA

Still working on it. Basic functionality works.

## PACKAGE DOCUMENTATION

    package yeelight
        import "http://github.com/edgard/yeelight"


    FUNCTIONS

    func Discover(timeout time.Duration) ([]Light, error)
        Discover uses SSDP to find and return the IP address of the lights

    TYPES

    type Command struct {
        ID     int32       `json:"id"`
        Method string      `json:"method"`
        Params interface{} `json:"params"`
    }
        Command to send to the light

    type Light struct {
        Location   string   `json:"location,omitempty"`
        ID         string   `json:"id,omitempty"`
        Model      string   `json:"model,omitempty"`
        FWVersion  int      `json:"fw_ver,omitempty"`
        Support    []string `json:"support,omitempty"`
        Power      string   `json:"power,omitempty"`
        Bright     int      `json:"bright,omitempty"`
        ColorMode  int      `json:"color_mode,omitempty"`
        ColorTemp  int      `json:"ct,omitempty"`
        RGB        int      `json:"rgb,omitempty"`
        Hue        int      `json:"hue,omitempty"`
        Saturation int      `json:"sat,omitempty"`
        Name       string   `json:"name,omitempty"`
    }
        Light properties

    func (m *Light) PowerOff(duration int) (string, error)
        PowerOff takes transition duration and power off the light

    func (m *Light) PowerOn(duration int) (string, error)
        PowerOn takes transition duration and power on the light

    func (m *Light) SendCommand(cmd Command) (string, error)
        SendCommand sends a single command to the light

    func (m *Light) SetBrightness(level, duration int) (string, error)
        SetBrightness takes the brightness (0-100), transition duration and set
        brightness of the light

    func (m *Light) SetRGB(red, green, blue, duration int) (string, error)
        SetRGB takes r, g, b values (0-255), transition duration and set color
        of the light

    func (m *Light) SetTemp(temp, duration int) (string, error)
        SetTemp takes temperature values (1700-6500K), transition duration and
        set color temperature of the light

    func (m *Light) Toggle() (string, error)
        Toggle toggles light state on or off

    func (m *Light) Update() error
        Update the current status of the light

    type Result struct {
        ID     int32       `json:"id"`
        Result interface{} `json:"result,omitempty"`
        Error  interface{} `json:"error,omitempty"`
    }
        Result reply of the sent command

    SUBDIRECTORIES

        examples
