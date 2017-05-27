# miflora

Go library/command line tool to read Mi Flora bluetooth sensors.

![Xiaomi Flora](https://github.com/barnybug/miflora/raw/master/miflora.jpg "Xiaomi Flora")

## Install

	$ go get github.com/barnybug/miflora/cmd/miflora

## Run

	$ miflora

## Requirements

You need bluez-utils installed for the `gatttool` utility to read bluetooth
attributes.

## Caveats

No retries are attempted when reading fails - an error is returned. User
implementation detail. :-)

`gatttool` has been deprecated and new versions of bluez-utils no longer ships
with it. The alternative is to interface directly with the bluetooth stack.
