// Service to retrieve sensors data for Mi Flora bluetooth sensors
package solaredge

import (
	"errors"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
	"github.com/goburrow/modbus"
	"github.com/u-root/u-root/pkg/uio"
)

const MaxRetries = 5
const MaxImportLimit = 5400
const MaxExportLimit = 5400

// Service solaredge
type Service struct {
	client               modbus.Client
	remoteMode           ChargeDischargeMode
	remoteTimeout        time.Time
	remoteChargeLimit    float32
	remoteDischargeLimit float32
}

// ID of the service
func (self *Service) ID() string {
	return "solaredge"
}

func scalei16(value int16, sf int16) float64 {
	return float64(value) * math.Pow10(int(sf))
}

// TODO: go 1.18 generics
func scalei32(value int32, sf uint16) float64 {
	return float64(value) * math.Pow10(int(sf))
}

func scaleu16(value uint16, sf int16) float64 {
	return float64(value) * math.Pow10(int(sf))
}

func scaleu32(value uint32, sf int16) float64 {
	return float64(value) * math.Pow10(int(sf))
}

func readInverter(client modbus.Client, handler *modbus.TCPClientHandler, serial string) (ev *pubsub.Event, ac_power, dc_power float64, err error) {
	var inverterData []byte
	for {
		inverterData, err = client.ReadHoldingRegisters(40069, 40)
		if err != nil {
			log.Printf("Error reading holding registers: %s", err.Error())
			log.Printf("Attempting to reconnect")
			_ = handler.Close()
			time.Sleep(7 * time.Second)
			_ = handler.Connect()
			continue
		}
		break
	}

	var i InverterModel
	i, err = NewInverterModel(inverterData)
	if err != nil {
		return
	}

	source := fmt.Sprintf("inverter.%s", serial)
	ac_power = scalei16(i.AC_Power, i.AC_Power_SF)
	dc_power = scalei16(i.DC_Power, i.DC_Power_SF)
	fields := map[string]interface{}{
		"source":       source,
		"ac_current":   scaleu16(i.AC_Current, i.AC_Current_SF),
		"ac_voltage":   scaleu16(i.AC_VoltageAN, i.AC_Voltage_SF),
		"ac_power":     scalei16(i.AC_Power, i.AC_Power_SF),
		"ac_frequency": scaleu16(i.AC_Frequency, i.AC_Frequency_SF),
		"ac_energy_wh": scalei32(i.AC_Energy_WH, i.AC_Energy_WH_SF),
		"dc_current":   scaleu16(i.DC_Current, i.DC_Current_SF),
		"dc_voltage":   scaleu16(i.DC_Voltage, i.DC_Voltage_SF),
		"dc_power":     scalei16(i.DC_Power, i.DC_Power_SF),
		"temp":         scalei16(i.Temp_Sink, i.Temp_SF),
	}
	ev = pubsub.NewEvent("solaredge", fields)
	services.Config.AddDeviceToEvent(ev)
	return
}

func readMeter(client modbus.Client, handler *modbus.TCPClientHandler, serial string) (event *pubsub.Event, power float64, err error) {
	var infoData []byte
	infoData, err = client.ReadHoldingRegisters(40188, 105)
	if err != nil {
		return
	}
	m, err := NewMeterModel(infoData)
	if err != nil {
		return
	}
	// validate
	if m.M_Exported == 0 {
		err = errors.New("Invalid meter data (export 0)")
		return
	}

	source := fmt.Sprintf("meter.grid") // serial unreliable read
	power = scalei16(m.M_AC_Power, m.M_AC_Power_SF)
	fields := map[string]interface{}{
		"source":    source,
		"current":   scaleu16(m.M_AC_Current, m.M_AC_Current_SF),
		"voltage":   scaleu16(m.M_AC_VoltageLN, m.M_AC_Voltage_SF),
		"power":     -power, // +ve import, -ve export
		"frequency": scaleu16(m.M_AC_Frequency, m.M_AC_Frequency_SF),
		"exported":  scaleu32(m.M_Exported, m.M_Energy_W_SF),
		"imported":  scaleu32(m.M_Imported, m.M_Energy_W_SF),
	}
	event = pubsub.NewEvent("power", fields)
	services.Config.AddDeviceToEvent(event)
	return event, power, nil
}

