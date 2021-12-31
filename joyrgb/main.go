package main

import (
	"fmt"
	"math"
	"strings"

	"gobot.io/x/gobot"
	"gobot.io/x/gobot/drivers/gpio"
	"gobot.io/x/gobot/platforms/joystick"
	"gobot.io/x/gobot/platforms/raspi"
)

func main() {
	r := raspi.NewAdaptor()
	joystickAdaptor := joystick.NewAdaptor()
	stick := joystick.NewDriver(joystickAdaptor, "custom.json")
	stick.Start()

	led := gpio.NewRgbLedDriver(r, "11", "12", "13")
	led.Start()

	red := 255.0
	green := 255.0
	blue := 255.0

	activeColors := make(map[string]bool, 0)

	work := func() {
		// buttons
		stick.On(joystick.SquarePress, func(data interface{}) {
			fmt.Println("square_press")
		})
		stick.On(joystick.SquareRelease, func(data interface{}) {
			fmt.Println("square_release")
		})
		stick.On(joystick.TrianglePress, func(data interface{}) {
			activeColors["green"] = !activeColors["green"]
			fmt.Printf("triangle_press, toggling green: %t\n", activeColors["green"])
		})
		stick.On(joystick.TriangleRelease, func(data interface{}) {
			fmt.Println("triangle_release")
		})
		stick.On(joystick.CirclePress, func(data interface{}) {
			activeColors["red"] = !activeColors["red"]
			fmt.Printf("circle_release, toggling red: %t\n", activeColors["red"])
		})
		stick.On(joystick.CircleRelease, func(data interface{}) {
			fmt.Println("circle_release")
		})
		stick.On(joystick.XPress, func(data interface{}) {
			activeColors["blue"] = !activeColors["blue"]
			fmt.Printf("x_press, toggling blue: %t\n", activeColors["blue"])
		})
		stick.On(joystick.XRelease, func(data interface{}) {
			fmt.Println("x_release")
		})
		stick.On(joystick.StartPress, func(data interface{}) {
			fmt.Println("start_press")
		})
		stick.On(joystick.StartRelease, func(data interface{}) {
			fmt.Println("start_release")
		})
		stick.On(joystick.SelectPress, func(data interface{}) {
			fmt.Println("select_press")
		})
		stick.On(joystick.SelectRelease, func(data interface{}) {
			fmt.Println("select_release")
		})

		// joysticks
		stick.On(joystick.LeftX, func(data interface{}) {
			fmt.Println("left_x", data)
		})
		stick.On(joystick.LeftY, func(data interface{}) {
			fmt.Println("left_y", data)
			if val, ok := data.(int16); !ok {
				fmt.Printf("error reading int16 value from %v\n", data)
			} else {
				val := 255 - math.Abs(float64(val)/32768.0)*255

				enabledColors := make([]string, 0, 3)
				for k, e := range activeColors {
					switch {
					case k == "red" && e:
						enabledColors = append(enabledColors, k)
						red = val
					case k == "green" && e:
						enabledColors = append(enabledColors, k)
						green = val
					case k == "blue" && e:
						enabledColors = append(enabledColors, k)
						blue = val
					}
				}

				fmt.Printf("Controlling colors: %s\nRed: %d, Green: %d, Blue: %d\n", strings.Join(enabledColors, ", "), byte(red), byte(green), byte(blue))
				led.SetRGB(byte(red), byte(green), byte(blue))
			}
		})
		stick.On(joystick.RightX, func(data interface{}) {
			fmt.Println("right_x", data)
		})
		stick.On(joystick.RightY, func(data interface{}) {
			fmt.Println("right_y", data)
		})

		// triggers
		stick.On(joystick.R1Press, func(data interface{}) {
			fmt.Println("R1Press", data)
		})
		stick.On(joystick.R2Press, func(data interface{}) {
			fmt.Println("R2Press", data)
		})
		stick.On(joystick.L1Press, func(data interface{}) {
			fmt.Println("L1Press", data)
		})
		stick.On(joystick.L2Press, func(data interface{}) {
			fmt.Println("L2Press", data)
		})
	}

	robot := gobot.NewRobot("blinkBot",
		[]gobot.Connection{r, joystickAdaptor},
		[]gobot.Device{led, stick},
		work,
	)

	robot.Start()
}
