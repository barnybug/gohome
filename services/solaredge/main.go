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

func readInverter(client modbus.Client, handler *modbus.TCPClientHandler, serial []byte) (*pubsub.Event, error) {
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

func readMeter(client modbus.Client, handler *modbus.TCPClientHandler, serial []byte) (*pubsub.Event, error) {
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

func readBattery(client modbus.Client, handler *modbus.TCPClientHandler) (*pubsub.Event, error) {
	data, err := client.ReadHoldingRegisters(0xE100, 76)
	if err != nil {
		return nil, err
	}
	fmt.Println(data)
	// m, err := NewBatteryModel(data)
	// if err != nil {
	// 	return nil, err
	// }
	// source := fmt.Sprintf("meter.%s", serial)
	// fields := map[string]interface{}{
	// 	"source":    source,
	// 	"current":   scaleu16(m.M_AC_Current, m.M_AC_Current_SF),
	// 	"voltage":   scaleu16(m.M_AC_VoltageLN, m.M_AC_Voltage_SF),
	// 	"power":     scalei16(m.M_AC_Power, m.M_AC_Power_SF),
	// 	"frequency": scaleu16(m.M_AC_Frequency, m.M_AC_Frequency_SF),
	// 	"exported":  scaleu32(m.M_Exported, m.M_Energy_W_SF),
	// 	"imported":  scaleu32(m.M_Imported, m.M_Energy_W_SF),
	// }
	// ev := pubsub.NewEvent("power", fields)
	// services.Config.AddDeviceToEvent(ev)
	// return ev, nil
	return nil, nil
}

func readCycle(client modbus.Client, handler *modbus.TCPClientHandler, serial []byte, meterSerial []byte) {
	inv, err := readInverter(client, handler, serial)
	if err != nil {
		log.Fatalf("Error reading inverter: %s", err)
	}
	if inv != nil {
		services.Publisher.Emit(inv)
	}
	meter, err := readMeter(client, handler, meterSerial)
	if err != nil {
		log.Fatalf("Error reading meter: %s", err)
	}
	if meter != nil {
		services.Publisher.Emit(meter)
	}

	readBattery(client, handler)
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
	cm, err := NewCommonModel(infoData)
	log.Printf("Inverter Model: %s", cm.C_Model)
	log.Printf("Inverter Serial: %s", cm.C_SerialNumber)
	log.Printf("Inverter Version: %s", cm.C_Version)

	infoData2, err := client.ReadHoldingRegisters(40121, 65)
	cm2, err := NewCommonMeter(infoData2)
	log.Printf("Meter Manufacturer: %s", cm2.C_Manufacturer)
	log.Printf("Meter Model: %s", cm2.C_Model)
	log.Printf("Meter Serial: %s", cm2.C_SerialNumber)
	log.Printf("Meter Version: %s", cm2.C_Version)
	log.Printf("Meter Option: %s", cm2.C_Option)

	readCycle(client, handler, cm.C_SerialNumber, cm2.C_SerialNumber)
	ticker := time.NewTicker(15 * time.Second)
	for range ticker.C {
		readCycle(client, handler, cm.C_SerialNumber, cm2.C_SerialNumber)
	}
	return nil
}
