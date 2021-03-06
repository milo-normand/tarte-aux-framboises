package main

import (
	"fmt"
	"log"
	"os"

	"github.com/briandowns/openweathermap"
	"github.com/milo-normand/tarte-aux-framboises/legohat"
	"gobot.io/x/gobot"
	"gobot.io/x/gobot/platforms/raspi"
)

func main() {
	apiKey := os.Getenv("OWM_API_KEY")
	if apiKey == "" {
		fmt.Printf("Missing OWM_API_KEY env variable")
		os.Exit(-1)
	}

	weather, err := openweathermap.NewCurrent("C", "en", apiKey)
	if err != nil {
		fmt.Printf("Unable to get weather from open weather: %s", err.Error())
		os.Exit(-1)
	}

	log.Printf("Weather is %v\n", weather.Main)

	r := raspi.NewAdaptor()
	lego := legohat.NewLegoHatDriver(r)

	work := func() {
		log.Printf("Started lego hat")
	}

	robot := gobot.NewRobot("legobot",
		[]gobot.Connection{r},
		[]gobot.Device{lego},
		work,
	)

	robot.Start()
}
