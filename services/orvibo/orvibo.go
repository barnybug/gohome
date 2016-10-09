package orvibo

// go-orvibo is a lightweight package that is used to control a variety of Orvibo products
// including the AllOne IR / 433mhz blaster and the S10 / S20 sockets

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"

	"net"
	"strings"
)

type ReadyMessage struct{}

type NewDeviceMessage struct {
	Device *Device
}

type SubscribeAckMessage struct {
	Device *Device
}

type StateChangedMessage struct {
	Device *Device
	State  bool
}

// Device is info about the type of device that's been detected (socket, allone etc.)
type Device struct {
	ID         int          // The ID of our socket
	Name       string       // The name of our item
	DeviceType int          // What type of device this is. See the const below for valid types
	IP         *net.UDPAddr // The IP address of our item
	MACAddress string       // The MAC Address of our item. Necessary for controlling the S10 / S20 / AllOne
	Subscribed bool         // Have we subscribed to this item yet? Doing so lets us control
	Queried    bool         // Have we queried this item for it's name and details yet?
	State      bool         // Is the item turned on or off? Will always be "false" for the AllOne, which doesn't do states, just IR & 433
}

const (
	UNKNOWN = -1 + iota // UNKNOWN is obviously a device that isn't implemented or is unknown. iota means add 1 to the next const, so SOCKET = 0, ALLONE = 1 etc.
	SOCKET              // SOCKET is an S10 / S20 powerpoint socket
)

// Events holds the events we'll be passing back to our calling code.
var Events = make(chan interface{}, 10) // Events is our events channel which will notify calling code that we have an event happening
var devices = map[string]*Device{}
var macPadding = "202020202020" // This is padding for the MAC Address. It appears often, so we define it here for brevity
var conn *net.UDPConn           // UDP Connection

const (
	magicWord    uint16 = 0x6864 // hd
	cmdDiscover  uint16 = 0x7161 // qa
	cmdSubscribe uint16 = 0x636c // cl
	cmdSetState  uint16 = 0x6463 // dc
)

// ===============
// Exported Events
// ===============

// Start listens on UDP port 10000 for incoming messages.
func Start() error {
	udpAddr, err := net.ResolveUDPAddr("udp4", ":10000") // Get our address ready for listening
	if err != nil {
		return err
	}

	conn, err = net.ListenUDP("udp", udpAddr) // Now we listen on the address we just resolved
	if err != nil {
		return err
	}

	// Run read loop forever
	go func() {
		for {
			checkForMessages()
		}
	}()

	// Hand a message back to our calling code. Because it's not about a particular device, we just pass back an empty AllOne
	passMessage(&ReadyMessage{})
	return nil
}

// Discover is a function that broadcasts 686400067161 over the network in order to find unpaired networks
func Discover() error {
	_, err := broadcastMessage(cmdDiscover, []byte{})
	return err
}

// Subscribe loops over all the devices we know about, and asks for control (subscription)
func Subscribe(device *Device) error {
	// We send a message to each socket. reverseMAC takes a MAC address and reverses each pair (e.g. AC CF 23 becomes CA FC 32)
	return sendMessage(cmdSubscribe, reverseMAC(device.MACAddress)+macPadding, device)
}

// Query asks all the sockets we know about, for their names. Current state is sent on Subscription confirmation, not here
func Query() error {
	var err error

	// for k := range devices { // Loop over all sockets we know about
	// 	if devices[k].Queried == false && devices[k].Subscribed == true { // If we've subscribed but not queried..
	// 		err = sendMessage("rt", "0000000004000000000000", devices[k])
	// 		// success, err = SendMessage("6864001D7274"+devices[k].MACAddress+macPadding+"0000000004000000000000", devices[k])
	// 	}
	// }
	// passMessage("query", nil)
	return err
}

// CheckForMessages does what it says on the tin -- checks for incoming UDP messages
func checkForMessages() error { // Now we're checking for messages
	var msg []byte     // Holds the incoming message
	var buf [1024]byte // We want to get 1024 bytes of messages (is this enough? Need to check!)
	var err error

	n, addr, _ := conn.ReadFromUDP(buf[0:]) // Read 1024 bytes from the buffer
	ip, _ := getLocalIP()                   // Get our local IP
	if n > 0 && addr.IP.String() != ip {    // If we've got more than 0 bytes and it's not from us
		msg = buf[0:n]                                     // n is how many bytes we grabbed from UDP
		err = handleMessage(hex.EncodeToString(msg), addr) // Hand it off to our handleMessage func. We pass on the message and the address (for replying to messages)
	}

	return err
}

// SetState sets the state of a socket, given its MAC address
func SetState(device *Device, state bool) error {
	statebit := "00"
	if state {
		statebit = "01"
	}
	err := sendMessage(cmdSetState, "00000000"+statebit, device)
	return err
}

func encodePacket(commandID uint16, payload []byte) []byte {
	// header (2) + length (2) + command(2) + payload (variable)
	var length uint16 = 6 + uint16(len(payload))
	buffer := new(bytes.Buffer)
	binary.Write(buffer, binary.BigEndian, magicWord)
	binary.Write(buffer, binary.BigEndian, length)
	binary.Write(buffer, binary.BigEndian, commandID)
	buffer.Write(payload)
	return buffer.Bytes()
}

