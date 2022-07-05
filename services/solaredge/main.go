// Service to retrieve sensors data for Mi Flora bluetooth sensors
package solaredge

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
	"github.com/goburrow/modbus"
)

const MaxRetries = 5

// Service solaredge
type Service struct{}

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

func readInverter(client modbus.Client, handler *modbus.TCPClientHandler, serial string) (*pubsub.Event, error) {
	for {
		inverterData, err := client.ReadHoldingRegisters(40069, 40)
		if err != nil {
			log.Printf("Error reading holding registers: %s", err.Error())
			log.Printf("Attempting to reconnect")
			_ = handler.Close()
			time.Sleep(7 * time.Second)
			_ = handler.Connect()
			continue
		}
		i, err := NewInverterModel(inverterData)
		if err != nil {
			return nil, err
		}

		source := fmt.Sprintf("inverter.%s", serial)
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
		ev := pubsub.NewEvent("solaredge", fields)
		services.Config.AddDeviceToEvent(ev)
		return ev, nil
	}
}

func readMeter(client modbus.Client, handler *modbus.TCPClientHandler, serial string) (*pubsub.Event, error) {
	infoData, err := client.ReadHoldingRegisters(40188, 105)
	if err != nil {
		return nil, err
	}
	m, err := NewMeterModel(infoData)
	if err != nil {
		return nil, err
	}
	source := fmt.Sprintf("meter.%s", serial)
	fields := map[string]interface{}{
		"source":    source,
		"current":   scaleu16(m.M_AC_Current, m.M_AC_Current_SF),
		"voltage":   scaleu16(m.M_AC_VoltageLN, m.M_AC_Voltage_SF),
		"power":     scalei16(m.M_AC_Power, m.M_AC_Power_SF),
		"frequency": scaleu16(m.M_AC_Frequency, m.M_AC_Frequency_SF),
		"exported":  scaleu32(m.M_Exported, m.M_Energy_W_SF),
		"imported":  scaleu32(m.M_Imported, m.M_Energy_W_SF),
	}
	ev := pubsub.NewEvent("power", fields)
	services.Config.AddDeviceToEvent(ev)
	return ev, nil
}

func readBatteryData(client modbus.Client, serial string) (*pubsub.Event, error) {
	b, err := ReadBatteryData(client)
	if err != nil {
		return nil, err
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
	}
	ev := pubsub.NewEvent("battery", fields)
	services.Config.AddDeviceToEvent(ev)
	return ev, nil
}

type Serials struct {
	inverter string
	meter    string
	battery  string
}

func readCycle(client modbus.Client, handler *modbus.TCPClientHandler, serials Serials) {
	inv, err := readInverter(client, handler, serials.inverter)
	if err != nil {
		log.Fatalf("Error reading inverter: %s", err)
	}
	if inv != nil {
		services.Publisher.Emit(inv)
	}
	meter, err := readMeter(client, handler, serials.meter)
	if err != nil {
		log.Fatalf("Error reading meter: %s", err)
	}
	if meter != nil {
		services.Publisher.Emit(meter)
	}

	battery, err := readBatteryData(client, serials.battery)
	if err != nil {
		log.Fatalf("Error reading meter: %s", err)
	}
	if battery != nil {
		services.Publisher.Emit(battery)
	}
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
	client := modbus.NewClient(handler)
	defer handler.Close()

	// Collect and log common inverter data
	infoData, err := client.ReadHoldingRegisters(40000, 70)
	inv, err := NewCommonModel(infoData)
	if err != nil {
		log.Fatalf("Error reading Inverter: %s", err.Error())
	}
	log.Printf("Inverter Model: %s", inv.C_Model)
	log.Printf("Inverter Serial: %s", inv.C_SerialNumber)
	log.Printf("Inverter Version: %s", inv.C_Version)

	infoData2, err := client.ReadHoldingRegisters(40121, 65)
	meter, err := NewCommonMeter(infoData2)
	if err != nil {
		log.Fatalf("Error reading Meter: %s", err.Error())
	}
	log.Printf("Meter Manufacturer: %s", meter.C_Manufacturer)
	log.Printf("Meter Model: %s", meter.C_Model)
	log.Printf("Meter Serial: %s", meter.C_SerialNumber)
	log.Printf("Meter Version: %s", meter.C_Version)
	log.Printf("Meter Option: %s", meter.C_Option)

	battery, err := ReadBatteryInfo(client)
	if err != nil {
		log.Fatalf("Error reading Battery: %s", err.Error())
	}
	log.Printf("Battery Manufacturer: %s", battery.Manufacturer)
	log.Printf("Battery Model: %s", battery.Model)
	log.Printf("Battery Firmware: %s", battery.Firmware)
	log.Printf("Battery Serial: %s", battery.Serial)
	log.Printf("Battery Rated Energy: %.1f kWh", battery.RatedEnergy/1000)
	log.Printf("Battery Charge/Discharge (Continuous): %.1f/%.1f kW", battery.MaxPowerContinuousCharge/1000, battery.MaxPowerContinuousDischarge/1000)

	serials := Serials{
		inverter: string(inv.C_SerialNumber),
		meter:    string(meter.C_SerialNumber),
		battery:  string(battery.Serial[:]),
	}

	readCycle(client, handler, serials)
	ticker := time.NewTicker(15 * time.Second)
	for range ticker.C {
		readCycle(client, handler, serials)
	}
	return nil
}
