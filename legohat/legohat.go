package legohat

import (
	_ "embed"

	"gobot.io/x/gobot"
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
	connection   *Adaptor
	gobot.Eventer
}

type LegoHatPortID int

const (
	PortOne   = LegoHatPortID(0)
	PortTwo   = LegoHatPortID(1)
	PortThree = LegoHatPortID(2)
	PortFour  = LegoHatPortID(3)
)

func hatPorts() (ports []LegoHatPortID) {
	return []LegoHatPortID{PortOne, PortTwo, PortThree, PortFour}
}

// NewLegoMotorDriver returns a new LegoHatDriver
func NewLegoMotorDriver(a *Adaptor, portID LegoHatPortID) *LegoHatDriver {
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
	// TODO
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
