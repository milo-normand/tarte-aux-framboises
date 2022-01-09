package main

import (
	"fmt"
	"log"
	"math"

	"github.com/milo-normand/tarte-aux-framboises/legohat"
	"gobot.io/x/gobot"
	"gobot.io/x/gobot/platforms/joystick"
	"gobot.io/x/gobot/platforms/raspi"
)

const (
	maxAngle = 60
	maxSpeed = 100
)

func main() {
	r := raspi.NewAdaptor()
	hat := legohat.NewAdaptor(r)
	motor := legohat.NewLegoMotorDriver(hat, legohat.PortA)
	direction := legohat.NewLegoMotorDriver(hat, legohat.PortB)
	joystickAdaptor := joystick.NewAdaptor()
	ctrl := joystick.NewDriver(joystickAdaptor, "dualshock4")

	currentAngle := 0.

	work := func() {
		log.Printf("Started lego hat")

		ctrl.On(joystick.RightX, func(data interface{}) {
			fmt.Println("right_x", data)
			if val, ok := data.(int16); !ok {
				log.Printf("error reading int16 value from %v\n", data)
			} else {
				angle := float64(val) / 32768.0 * float64(maxAngle)
				if math.Abs(angle-currentAngle) > 5 {
					log.Printf("Adjusting front motor to degree %d", int(angle))

					_, err := direction.RunToAngle(int(angle), legohat.WithSpeed(100))
					if err != nil {
						log.Printf("error setting angle: %s", err.Error())
					}
					currentAngle = angle
				}
			}
		})
		ctrl.On(joystick.LeftY, func(data interface{}) {
			fmt.Println("left_y", data)
			if val, ok := data.(int16); !ok {
				log.Printf("error reading int16 value from %v\n", data)
			} else {
				speed := int(float64(val) / 32768.0 * float64(maxSpeed))
				log.Printf("Adjusting speed motor to %d", speed)

				err := motor.TurnOn(speed)
				if err != nil {
					log.Printf("error setting forward speed: %s", err.Error())
				}
			}
		})
	}

	robot := gobot.NewRobot("legocar",
		[]gobot.Connection{r, hat, joystickAdaptor},
		[]gobot.Device{motor, direction, ctrl},
		work,
	)

	robot.Start()
}