func readBatteryData(client modbus.Client, serial string) (*pubsub.Event, float32, error) {
	b, err := ReadBatteryData(client)
	if err != nil {
		return nil, 0, err
	}
	source := fmt.Sprintf("battery.%s", serial)
	fields := map[string]interface{}{
		"source":         source,
		"temp":           b.TempAvg,
		"voltage":        b.Voltage,
		"current":        b.Current,
		"power":          b.Power,
		"discharged":     b.Discharged,
		"charged":        b.Charged,
		"capacity_max":   b.BatteryMax,
		"capacity_avail": b.BatteryAvailable,
		"soh":            b.BatterySoH,
		"soc":            b.BatterySoC,
		"status":         b.Status,
		"status_text":    BatteryStatuses[b.Status],
	}
	ev := pubsub.NewEvent("power", fields)
	services.Config.AddDeviceToEvent(ev)
	return ev, b.Power, nil
}

type Serials struct {
	inverter string
	meter    string
	battery  string
}

func (self *Service) readCycle(handler *modbus.TCPClientHandler, serials Serials) {
	inv, ac_power, dc_power, err := readInverter(self.client, handler, serials.inverter)
	if err != nil {
		log.Printf("Error reading inverter: %s", err)
		return
	}
	if inv != nil {
		services.Publisher.Emit(inv)
	}
	battery, battery_power, err := readBatteryData(self.client, serials.battery)
	if err != nil {
		log.Printf("Error reading battery: %s", err)
		return
	}
	if battery != nil {
		services.Publisher.Emit(battery)
	}
	meter, power, err := readMeter(self.client, handler, serials.meter)
	if err != nil {
		log.Printf("Error reading meter: %s", err)
		return
	}
	if meter != nil {
		load := ac_power - power
		solar := dc_power + float64(battery_power)
		if solar < 0 {
			solar = 0
		}
		// TODO: fix transients
		meter.SetField("load", load)
		meter.SetField("solar", solar)
		services.Publisher.Emit(meter)
	}
}

var chargeModes = map[string]ChargeDischargeMode{
	"off": Off,
	// 0 – Off
	"load_then_battery": ChargeFromExcessPVPowerOnly,
	// 1 – Charge excess PV power only.
	// Only PV excess power not going to AC is used for charging the battery. Inverter NominalActivePowerLimit (or the
	// inverter rated power whichever is lower) sets how much power the inverter is producing to the AC. In this mode,
	// the battery cannot be discharged. If the PV power is lower than NominalActivePowerLimit the AC production will
	// be equal to the PV power.
	"battery_then_load": ChargeFromPVFirst,
	// 2 – Charge from PV first, before producing power to the AC.
	// The Battery charge has higher priority than AC production. First charge the battery then produce AC.
	// If StorageRemoteCtrl_ChargeLimit is lower than PV excess power goes to AC according to
	// NominalActivePowerLimit. If NominalActivePowerLimit is reached and battery StorageRemoteCtrl_ChargeLimit is
	// reached, PV power is curtailed.
	"charge": ChargeFromPVAndAC,
	// 3 – Charge from PV+AC according to the max battery power.
	// Charge from both PV and AC with priority on PV power.
	// If PV production is lower than StorageRemoteCtrl_ChargeLimit, the battery will be charged from AC up to
	// NominalActivePow-erLimit. In this case AC power = StorageRemoteCtrl_ChargeLimit- PVpower.
	// If PV power is larger than StorageRemoteCtrl_ChargeLimit the excess PV power will be directed to the AC up to the
	// Nominal-ActivePowerLimit beyond which the PV is curtailed.
	"export": MaximizeExport,
	// 4 – Maximize export – discharge battery to meet max inverter AC limit.
	// AC power is maintained to NominalActivePowerLimit, using PV power and/or battery power. If the PV power is not
	// sufficient, battery power is used to complement AC power up to StorageRemoteCtrl_DishargeLimit. In this mode,
	// charging excess power will occur if there is more PV than the AC limit.
	"load": DischargeToMatchLoad,
	// 5 – Discharge to meet loads consumption. Discharging to the grid is not allowed.
	"unused":  Unused,
	"optimum": MaximizeSelfConsumption,
	// 7 – Maximize self-consumption
}

