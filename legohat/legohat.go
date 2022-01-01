package legohat

import (
	_ "embed"
	"log"
	"strings"

	"gobot.io/x/gobot"
	"gobot.io/x/gobot/drivers/gpio"
)

//go:embed data/version
var version string

// LegoHatDriver Represents a lego hat driver
type LegoHatDriver struct {
	name         string
	Active       bool
	DefaultState int
	pin          string
	halt         chan bool
	connection   gpio.DigitalReader
	gobot.Eventer
}

// NewLegoHatDriver returns a new LegoHatDriver
func NewLegoHatDriver(a gpio.DigitalReader) *LegoHatDriver {
	b := &LegoHatDriver{
		name:         gobot.DefaultName("LegoHat"),
		connection:   a,
		Active:       false,
		DefaultState: 0,
		Eventer:      gobot.NewEventer(),
		halt:         make(chan bool),
	}

	// b.AddEvent(ButtonPush)
	// b.AddEvent(ButtonRelease)
	// b.AddEvent(Error)

	return b
}

// Start starts the LegoHatDriver

// Emits the Events:
// 	Push int - On button push
//	Release int - On button release
//	Error error - On button error
func (l *LegoHatDriver) Start() (err error) {
	err = initialize("/dev/serial0", strings.Replace(version, "\n", "", -1))
	if err != nil {
		return err
	}
	state := l.DefaultState
	log.Printf("Default state: %s\n", state)
	// go func() {
	// 	for {
	// 		newValue, err := l.connection.DigitalRead(l.Pin())
	// 		if err != nil {
	// 			l.Publish(Error, err)
	// 		} else if newValue != state && newValue != -1 {
	// 			state = newValue
	// 			l.update(newValue)
	// 		}
	// 		select {
	// 		case <-time.After(l.interval):
	// 		case <-l.halt:
	// 			return
	// 		}
	// 	}
	// }()
	return
}

// Halt stops polling the button for new information
func (l *LegoHatDriver) Halt() (err error) {
	return nil
}

// Name returns the ButtonDrivers name
func (l *LegoHatDriver) Name() string { return l.name }

// SetName sets the ButtonDrivers name
func (l *LegoHatDriver) SetName(n string) { l.name = n }

// Pin returns the ButtonDrivers pin
func (l *LegoHatDriver) Pin() string { return l.pin }

// Connection returns the ButtonDrivers Connection
func (l *LegoHatDriver) Connection() gobot.Connection { return l.connection.(gobot.Connection) }
