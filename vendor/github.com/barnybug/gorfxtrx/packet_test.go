package gorfxtrx

import (
	"encoding/hex"
	"fmt"
)

func ExampleStatus() {
	pkt, err := Parse([]byte{0x14, 0x01, 0x00, 0x01, 0x03, 0x53, 0x09, 0x20, 0x00, 0x2f, 0x00, 0x01, 0x01, 0x1c, 0x01, 0x00, 0x049, 0x00, 0x00, 0x00, 0x00})
	fmt.Printf("%v\n", pkt)
	fmt.Println(err)
	//Output:
	// Status: type: 433.92MHz transceiver: 83 firmware: 9 protocols: ac, arc, byron sx, homeeasy, oregon, x10
	// <nil>
}

func ExampleShortBytes() {
	_, err := Parse([]byte{0x0d, 0x01, 0x00, 0x01, 0x02, 0x53, 0x3e, 0x00, 0x0c, 0x2f, 0x01, 0x01, 0x00})
	fmt.Println(err)
	//Output:
	// Packet unexpected length: 12 != 13
}

func ExampleShortData() {
	_, err := Parse([]byte{0x01, 0x01})
	fmt.Println(err)
	//Output:
	// Status packet incorrect length, expected: 20 actual: 1
}

func ExampleStatusSend() {
	p := &Status{}
	fmt.Println(p.Send())
	//Output:
	// [13 0 0 1 2 0 0 0 0 0 0 0 0 0]
}

func ExampleResetSend() {
	p, err := NewReset()
	fmt.Println(err)
	fmt.Println(p.Send())
	//Output:
	// <nil>
	// [13 0 0 0 0 0 0 0 0 0 0 0 0 0]
}

func ExampleTransmitAck() {
	pkt, err := Parse([]byte{0x04, 0x02, 0x01, 0x00, 0x00})
	fmt.Printf("%v\n", pkt)
	fmt.Println(err)
	//Output:
	// TransmitAck: ACK
	// <nil>
}

func ExampleLightingX10() {
	x, _ := Parse([]byte{0x07, 0x10, 0x00, 0x2a, 0x45, 0x05, 0x01, 0x70})
	lighting := *x.(*LightingX10)
	fmt.Printf("%+v\n", lighting)
	fmt.Println(lighting.Type())
	fmt.Println(lighting.Id())
	fmt.Println(lighting.Command())
	//Output:
	// {typeId:0 SequenceNumber:42 HouseCode:69 UnitCode:5 command:1}
	// X10 lighting
	// e05
	// on
}

func ExampleLightingX10Send() {
	p, _ := NewLightingX10(0x01, "e05", "on")
	fmt.Println(p.Send())
	//Output:
	// [7 16 1 0 69 5 1 0]
}

func ExampleLightingHE() {
	x, _ := Parse([]byte{0x0b, 0x11, 0x00, 0x2a, 0x01, 0x23, 0x45, 0x67, 0x05, 0x02, 0x08, 0x70})
	lighting := *x.(*LightingHE)
	fmt.Printf("%+v\n", lighting)
	fmt.Println(lighting.Type())
	fmt.Println(lighting.Id())
	fmt.Println(lighting.Command())
	//Output:
	// {typeId:0 SequenceNumber:42 HouseCode:19088743 UnitCode:5 command:2 Level:8}
	// AC
	// 12345675
	// set level
}

func ExampleLightingHENewBad() {
	_, err := NewLightingHE(0x00, "bad", "on")
	fmt.Println(err)
	//Output:
	// id should be 8 characters (eg. 1234567b)
}

func ExampleLightingHESend() {
	p, err := NewLightingHE(0x00, "002A41E6", "on")
	fmt.Println(err)
	fmt.Println(p.Send())
	//Output:
	// <nil>
	// [11 17 0 0 0 2 164 30 6 1 0 0]
}

