package legohat

import (
	_ "embed"
	"fmt"
	"log"

	"gobot.io/x/gobot"
)

//go:embed data/version
var version string

// LegoHatDriver Represents a lego hat driver
type LegoHatDriver struct {
	name         string
	Active       bool
	halt         chan bool
	connection   *Adaptor
	registration *deviceRegistration

	gobot.Eventer
}

// NewLegoMotorDriver returns a new LegoHatDriver
func NewLegoMotorDriver(a *Adaptor, portID LegoHatPortID) *LegoHatDriver {
	b := &LegoHatDriver{
		name:       gobot.DefaultName(fmt.Sprintf("LegoHat %s", Motor)),
		connection: a,
		Active:     false,
		Eventer:    gobot.NewEventer(),
		halt:       make(chan bool),
	}

	r := b.connection.registerDevice(portID, Motor)
	b.registration = r
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
	log.Printf("Waiting for %s to connect on port %d...\n", Motor, l.registration.id)
	for e := range l.registration.fromDevice {
		if e.msgType == ConnectedMessage {
			log.Printf("Device connected")
			break
		} else {
			log.Printf("Received unexpected event: %s", e.msgType)
		}
	}

	return nil
}

// Halt releases the connection to the port
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
