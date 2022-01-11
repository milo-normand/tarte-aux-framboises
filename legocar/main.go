package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/milo-normand/tarte-aux-framboises/legohat"
	"gobot.io/x/gobot"
	"gobot.io/x/gobot/platforms/joystick"
	"gobot.io/x/gobot/platforms/raspi"
)

const (
	maxAngle = 90
	maxSpeed = 100
)

type directionController struct {
	stickValue    int16
	listener      chan int
	lastDirection int
	lastUpdate    time.Time
	done          chan os.Signal
}

func (d *directionController) driveUpdates() {
	ticker := time.NewTicker(50 * time.Millisecond)

	for {
		select {
		case <-d.done:
			close(d.listener)
			return
		case t := <-ticker.C:
			convertedAngle := float64(d.stickValue) / -32768.0 * float64(maxAngle)

			if abs(int(convertedAngle)-d.lastDirection) > 3 || (t.Sub(d.lastUpdate) > 1*time.Second && abs(int(convertedAngle)-d.lastDirection) > 0) {
				d.listener <- int(convertedAngle)
				d.lastDirection = int(convertedAngle)
			}
		}
	}
}

func abs(a int) (ab int) {
	if a < 0 {
		return a * -1
	}

	return a
}

type directionUpdater struct {
	directionMotor *legohat.LegoHatMotorDriver
	input          chan int
}

func (u *directionUpdater) updateDirection() {
	for v := range u.input {
		log.Printf("Adjusting front motor to angle %d\n", v)

		_, err := u.directionMotor.RunToAngle(v)
		if err != nil {
			log.Printf("error setting angle: %s", err.Error())
		}
	}
}

func main() {
	err := connectController()
	if err != nil {
		log.Fatal(err)
	}

	r := raspi.NewAdaptor()
	hat := legohat.NewAdaptor(r)
	motor := legohat.NewLegoMotorDriver(hat, legohat.PortA)
	direction := legohat.NewLegoMotorDriver(hat, legohat.PortB)
	light := legohat.NewLegoLightDriver(hat, legohat.PortC)
	powerSensor := legohat.NewLegoHatPowerSensorDriver(hat, legohat.WithLowPowerThreshold(7.2), legohat.WithNotificationInterval(10*time.Second))
	joystickAdaptor := joystick.NewAdaptor()
	ctrl := joystick.NewDriver(joystickAdaptor, "dualshock4")

	directionUpdater := directionUpdater{
		directionMotor: direction,
		input:          make(chan int),
	}

	directionCtrl := directionController{
		listener:   directionUpdater.input,
		stickValue: 0,
		done:       make(chan os.Signal),
	}

	controllerLoopDone := make(chan os.Signal)
	signal.Notify(directionCtrl.done, os.Interrupt, syscall.SIGTERM)
	signal.Notify(controllerLoopDone, os.Interrupt, syscall.SIGTERM)
	go reconnectController(controllerLoopDone)

	work := func() {
		log.Printf("Started lego hat")
		direction.RunToAngle(0, legohat.WithSpeed(100))
		direction.SetBias(0.5)
		motor.SetPLimit(1.0)
		light.Blink(500*time.Millisecond, 4*time.Second)

		state, err := direction.GetState()
		if err != nil {
			log.Printf("error getting state: %s", err.Error())
		} else {
			log.Printf("Current state: %s\n", state)
		}

		go directionCtrl.driveUpdates()
		go directionUpdater.updateDirection()

		powerSensor.On(string(legohat.PowerUpdateEvent), func(data interface{}) {
			if val, ok := data.(float64); !ok {
				log.Printf("Error receiving power update, invalid format: %v", data)
			} else {
				log.Printf("Received power update: %.2f V", val)
			}
		})

		powerSensor.On(string(legohat.LowPowerEvent), func(data interface{}) {
			if val, ok := data.(float64); !ok {
				log.Printf("Error receiving low power update, invalid format: %v", data)
			} else {
				log.Printf("Low power update: %.2f V", val)
			}
		})

		ctrl.On(joystick.RightX, func(data interface{}) {
			fmt.Println("right_x", data)

			if val, ok := data.(int16); !ok {
				log.Printf("error reading int16 value from %v\n", data)
			} else {
				directionCtrl.stickValue = val
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
		[]gobot.Device{motor, direction, ctrl, light, powerSensor},
		work,
	)

	robot.Start()
}

func reconnectController(done chan os.Signal) {
	for {
		select {
		case <-done:
			log.Printf("Terminating controller connection loop")
			return
		default:
			connectController()
		}
	}
}

func connectController() (err error) {
	out, err := exec.Command("bluetoothctl connect D0:BC:C1:CF:D5:81").Output()
	if err != nil {
		return err
	}

	if !strings.Contains(string(out), "Connection successful") {
		fmt.Printf("controller connection failed, got output: %s", out)
	}

	return nil
}
