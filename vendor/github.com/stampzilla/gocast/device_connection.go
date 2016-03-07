package gocast

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"

	"github.com/davecgh/go-spew/spew"
	"github.com/gogo/protobuf/proto"
	"github.com/stampzilla/gocast/api"
	"github.com/stampzilla/gocast/events"
	"github.com/stampzilla/gocast/handlers"
)

func (d *Device) reader() {
	for {
		packet := d.wrapper.Read()

		if packet == nil {
			for _, subscription := range d.subscriptions {
				subscription.Handler.Disconnect()
			}

			d.subscriptions = make([]*Subscription, 0)

			event := events.Disconnected{}
			d.Dispatch(event)
			return
		}

		message := &api.CastMessage{}
		err := proto.Unmarshal(*packet, message)
		if err != nil {
			log.Fatalf("Failed to unmarshal CastMessage: %s", err)
		}

		//spew.Dump("Message!", message)

		var headers handlers.Headers

		err = json.Unmarshal([]byte(*message.PayloadUtf8), &headers)

		if err != nil {
			log.Fatalf("Failed to unmarshal message: %s", err)
		}

		catched := false
		for _, subscription := range d.subscriptions {
			if subscription.Receive(message, &headers) {
				catched = true
			}
		}

		if !catched {
			fmt.Println("LOST MESSAGE:")
			spew.Dump(message)
		}
	}
}

func (d *Device) Connect() error {
	//log.Printf("connecting to %s:%d ...", d.ip, d.port)

	var err error
	d.conn, err = tls.Dial("tcp", fmt.Sprintf("%s:%d", d.ip, d.port), &tls.Config{
		InsecureSkipVerify: true,
	})

	if err != nil {
		return fmt.Errorf("Failed to connect to Chromecast. Error:%s", err)
	}

	event := events.Connected{}
	d.Dispatch(event)

	d.wrapper = NewPacketStream(d.conn)
	go d.reader()

	d.Subscribe("urn:x-cast:com.google.cast.tp.connection", d.connectionHandler)
	d.Subscribe("urn:x-cast:com.google.cast.tp.heartbeat", d.heartbeatHandler)
	d.Subscribe("urn:x-cast:com.google.cast.receiver", d.receiverHandler)

	return nil
}

func (d *Device) Disconnect() {
	d.conn.Close()
	d.conn = nil
}

func (d *Device) Send(urn, sourceId, destinationId string, payload interface{}) error {
	if p, ok := payload.(handlers.Headers); ok {
		d.id++
		p.RequestId = &d.id

		payload = p
	}

	payloadJson, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Failed to json.Marshal: ", err)
		return err
	}
	payloadString := string(payloadJson)

	message := &api.CastMessage{
		ProtocolVersion: api.CastMessage_CASTV2_1_0.Enum(),
		SourceId:        &sourceId,
		DestinationId:   &destinationId,
		Namespace:       &urn,
		PayloadType:     api.CastMessage_STRING.Enum(),
		PayloadUtf8:     &payloadString,
	}

	proto.SetDefaults(message)

	data, err := proto.Marshal(message)
	if err != nil {
		fmt.Println("Failed to proto.Marshal: ", err)
		return err
	}

	//spew.Dump("Writing", message)

	if d.conn == nil {
		return fmt.Errorf("We are disconnected, cannot send!")
	}

	_, err = d.wrapper.Write(&data)

	return err
}
