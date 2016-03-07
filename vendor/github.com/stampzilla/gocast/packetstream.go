package gocast

import (
	"encoding/binary"
	"fmt"
	"io"
)

type packetStream struct {
	stream  io.ReadWriteCloser
	packets chan *[]byte
}

func NewPacketStream(stream io.ReadWriteCloser) *packetStream {
	wrapper := packetStream{stream, make(chan *[]byte)}
	wrapper.readPackets()

	return &wrapper
}

func (w *packetStream) readPackets() {
	var length uint32

	go func() {
		for {

			err := binary.Read(w.stream, binary.BigEndian, &length)
			if err != nil {
				//fmt.Printf("Failed binary.Read packet: %s", err)
				close(w.packets)
				return
			}

			if length > 0 {
				packet := make([]byte, length)

				i, err := w.stream.Read(packet)
				if err != nil {
					fmt.Printf("Failed to read packet: %s", err)
					continue
				}

				if i != int(length) {
					fmt.Printf("Invalid packet size. Wanted: %d Read: %d", length, i)
					continue
				}

				w.packets <- &packet
			}

		}
	}()
}

func (w *packetStream) Read() *[]byte {
	return <-w.packets
}

func (w *packetStream) Write(data *[]byte) (int, error) {

	err := binary.Write(w.stream, binary.BigEndian, uint32(len(*data)))

	if err != nil {
		err = fmt.Errorf("Failed to write packet length %d. error:%s\n", len(*data), err)
		return 0, err
	}

	return w.stream.Write(*data)
}