func (self *Service) handleCommand(ev *pubsub.Event) {
	if _, ok := services.Config.LookupDeviceProtocol(ev.Device(), "inverter"); !ok {
		return
	}
	timeout := uint32(ev.IntField("timeout"))
	mode := ev.StringField("mode")
	if timeout > 86400 {
		log.Println("command error: timeout over 86400")
		return
	}
	remoteMode := Unused
	if ev.IsSet("charge_limit") {
		self.remoteChargeLimit = float32(ev.FloatField("charge_limit"))
	} else {
		self.remoteChargeLimit = MaxImportLimit
	}
	if ev.IsSet("discharge_limit") {
		self.remoteDischargeLimit = float32(ev.FloatField("discharge_limit"))
	} else {
		self.remoteDischargeLimit = MaxExportLimit
	}

	if mode != "" {
		if m, ok := chargeModes[mode]; ok {
			remoteMode = m
		} else {
			log.Println("command error: invalid mode")
			log.Println("Supported modes:")
			for value, _ := range chargeModes {
				log.Printf("- %s", value)
			}
			return
		}
		if timeout == 0 {
			log.Println("timeout is required")
			return
		}
		self.remoteMode = remoteMode
		self.remoteTimeout = time.Now().Add(time.Second * time.Duration(timeout))
		self.sendRemoteMode()
	} else {
		// revert
		self.client.WriteSingleRegister(AddressStoredgeControl, uint16(ControlModeMaximizeSelfConsumption))
	}

	log.Println("Command sent to inverter")

	ci, err := ReadControlInfo(self.client)
	if err != nil {
		log.Println("Error reading control info")
	} else {
		printControlInfo(ci)
	}
}

func (self *Service) sendRemoteMode() {
	now := time.Now()
	if now.After(self.remoteTimeout) {
		return
	}
	buf := uio.NewBigEndianBuffer([]byte{})
	buf.Write16(uint16(ControlModeRemoteControl)) // 0xE004
	buf.Write16(uint16(1))                        // 0xE005 Always allowed
	encode_float32(buf, 0)                        // 0xE006 AC Charge Limit
	encode_float32(buf, 0)                        // 0xE008 Backup Reserved Setting
	defaultMode := MaximizeSelfConsumption
	buf.Write16(uint16(defaultMode)) // 0xE00A
	timeout := self.remoteTimeout.Sub(now).Seconds()
	encode_bele32(buf, uint32(timeout))            // 0xE00B
	buf.Write16(uint16(self.remoteMode))           // 0xE00D
	encode_float32(buf, self.remoteChargeLimit)    // 0xE00E
	encode_float32(buf, self.remoteDischargeLimit) // 0xE010
	self.client.WriteMultipleRegisters(AddressStoredgeControl, uint16(buf.Len()/2), buf.Data())
}

func printControlInfo(ci ControlInfo) {
	log.Printf("Control Current Mode: %s", ci.ControlMode)
	if ci.ControlMode == ControlModeRemoteControl {
		log.Printf("Control Remote Mode: '%s' (Default: '%s') Timeout: %ds Charge/Discharge Limit: %.1f/%0.1f kW", ci.RemoteMode, ci.DefaultMode, ci.RemoteTimeout, ci.RemoteChargeLimit/1000, ci.RemoteDischargeLimit/1000)
	}
	log.Printf("Control AC Charge Policy: %s Limit: %.1f Backup Reserved: %.0f%%", ci.ACChargePolicy, ci.ACChargeLimit, ci.BackupReserved)
}

func (self *Service) QueryHandlers() services.QueryHandlers {
	return services.QueryHandlers{
		"status": services.TextHandler(self.queryStatus),
		"help": services.StaticHandler("" +
			"status: get status\n" +
			"mode: set mode\n"),
	}
}

func formatPower(value float32) string {
	if value > 800 {
		return fmt.Sprintf("%.1fkW", value/1000)
	}
	return fmt.Sprintf("%.0fW", value)
}