func encodeDevicePacket(commandID uint16, msg string, device *Device) []byte {
	payload := device.MACAddress + macPadding + msg
	buf, _ := hex.DecodeString(payload)
	return encodePacket(commandID, buf)
}

// sendMessage pieces together a lot of the standard Orvibo packet, including correct packet length.
// It ultimately uses sendMessageRaw to sent out the packet
func sendMessage(commandID uint16, msg string, device *Device) error {
	packet := encodeDevicePacket(commandID, msg, device)
	return sendMessageRaw(packet, device)
}

// sendMessageRaw is the heart of our library. Sends UDP messages to specified IP addresses
func sendMessageRaw(buf []byte, device *Device) error {
	// Resolve our address, ready for sending data
	udpAddr, err := net.ResolveUDPAddr("udp4", device.IP.String())
	if err != nil {
		return err
	}

	// Actually write the data and send it off
	_, err = conn.WriteToUDP(buf, udpAddr)
	// If we've got an error
	if err != nil {
		return err
	}

	return nil
}

// ==================
// Internal functions
// ==================

// handleMessage parses a message found by checkForMessages
func handleMessage(message string, addr *net.UDPAddr) error {
	if len(message) < 12 {
		return errors.New("Short message")
	}

	// If this is a broadcast message
	if message == "686400067161" {
		return nil
	}

	commandID := message[8:12]                  // What command we've received back
	macStart := strings.Index(message, "accf")  // Find where our MAC Address starts
	macAdd := message[macStart:(macStart + 12)] // The MAC address of the socket responding

	switch commandID {
	case "7161": // We've had a response to our broadcast message
		if strings.Index(message, "534f4330") > 0 { // Contains SOC0? It's a socket!
			device := &Device{
				Name:       "",
				DeviceType: SOCKET,
				IP:         addr,
				MACAddress: macAdd,
				Subscribed: false,
				Queried:    false,
				State:      false,
			}

			lastBit := message[(len(message) - 1):] // Get the last bit from our message. 0 or 1 for off or on
			device.State = lastBit != "0"

			if _, exists := devices[macAdd]; !exists {
				devices[macAdd] = device
				passMessage(&NewDeviceMessage{device})
			}
		}

	case "636c": // We've had confirmation of subscription

		// Sometimes we receive messages for sockets we don't know about. The WiWo
		// app does this sometimes, as it sends messages to all AllOnes it knows about,
		// regardless of whether or not they're active on the network. So we
		// check to see if the socket that needs updating exists in our list. If it doesn't,
		// we return false.
		if _, ok := devices[macAdd]; !ok {
			devices[macAdd] = &Device{MACAddress: macAdd}
		}

		device := devices[macAdd]
		device.State = message[(len(message)-1):] == "1"
		device.Subscribed = true
		passMessage(&SubscribeAckMessage{device})

	// case "7274": // We've queried our socket, this is the data back
	// 	// Our name starts after the fourth 202020202020, or 140 bytes in
	// 	strName := strings.TrimRight(message[140:172], "")

	// 	// And our name is 32 bytes long.
	// 	strDecName, _ := hex.DecodeString(strName[0:32])

	// 	// If no name has been set, we get 32 bytes of F back, so
	// 	// we create a generic name so our socket name won't be spaces
	// 	if strName == "20202020202020202020202020202020" || strName == "ffffffffffffffffffffffffffffffff" {
	// 		devices[macAdd].Name = "Socket " + macAdd

	// 	} else { // If a name WAS set
	// 		devices[macAdd].Name = string(strDecName) // Convert back to text and assign
	// 	}

	// 	passMessage("queried", devices[macAdd])

	case "7366": // Confirmation of state change

		if device, ok := devices[macAdd]; ok {
			state := message[(len(message)-1):] == "1"
			if device.State != state {
				device.State = state
				passMessage(&StateChangedMessage{device, state})
			}
		}

	default: // No message? Return true
		return nil
	}

	return nil
}

// Gets our current IP address. This is used so we can ignore messages from ourselves
func getLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}

	ifaces, _ := net.Interfaces()
	// handle err
	for _, i := range ifaces {
		addrs, _ := i.Addrs()
		// handle err
		for _, addr := range addrs {
			switch v := addr.(type) {
			case *net.IPAddr:
				return v.IP.String(), nil
			}

		}
	}

	return "", errors.New("Unable to find IP address. Ensure you're connected to a network")
}

// passMessage adds items to our Events channel so the calling code can be informed
// It's non-blocking or whatever.
func passMessage(event interface{}) bool {
	select {
	case Events <- event:
	default:
	}

	return true
}

// broadcastMessage is another core part of our code. It lets us broadcast a message to the whole network.
// It's essentially SendMessage with a IPv4 Broadcast address
func broadcastMessage(commandID uint16, payload []byte) (bool, error) {
	udpAddr, err := net.ResolveUDPAddr("udp4", net.IPv4bcast.String()+":10000")
	packet := encodePacket(commandID, payload)
	sendMessageRaw(packet, &Device{IP: udpAddr})
	if err != nil {
		return false, err
	}
	return true, nil
}

// Via http://stackoverflow.com/questions/19239449/how-do-i-reverse-an-array-in-go
// Splits up a hex string into bytes then reverses the bytes
func reverseMAC(mac string) string {
	s, _ := hex.DecodeString(mac)
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return hex.EncodeToString(s)
}
