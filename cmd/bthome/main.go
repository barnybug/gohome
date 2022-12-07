package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/go-ble/ble"
	"github.com/go-ble/ble/linux"
	"github.com/pkg/errors"
)

func main() {
	d, err := linux.NewDevice()
	if err != nil {
		log.Fatal("Can't create new device:", err)
	}
	ble.SetDefaultDevice(d)
	scan()
}

type Reading struct {
	mac      ble.Addr
	packetId uint8
	temp     *float32
	humidity *float32
	voltage  *float32
	power    *bool
}

func readInt16(data []byte, scale int16) float32 {
	var i int16
	binary.Read(bytes.NewReader(data), binary.LittleEndian, &i)
	return float32(i) / float32(scale)
}

func readUint16(data []byte, scale uint16) float32 {
	var i uint16
	binary.Read(bytes.NewReader(data), binary.LittleEndian, &i)
	return float32(i) / float32(scale)
}

func decodeReading(data []byte) (Reading, error) {
	reading := Reading{}
	offset := data
	// [2 0 216   2 16 1   3 12 108 12]
	for len(offset) > 0 {
		length := offset[0] & 0xf
		datatype := offset[1]
		switch datatype {
		case 0x0: // packet id
			reading.packetId = offset[2]
		case 0x2: // temp
			temp := readInt16(offset[2:4], 100)
			reading.temp = &temp
		case 0x3: // humidity
			humidity := readUint16(offset[2:4], 100)
			reading.humidity = &humidity
		case 0xC: // voltage
			voltage := readUint16(offset[2:4], 1000)
			reading.voltage = &voltage
		case 0x10: // power
			power := true
			if offset[2] == 0 {
				power = false // !
			}
			reading.power = &power
		}
		offset = offset[length+1:]
	}
	return reading, nil
}

var readingChannel chan Reading

func adScanHandler(a ble.Advertisement) {
	for _, serviceData := range a.ServiceData() {
		if serviceData.UUID.Equal(ble.UUID16(0x181c)) {
			reading, _ := decodeReading(serviceData.Data)
			reading.mac = a.Addr()
			readingChannel <- reading
		}
	}
}

func readings() {
	lastPacketIds := map[string]byte{}
	for reading := range readingChannel {
		if reading.packetId == lastPacketIds[reading.mac.String()] {
			continue
		}
		if reading.temp != nil && reading.humidity != nil {
			event := map[string]interface{}{
				"topic":    "temp",
				"temp":     *reading.temp,
				"humidity": *reading.humidity,
				"source":   fmt.Sprintf("ble.%s", reading.mac),
			}
			data, _ := json.Marshal(event)
			fmt.Println(string(data))
		}
		if reading.voltage != nil {
			log.Printf("%s voltage: %.3fV", reading.mac, *reading.voltage)
		}
		lastPacketIds[reading.mac.String()] = reading.packetId
	}
}

func scan() {
	readingChannel = make(chan Reading, 10)
	dur := 5 * time.Second
	log.Println("Started scanning")

	go readings()
	for {
		ctx := ble.WithSigHandler(context.WithTimeout(context.Background(), dur))
		err := ble.Scan(ctx, false, adScanHandler, nil)
		if errors.Cause(err) == context.Canceled {
			break
		} else if errors.Cause(err) == context.DeadlineExceeded {
			continue
		}
		log.Fatalf(err.Error())
	}
}