func (self *Service) queryStatus(q services.Question) string {
	b, err := ReadBatteryData(self.client)
	if err != nil {
		return fmt.Sprintf("error reading battery: %s", err)
	}

	status := BatteryStatuses[b.Status]
	message := fmt.Sprintf("Battery: %.1f%% (%s) %s", b.BatterySoC, status, formatPower(b.Power))
	return message
}

func parseMode(value string) (err error, mode ChargeDischargeMode, timeout int) {
	vs := strings.Split(value, " ")
	if len(vs) != 2 {
		err = errors.New("mode 60")
		return
	}
	if m, ok := chargeModes[vs[0]]; ok {
		mode = m
	} else {
		err = errors.New("Invalid mode")
		return
	}
	timeout, err = strconv.Atoi(vs[1])
	if err != nil {
		err = errors.New("Invalid timeout")
		return
	}
	return
}

// Run the service
func (self *Service) Run() error {
	handler := modbus.NewTCPClientHandler(services.Config.Solaredge.Inverter)
	handler.Timeout = 10 * time.Second
	handler.SlaveId = 0x01
	err := handler.Connect()
	if err != nil {
		log.Fatalf("Error connecting to Inverter: %s", err.Error())
	}
	self.client = modbus.NewClient(handler)
	defer handler.Close()

	// Collect and log common inverter data
	infoData, err := self.client.ReadHoldingRegisters(40000, 70)
	if err != nil {
		log.Fatalf("Error reading Inverter: %s", err.Error())
	}
	inv, err := NewCommonModel(infoData)
	if err != nil {
		log.Fatalf("Error decoding Inverter: %s", err.Error())
	}
	log.Printf("Inverter Model: %s", inv.C_Model)
	log.Printf("Inverter Serial: %s", inv.C_SerialNumber)
	log.Printf("Inverter Version: %s", inv.C_Version)

	infoData2, err := self.client.ReadHoldingRegisters(40121, 65)
	if err != nil {
		log.Fatalf("Error reading Meter: %s", err.Error())
	}
	meter, err := NewCommonMeter(infoData2)
	if err != nil {
		log.Fatalf("Error decoding Meter: %s", err.Error())
	}
	log.Printf("Meter Manufacturer: %s", meter.C_Manufacturer)
	log.Printf("Meter Model: %s", meter.C_Model)
	log.Printf("Meter Serial: %s", meter.C_SerialNumber)
	log.Printf("Meter Version: %s", meter.C_Version)
	log.Printf("Meter Option: %s", meter.C_Option)

	battery, err := ReadBatteryInfo(self.client)
	if err != nil {
		log.Fatalf("Error reading Battery: %s", err.Error())
	}
	log.Printf("Battery Manufacturer: %s", battery.Manufacturer)
	log.Printf("Battery Model: %s", battery.Model)
	log.Printf("Battery Firmware: %s", battery.Firmware)
	log.Printf("Battery Serial: %s", battery.Serial)
	log.Printf("Battery Rated Energy: %.1f kWh", battery.RatedEnergy/1000)
	log.Printf("Battery Charge/Discharge (Continuous): %.1f/%.1f kW", battery.MaxPowerContinuousCharge/1000, battery.MaxPowerContinuousDischarge/1000)
	self.remoteChargeLimit = battery.MaxPowerContinuousCharge
	self.remoteDischargeLimit = battery.MaxPowerContinuousDischarge

	ci, err := ReadControlInfo(self.client)
	if err != nil {
		log.Fatalf("Error reading Control: %s", err.Error())
	}
	printControlInfo(ci)

	serials := Serials{
		inverter: string(inv.C_SerialNumber),
		meter:    string(meter.C_SerialNumber),
		battery:  string(battery.Serial[:]),
	}

	self.readCycle(handler, serials)
	commands := services.Subscriber.Subscribe(pubsub.Prefix("command"))
	ticker := time.NewTicker(6 * time.Second)
	remote := time.NewTicker(60 * time.Second)
	for {
		select {
		case <-ticker.C:
			self.readCycle(handler, serials)
		case ev := <-commands:
			self.handleCommand(ev)
		case <-remote.C:
			self.sendRemoteMode()
		}
	}
}
