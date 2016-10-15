// Package spi provides low level control over the linux spi bus.
//
// Before usage you should load the spidev kenel module
//
//      sudo modprobe spidev
package spi

import "os"
import "fmt"
import "bytes"
import "syscall"
import "unsafe"

const (
	spiIOCWrMode        = 0x40016B01
	spiIOCWrBitsPerWord = 0x40016B03
	spiIOCWrMaxSpeedHz  = 0x40046B04

	spiIOCRdMode        = 0x80016B01
	spiIOCRdBitsPerWord = 0x80016B03
	spiIOCRdMaxSpeedHz  = 0x80046B04

	spiIOCMessage0    = 1073769216 //0x40006B00
	spiIOCIncrementor = 2097152    //0x200000

	defaultDelayms  = 0
	defaultSPIBPW   = 8
	defaultSPISpeed = 10000000
)

const (
	spiCpha = 0x01
	spiCpol = 0x02

	// SPIMode0 represents the mode0 operation (CPOL=0 CPHA=0) of spi.
	SPIMode0 = (0 | 0)

	// SPIMode1 represents the mode0 operation (CPOL=0 CPHA=1) of spi.
	SPIMode1 = (0 | spiCpha)

	// SPIMode2 represents the mode0 operation (CPOL=1 CPHA=0) of spi.
	SPIMode2 = (spiCpol | 0)

	// SPIMode3 represents the mode0 operation (CPOL=1 CPHA=1) of spi.
	SPIMode3 = (spiCpol | spiCpha)
)

// SPI represents a SPI-Device
// It does always write and read at the same time. If you write to the device,
// the same amount of bytes is read and buffered.
type SPI struct {
	rc      *os.File
	buf     bytes.Buffer
	speedHz uint32
}

// spiIocTransfer is used internaly ro receive data from ioctl
type spiIOCTransfer struct {
	txBuf uint64
	rxBuf uint64

	length      uint32
	speedHz     uint32
	delayus     uint16
	bitsPerWord uint8
	csChange    uint8
	txNbits     uint8
	rxNbits     uint8
	pad         uint16
}

// New initializes and opens an SPI-Device
func New(bus byte, channel byte, mode byte, speed uint32) (*SPI, error) {
	f, err := os.OpenFile(fmt.Sprintf("/dev/spidev%d.%d", bus, channel), os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}

	ret := new(SPI)
	ret.rc = f

	err = ret.setup(mode, speed)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (spi *SPI) setup(mode byte, speed uint32) error {
	var err error
	err = spi.ioctl(spiIOCWrMode, uintptr(unsafe.Pointer(&mode)))
	if err != nil {
		return err
	}
	err = spi.ioctl(spiIOCRdMode, uintptr(unsafe.Pointer(&mode)))
	if err != nil {
		return err
	}
	var bits byte = 8
	err = spi.ioctl(spiIOCWrBitsPerWord, uintptr(unsafe.Pointer(&bits)))
	if err != nil {
		return err
	}
	err = spi.ioctl(spiIOCRdBitsPerWord, uintptr(unsafe.Pointer(&bits)))
	if err != nil {
		return err
	}
	err = spi.ioctl(spiIOCWrMaxSpeedHz, uintptr(unsafe.Pointer(&speed)))
	if err != nil {
		return err
	}
	err = spi.ioctl(spiIOCRdMaxSpeedHz, uintptr(unsafe.Pointer(&speed)))
	if err != nil {
		return err
	}
	return nil
}

// Write data to device. Receives data and stores it in internal buffer.
func (spi *SPI) Write(buf []byte) (int, error) {
	length := len(buf)
	var dataCarrier spiIOCTransfer
	rBuffer := make([]byte, length)

	dataCarrier.length = uint32(length)
	dataCarrier.txBuf = uint64(uintptr(unsafe.Pointer(&buf[0])))
	dataCarrier.rxBuf = uint64(uintptr(unsafe.Pointer(&rBuffer[0])))
	dataCarrier.speedHz = spi.speedHz
	dataCarrier.delayus = defaultDelayms
	dataCarrier.bitsPerWord = defaultSPIBPW

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, spi.rc.Fd(), uintptr(spiIOCMessageN(1)), uintptr(unsafe.Pointer(&dataCarrier)))
	if errno != 0 {
		err := syscall.Errno(errno)
		return 0, err
	}
	return spi.buf.Write(rBuffer)
}

// Read can only succeed if Write was called first.
func (spi *SPI) Read(p []byte) (int, error) {
	return spi.buf.Read(p)
}

// Perform an SPI transaction of write, following by read of same size back into same buffer.
func (spi *SPI) Xfer(buf []byte) error {
	length := len(buf)
	var dataCarrier spiIOCTransfer

	dataCarrier.length = uint32(length)
	dataCarrier.txBuf = uint64(uintptr(unsafe.Pointer(&buf[0])))
	dataCarrier.rxBuf = uint64(uintptr(unsafe.Pointer(&buf[0])))
	dataCarrier.speedHz = spi.speedHz
	dataCarrier.delayus = defaultDelayms
	dataCarrier.bitsPerWord = defaultSPIBPW

	err := spi.ioctl(uintptr(spiIOCMessageN(1)), uintptr(unsafe.Pointer(&dataCarrier)))
	return err
}

// Reset clears the internal buffer.
func (spi *SPI) Reset() {
	spi.buf.Reset()
}

// Close will close the filehandle.
func (spi *SPI) Close() error {
	return spi.rc.Close()
}

func (spi *SPI) setMode(mode uint8) error {
	err := spi.ioctl(spiIOCWrMode, uintptr(unsafe.Pointer(&mode)))
	if err != nil {
		return err
	}
	return nil
}

// setSpeed will try to set the clock frequency of the spi bus.
func (spi *SPI) setSpeed(speed uint32) error {
	var tspeed uint32 = defaultSPISpeed
	if speed > 0 {
		tspeed = uint32(speed)
	}

	err := spi.ioctl(spiIOCWrMaxSpeedHz, uintptr(unsafe.Pointer(&tspeed)))
	if err != nil {
		return err
	}
	spi.speedHz = tspeed

	return nil
}

func (spi *SPI) ioctl(a2, a3 uintptr) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, spi.rc.Fd(), a2, a3)
	if errno != 0 {
		err := syscall.Errno(errno)
		return err
	}
	return nil
}

func spiIOCMessageN(n uint32) uint32 {
	return (spiIOCMessage0 + (n * spiIOCIncrementor))
}
