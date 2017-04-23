package main

import (
	"fmt"
	"os"

	"github.com/barnybug/miflora"
)

func checkError(err error) {
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: miflora MAC HCI_DEVICE\n\neg: miflora C4:7C:8D:XX:XX:XX hci0")
		os.Exit(1)
	}

	mac := os.Args[1]
	adapter := os.Args[2]
	fmt.Println("Reading miflora...")
	dev := miflora.NewMiflora(mac, adapter)

	firmware, err := dev.ReadFirmware()
	checkError(err)
	fmt.Printf("Firmware: %+v\n", firmware)

	sensors, err := dev.ReadSensors()
	checkError(err)
	fmt.Printf("Sensors: %+v\n", sensors)
}
