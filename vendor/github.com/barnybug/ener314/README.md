## ener314

[![Build Status](https://travis-ci.org/barnybug/ener314.svg?branch=master)](https://travis-ci.org/barnybug/ener314)
[![GoDoc](https://godoc.org/github.com/barnybug/ener314?status.svg)](http://godoc.org/github.com/barnybug/ener314)

Go library for the Energenie ENER314-RT transceiver for the Raspberry Pi:

https://energenie4u.co.uk/catalogue/product/ENER314-RT

### Supported devices

- MIHO013 eTRV Heating Valve
  https://energenie4u.co.uk/catalogue/product/MIHO013

If you send me other devices, I'll add them!

## Supported operations

- MIHO013:
  - Reading temperature
  - Set target temperature
  - Requesting voltage
  - Diagnostics
  - Exercise valve
  - Join / join response
  - Set valve state
  - Set power mode

### Installation

    go get github.com/barnybug/ener314

### Usage

See the example under cmd/ener314.

To run it:

	go install github.com/barnybug/ener314/cmd/ener314
	ener314

### Permissions

The program doesn't need root as it doesn't access GPIO pins directly. It just
needs permission to read/write the devices /dev/spidev0.1 and /dev/gpiomem.

You need the kernel modules `spi_bcm2835` and `bcm2835_gpiomem` loaded.

Then you can accomplish this with some udev rules (see examples in udev) and
creating groups 'gpio' and 'spi' and adding your current user to the groups
(don't forget to logout and back in).

### Technical details

The ENER314-RT is a HopeRF RFM69W chip connected to the Raspberry Pi SPI bus.

The datasheet for the RFM69W is here:

http://www.hoperf.com/upload/rf/RFM69W-V1.3.pdf

The library uses spidev and gpiomem to communicate with the device. gpiomem is
used to control the Reset and LED pins, and spidev handles the SPI master-
slave comms.

The radio link is 433Mhz FSK (frequency shift keying) protocol implementing
the Openthings specification:

http://www.sentec.co.uk/our-technology/micromonitor/openthings

### Other notes

The eTRVs wake to receive commands every time they transmit the temperature -
by default roughly every 5 mins. The time window to send a command to the eTRV
is just after this point. So you should transmit set target temperature, or
request diagnostics at this point in time.

### Changelog
0.1.0

- First release
