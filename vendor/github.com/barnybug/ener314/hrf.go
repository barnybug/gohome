package ener314

import (
	"bytes"
	"encoding/hex"
	"time"

	"github.com/barnybug/ener314/rpio"
	"github.com/barnybug/ener314/spi"
)

type HRF struct {
	spi *spi.SPI
}

const (
	SEED_PID               = 0x01
	MANUF_SENTEC           = 0x01
	PRODUCT_SENTEC_DEFAULT = 0x01
	MESSAGE_BUF_SIZE       = 66
	MAX_FIFO_SIZE          = 66
	TRUE                   = 1
	FALSE                  = 0

	ADDR_FIFO         = 0x00
	ADDR_OPMODE       = 0x01 // Operating modes
	ADDR_REGDATAMODUL = 0x02
	ADDR_BITRATEMSB   = 0x03
	ADDR_BITRATELSB   = 0x04
	ADDR_FDEVMSB      = 0x05
	ADDR_FDEVLSB      = 0x06
	ADDR_FRMSB        = 0x07
	ADDR_FRMID        = 0x08
	ADDR_FRLSB        = 0x09
	ADDR_OSC1         = 0x0A
	ADDR_AFCCTRL      = 0x0B
	ADDR_RESERVED     = 0x0C
	ADDR_LISTEN1      = 0x0D
	ADDR_LISTEN2      = 0x0E
	ADDR_LISTEN3      = 0x0F
	ADDR_VERSION      = 0x10
	ADDR_PALEVEL      = 0x11
	ADDR_PARAMP       = 0x12
	ADDR_OCP          = 0x13
	// Reserved 0x14-0x17
	ADDR_LNA           = 0x18
	ADDR_RXBW          = 0x19
	ADDR_AFCBW         = 0x1A
	ADDR_OOKPEAK       = 0x1B
	ADDR_OOKAVG        = 0x1C
	ADDR_OOKFIX        = 0x1D
	ADDR_AFCFEI        = 0x1E
	ADDR_AFCMSB        = 0x1F
	ADDR_AFCLSB        = 0x20
	ADDR_FEIMSB        = 0x21
	ADDR_FEILSB        = 0x22
	ADDR_RSSICONFIG    = 0x23
	ADDR_RSSIVALUE     = 0x24
	ADDR_DIOMAPPING1   = 0x25
	ADDR_DIOMAPPING2   = 0x26
	ADDR_IRQFLAGS1     = 0x27
	ADDR_IRQFLAGS2     = 0x28
	ADDR_RSSITHRESH    = 0x29
	ADDR_RXTIMEOUT1    = 0x2A
	ADDR_RXTIMEOUT2    = 0x2B
	ADDR_PREAMBLEMSB   = 0x2C
	ADDR_PREAMBLELSB   = 0x2D
	ADDR_SYNCCONFIG    = 0x2E
	ADDR_SYNCVALUE1    = 0x2F
	ADDR_SYNCVALUE2    = 0X30
	ADDR_SYNCVALUE3    = 0x31
	ADDR_SYNCVALUE4    = 0X32
	ADDR_SYNCVALUE5    = 0X33
	ADDR_SYNCVALUE6    = 0X34
	ADDR_SYNCVALUE7    = 0X35
	ADDR_SYNCVALUE8    = 0X36
	ADDR_PACKETCONFIG1 = 0X37
	ADDR_PAYLOADLEN    = 0X38
	ADDR_NODEADDRESS   = 0X39
	ADDR_BROADCASTADRS = 0X3A
	ADDR_AUTOMODES     = 0X3B
	ADDR_FIFOTHRESH    = 0X3C
	ADDR_PACKETCONFIG2 = 0X3D
	// AES Key 0x3e-0x4d
	ADDR_TEMP1 = 0X4E
	ADDR_TEMP2 = 0X4F
	// 0x50- test registers
	ADDR_TESTLNA  = 0X58
	ADDR_TESTPA1  = 0X5A
	ADDR_TESTPA2  = 0X5C
	ADDR_TESTDAGC = 0X6F
	ADDR_TESTAFC  = 0X71

	MASK_REGDATAMODUL_OOK = 0x08
	MASK_REGDATAMODUL_FSK = 0x00
	MASK_WRITE_DATA       = 0x80
	MASK_MODEREADY        = 0x80
	MASK_FIFONOTEMPTY     = 0x40
	MASK_FIFOLEVEL        = 0x20
	MASK_FIFOOVERRUN      = 0x10
	MASK_PACKETSENT       = 0x08
	MASK_TXREADY          = 0x20
	MASK_PACKETMODE       = 0x60
	MASK_MODULATION       = 0x18
	MASK_PAYLOADRDY       = 0x04
	MASK_RSSIDONE         = 0x02
	MASK_RSSISTART        = 0x01
	MASK_TEMPMEASSTART    = 0x08
	MASK_TEMPMEASRUNNING  = 0x04

	/* Precise register description can be found on:
	 * www.hoperf.com/upload/rf/RFM69W-V1.3.pdf
	 * on page 63 - 74
	 */
	MODE_STANDBY         = 0x04        // Standby
	MODE_TRANSMITTER     = 0x0C        // Transmitter
	MODE_RECEIVER        = 0x10        // Receiver
	VAL_REGDATAMODUL_FSK = 0x00        // Modulation scheme FSK
	VAL_REGDATAMODUL_OOK = 0x08        // Modulation scheme OOK
	VAL_FDEVMSB30        = 0x01        // frequency deviation 5kHz 0x0052 -> 30kHz 0x01EC
	VAL_FDEVLSB30        = 0xEC        // frequency deviation 5kHz 0x0052 -> 30kHz 0x01EC
	VAL_FRMSB434         = 0x6C        // carrier freq -> 434.3MHz 0x6C9333
	VAL_FRMID434         = 0x93        // carrier freq -> 434.3MHz 0x6C9333
	VAL_FRLSB434         = 0x33        // carrier freq -> 434.3MHz 0x6C9333
	VAL_FRMSB433         = 0x6C        // carrier freq -> 433.92MHz 0x6C7AE1
	VAL_FRMID433         = 0x7A        // carrier freq -> 433.92MHz 0x6C7AE1
	VAL_FRLSB433         = 0xE1        // carrier freq -> 433.92MHz 0x6C7AE1
	VAL_AFCCTRLS         = 0x00        // standard AFC routine
	VAL_AFCCTRLI         = 0x20        // improved AFC routine
	VAL_LNA50            = 0x08        // LNA input impedance 50 ohms
	VAL_LNA50G           = 0x0E        // LNA input impedance 50 ohms, LNA gain -> 48db
	VAL_LNA200           = 0x88        // LNA input impedance 200 ohms
	VAL_RXBW60           = 0x43        // channel filter bandwidth 10kHz -> 60kHz  page:26
	VAL_RXBW120          = 0x41        // channel filter bandwidth 120kHz
	VAL_AFCFEIRX         = 0x04        // AFC is performed each time RX mode is entered
	VAL_RSSITHRESH220    = 0xDC        // RSSI threshold 0xE4 -> 0xDC (220)
	VAL_PREAMBLELSB3     = 0x03        // preamble size LSB 3
	VAL_PREAMBLELSB5     = 0x05        // preamble size LSB 5
	VAL_SYNCCONFIG2      = 0x88        // Size of the Synch word = 2 (SyncSize + 1)
	VAL_SYNCCONFIG4      = 0x98        // Size of the Synch word = 4 (SyncSize + 1)
	VAL_SYNCVALUE1FSK    = 0x2D        // 1st byte of Sync word
	VAL_SYNCVALUE2FSK    = 0xD4        // 2nd byte of Sync word
	VAL_SYNCVALUE1OOK    = 0x80        // 1nd byte of Sync word
	VAL_PACKETCONFIG1FSK = 0xA2        // Variable length, Manchester coding, Addr must match NodeAddress
	VAL_PACKETCONFIG1OOK = 0           // Fixed length, no Manchester coding
	VAL_PAYLOADLEN255    = 0xFF        // max Length in RX, not used in Tx
	VAL_PAYLOADLEN64     = 0x40        // max Length in RX, not used in Tx
	VAL_PAYLOADLEN_OOK   = (13 + 8*17) // Payload Length
	VAL_NODEADDRESS01    = 0x04        // Node address used in address filtering
	VAL_FIFOTHRESH1      = 0x81        // Condition to start packet transmission: at least one byte in FIFO
	VAL_FIFOTHRESH30     = 0x1E        // Condition to start packet transmission: wait for 30 bytes in FIFO

	GreenLed = 27 // GPIO 13
	RedLed   = 22 // GPIO 15
	ResetPin = 25 // GPIO 22
)