func ExampleChime() {
	x, _ := Parse([]byte{0x07, 0x16, 0x00, 0x06, 0x00, 0x7a, 0x01, 0x70})
	chime := *x.(*Chime)
	fmt.Printf("%+v\n", chime)
	fmt.Printf("%+v\n", chime.Id())
	fmt.Printf("%+v\n", chime.Type())
	//Output:
	// {typeId:0 SequenceNumber:6 id:122 Chime:1 Battery:0 Rssi:7}
	// 00:7a
	// Byron SX
}

func ExampleTemp() {
	x, _ := Parse([]byte{0x08, 0x50, 0x02, 0x2a, 0x96, 0x03, 0x81, 0x41, 0x79})
	temp := *x.(*Temp)
	fmt.Printf("%+v\n", temp)
	fmt.Printf("%+v\n", temp.Id())
	fmt.Printf("%+v\n", temp.Type())
	//Output:
	// {typeId:2 SequenceNumber:42 id:38403 Temp:-32.1 Battery:90 Rssi:7}
	// 96:03
	// THC238/268,THN132,THWR288,THRN122,THN122,AW129/131
}

func ExampleTempHumid() {
	x, _ := Parse([]byte{0x0a, 0x52, 0x01, 0x2a, 0x96, 0x03, 0x81, 0x41, 0x60, 0x03, 0x79})
	temp := *x.(*TempHumid)
	fmt.Printf("%+v\n", temp)
	fmt.Printf("%+v\n", temp.Id())
	fmt.Printf("%+v\n", temp.Type())
	//Output:
	// {TypeId:1 SequenceNumber:42 id:38403 Temp:-32.1 Humidity:96 HumidityStatus:3 Battery:90 Rssi:7}
	// 96:03
	// THGN122/123, THGN132, THGR122/228/238/268
}

func ExampleWind() {
	x, _ := Parse([]byte{0x10, 0x56, 0x01, 0x03, 0x2F, 0x00, 0x00, 0xF7, 0x00, 0x20, 0x00, 0x24, 0x01, 0x60, 0x00, 0x00, 0x59})
	wind := *x.(*Wind)
	fmt.Printf("%+v\n", wind)
	fmt.Println(wind.Id())
	fmt.Println(wind.Type())
	//Output:
	// {data:[16 86 1 3 47 0 0 247 0 32 0 36 1 96 0 0 89] typeId:1 SequenceNumber:3 id:12032 Direction:247 AverageSpeed:3.2 Gust:3.6 Battery:90 Rssi:5}
	// 2f:00
	// WTGR800
}

func ExampleRain() {
	x, _ := Parse([]byte{0x0b, 0x55, 0x02, 0x03, 0x12, 0x34, 0x02, 0x50, 0x01, 0x23, 0x45, 0x57})
	rain := *x.(*Rain)
	fmt.Printf("%+v\n", rain)
	fmt.Println(rain.Id())
	fmt.Println(rain.Type())
	//Output:
	// {typeId:2 SequenceNumber:3 id:4660 RainRate:5.92 RainTotal:7456.5 Battery:70 Rssi:5}
	// 12:34
	// PCR800
}

func ExampleOwl113() {
	data, _ := hex.DecodeString("0D59010C8900070000001A000079")
	pkt, err := Parse(data)
	fmt.Printf("%v\n", pkt)
	fmt.Println(err)
	//Output:
	// Current id: 8900 current1: 0.0A current2: 2.6A current3: 0.0A signal: 7 battery: 90
	// <nil>
}

func ExampleOwl180() {
	data, _ := hex.DecodeString("115a02028782000000010100000000849069") // 257 W 151 Wh
	pkt, err := Parse(data)
	fmt.Printf("%v\n", pkt)
	fmt.Println(err)
	//Output:
	// Power id: 8782 power: 257W total: 151.73Wh signal: 6 battery: 90
	// <nil>
}

func ExampleUnknown() {
	pkt, _ := Parse([]byte{0x01, 0xFF})
	fmt.Printf("%+v\n", pkt)
	//Output:
	// Unknown: 01ff
}
