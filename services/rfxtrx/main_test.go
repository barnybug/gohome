package rfxtrx

import (
	"fmt"
	"github.com/barnybug/gohome/config"
	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
	"time"

	"github.com/barnybug/gorfxtrx"
)

func ExampleTranslateCommand() {
	ev := pubsub.Parse(`{"device": "light.glowworm", "timestamp": "2014-03-13 19:40:58.368298", "state": true, "topic": "command"}`)
	services.Config = config.ExampleConfig
	pkt, err := translateCommands(ev)
	fmt.Println(err)
	fmt.Printf("%+v\n", pkt)
	// Output:
	// <nil>
	// &{typeId:0 SequenceNumber:0 HouseCode:74565 UnitCode:3 command:1 Level:0}
}

func ExampleTranslatePacketStatus() {
	pkt, _ := gorfxtrx.Parse([]byte{0x0d, 0x01, 0x00, 0x01, 0x02, 0x53, 0x3e, 0x00, 0x0c, 0x2f, 0x01, 0x01, 0x00, 0x00})
	ev := translatePacket(pkt)
	fmt.Println(ev)
	// Output:
	// <nil>
}

func ExampleTranslatePacketX10() {
	pkt, _ := gorfxtrx.Parse([]byte{0x07, 0x10, 0x00, 0x2a, 0x45, 0x05, 0x01, 0x70})
	ev := translatePacket(pkt)
	loc, _ := time.LoadLocation("UTC")
	ev.Timestamp = time.Date(2014, 1, 2, 3, 4, 5, 987654321, loc)
	fmt.Println(ev)
	// Output:
	// {"command":"on","group":"e","origin":"rfxtrx","source":"e05","timestamp":"2014-01-02 03:04:05.987654","topic":"x10"}
}

func ExampleTranslatePacketHE() {
	pkt, _ := gorfxtrx.Parse([]byte{0x0b, 0x11, 0x00, 0x2a, 0x01, 0x23, 0x45, 0x67, 0x05, 0x02, 0x08, 0x70})
	ev := translatePacket(pkt)
	loc, _ := time.LoadLocation("UTC")
	ev.Timestamp = time.Date(2014, 1, 2, 3, 4, 5, 987654321, loc)
	fmt.Println(ev)
	// Output:
	// {"command":"set level","origin":"rfxtrx","source":"12345675","timestamp":"2014-01-02 03:04:05.987654","topic":"homeeasy"}
}

func ExampleTranslatePacketChime() {
	pkt, _ := gorfxtrx.Parse([]byte{0x07, 0x16, 0x00, 0x06, 0x00, 0x7a, 0x01, 0x70})
	ev := translatePacket(pkt)
	loc, _ := time.LoadLocation("UTC")
	ev.Timestamp = time.Date(2014, 1, 2, 3, 4, 5, 987654321, loc)
	fmt.Println(ev)
	// Output:
	//{"battery":0,"chime":1,"command":"on","origin":"rfxtrx","source":"byronsx.007a","timestamp":"2014-01-02 03:04:05.987654","topic":"chime"}
}