func NewHRF() (*HRF, error) {
	dev, err := spi.New(0, 1, spi.SPIMode0, 9600000)
	if err != nil {
		return nil, err
	}
	err = rpio.Open()
	if err != nil {
		return nil, err
	}
	inst := &HRF{spi: dev}
	return inst, err
}

type Cmd struct {
	addr byte
	val  byte
}

func (self *HRF) Close() {
	rpio.Close()
}

func (self *HRF) Reset() error {
	// light both leds whilst resetting
	green := rpio.Pin(GreenLed)
	red := rpio.Pin(RedLed)
	green.Output()
	red.Output()
	green.High()
	red.High()

	pin := rpio.Pin(ResetPin)
	pin.Output()

	pin.High()
	time.Sleep(100 * time.Millisecond)
	pin.Low()
	time.Sleep(100 * time.Millisecond)

	green.Low()
	red.Low()

	return nil
}

func (self *HRF) ConfigFSK() error {
	regSetup := []Cmd{
		{ADDR_REGDATAMODUL, VAL_REGDATAMODUL_FSK}, // modulation scheme FSK
		{ADDR_FDEVMSB, VAL_FDEVMSB30},             // frequency deviation 5kHz 0x0052 -> 30kHz 0x01EC
		{ADDR_FDEVLSB, VAL_FDEVLSB30},             // frequency deviation 5kHz 0x0052 -> 30kHz 0x01EC
		{ADDR_FRMSB, VAL_FRMSB434},                // carrier freq -> 434.3MHz 0x6C9333
		{ADDR_FRMID, VAL_FRMID434},                // carrier freq -> 434.3MHz 0x6C9333
		{ADDR_FRLSB, VAL_FRLSB434},                // carrier freq -> 434.3MHz 0x6C9333
		{ADDR_AFCCTRL, VAL_AFCCTRLS},              // standard AFC routine
		{ADDR_LNA, VAL_LNA50},                     // 200ohms, gain by AGC loop -> 50ohms
		{ADDR_RXBW, VAL_RXBW60},                   // channel filter bandwidth 10kHz -> 60kHz  page:26
		//{ADDR_AFCFEI, 		VAL_AFCFEIRX},		// AFC is performed each time rx mode is entered
		//{ADDR_RSSITHRESH, 	VAL_RSSITHRESH220},	// RSSI threshold 0xE4 -> 0xDC (220)
		{ADDR_PREAMBLELSB, VAL_PREAMBLELSB3},       // preamble size LSB -> 3
		{ADDR_SYNCCONFIG, VAL_SYNCCONFIG2},         // Size of the Synch word = 2 (SyncSize + 1)
		{ADDR_SYNCVALUE1, VAL_SYNCVALUE1FSK},       // 1st byte of Sync word
		{ADDR_SYNCVALUE2, VAL_SYNCVALUE2FSK},       // 2nd byte of Sync word
		{ADDR_PACKETCONFIG1, VAL_PACKETCONFIG1FSK}, // Variable length, Manchester coding, Addr must match NodeAddress
		{ADDR_PAYLOADLEN, VAL_PAYLOADLEN64},        // max Length in RX, not used in Tx
		{ADDR_NODEADDRESS, VAL_NODEADDRESS01},      // Node address used in address filtering
		{ADDR_FIFOTHRESH, VAL_FIFOTHRESH1},         // Condition to start packet transmission: at least one byte in FIFO
		{ADDR_OPMODE, MODE_RECEIVER},               // Operating mode to Receiver
	}
	for _, cmd := range regSetup {
		err := self.regW(cmd.addr, cmd.val)
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *HRF) WaitFor(addr byte, mask byte, val bool) {
	cnt := 0
	for {
		cnt += 1 // Uncomment to wait in a loop finite amount of time
		if cnt > 100000 {
			panic("timeout inside a while for addr")
			// log4c_category_warn(hrflog, "timeout inside a while for addr %02x\n", addr);
			// break
		}
		ret := self.regR(addr)
		if val {
			if (ret & mask) == mask {
				break
			}
		} else {
			if (ret & mask) == 0 {
				break
			}
		}
	}
}

func (self *HRF) ClearFifo() {
	for {
		if self.regR(ADDR_IRQFLAGS2)&MASK_FIFONOTEMPTY == 0 {
			break
		}
		self.regR(ADDR_FIFO)
	}
}

func (self *HRF) GetVersion() byte {
	return self.regR(ADDR_VERSION)
}

func (self *HRF) GetRSSI() float32 {
	self.regW(ADDR_RSSICONFIG, MASK_RSSISTART)
	self.WaitFor(ADDR_RSSICONFIG, MASK_RSSIDONE, true)
	return -float32(self.regR(ADDR_RSSIVALUE)) / 2
}

func (self *HRF) GetTemperature() int {
	// must be in standby
	self.regW(ADDR_OPMODE, MODE_STANDBY)
	// request temperature
	self.regW(ADDR_TEMP1, MASK_TEMPMEASSTART)
	// wait for measuring to finish running
	self.WaitFor(ADDR_TEMP1, MASK_TEMPMEASRUNNING, false)
	// approx figure - needs calibration
	temp := 160 - int(self.regR(ADDR_TEMP2))
	// switch back to receiver
	self.regW(ADDR_OPMODE, MODE_RECEIVER)
	return temp
}

func (self *HRF) ReceiveFSKMessage() *Message {
	if self.regR(ADDR_IRQFLAGS2)&MASK_PAYLOADRDY == MASK_PAYLOADRDY {
		// light green whilst receiving
		green := rpio.Pin(GreenLed)
		green.High()

		length := self.regR(ADDR_FIFO)
		data := make([]byte, length)
		for i := 0; i < int(length); i += 1 {
			data[i] = self.regR(ADDR_FIFO)
		}
		green.Low()

		cryptPacket(data)
		logs(LOG_TRACE, "<-", hex.EncodeToString(data)) // log decrypted packet
		message, err := decodePacket(data)
		if err != nil {
			logs(LOG_ERROR, "Error:", err)
			return nil
		}
		return message
	}

	return nil
}

func (self *HRF) SendFSKMessage(msg *Message) error {
	data := encodeMessage(msg)
	logs(LOG_TRACE, "->", hex.EncodeToString(data)) // log decrypted packet
	encryptData(data)

	var buf bytes.Buffer
	buf.WriteByte(MASK_WRITE_DATA) // address
	buf.WriteByte(byte(len(data))) // packet length
	buf.Write(data)                // packet

	// light red whilst transmitting
	red := rpio.Pin(RedLed)
	red.High()

	// switch to transmission mode
	err := self.regW(ADDR_OPMODE, MODE_TRANSMITTER)
	if err != nil {
		return err
	}
	self.WaitFor(ADDR_IRQFLAGS1, MASK_MODEREADY|MASK_TXREADY, true)

	data = buf.Bytes()
	err = self.spi.Xfer(data)
	if err != nil {
		return err
	}

	// wait until the packet is sent
	self.WaitFor(ADDR_IRQFLAGS2, MASK_PACKETSENT, true)
	// HRF_assert_reg_val(ADDR_IRQFLAGS2, MASK_FIFONOTEMPTY|MASK_FIFOOVERRUN, FALSE, "are all bytes sent?")

	// switch back to receiver mode
	err = self.regW(ADDR_OPMODE, MODE_RECEIVER)
	if err != nil {
		return err
	}
	self.WaitFor(ADDR_IRQFLAGS1, MASK_MODEREADY, true)

	red.Low()
	logs(LOG_TRACE, "Sent:", msg)

	return nil
}

func (self *HRF) regR(addr byte) byte {
	buf := []byte{addr & 0x7f, 0}
	self.spi.Xfer(buf)
	return buf[1]
}

func (self *HRF) regW(addr byte, val byte) error {
	buf := []byte{addr | MASK_WRITE_DATA, val}
	err := self.spi.Xfer(buf)
	return err
}
