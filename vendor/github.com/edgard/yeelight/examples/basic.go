package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/edgard/yeelight"
)

func main() {
	fmt.Println("Discovering lights...")
	lights, err := yeelight.Discover(2 * time.Second)
	if err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(len(lights))

	for i := range lights {
		go func(light *yeelight.Light) {
			fmt.Printf("Turn light %s on with a 300ms transition duration\n", light.ID)
			light.PowerOn(300)
			time.Sleep(2 * time.Second)

			fmt.Printf("Set brightness of the light %s to 10%% with a 300ms transition duration\n", light.ID)
			light.SetBrightness(10, 300)
			time.Sleep(2 * time.Second)

			fmt.Printf("Set RGB color of the light %s to blue without a transition duration\n", light.ID)
			light.SetRGB(0, 0, 255, 0)
			time.Sleep(2 * time.Second)

			fmt.Printf("Set color temperature of the light %s to 2400K with a 300ms transition duration\n", light.ID)
			light.SetTemp(2400, 300)
			time.Sleep(2 * time.Second)

			fmt.Printf("Turn light %s off with a 300ms transition duration\n", light.ID)
			light.PowerOff(300)

			wg.Done()
		}(&lights[i])
	}

	wg.Wait()
	fmt.Println("Done.")
}
