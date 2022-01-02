package legohat

import (
	_ "embed"
	"fmt"
	"log"
	"time"

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

	for {
		select {
		case e := <-l.registration.fromDevice:
			if e.msgType != ConnectedMessage {
				log.Printf("Device %s connected on port %d", l.registration.class, l.registration.id)
				return nil
			}
		case <-time.After(5 * time.Second):
			return fmt.Errorf("timed out waiting for connection of device %s on port %d", l.registration.class, l.registration.id)
		}
	}
}

// Halt releases the connection to the port
func (l *LegoHatDriver) Halt() (err error) {
	l.registration.toDevice <- []byte(fmt.Sprintf("port %d ; pwm ; coast ; off \r", l.registration.id))
	l.registration.toDevice <- []byte(fmt.Sprintf("port %d ; select ; echo 0\r", l.registration.id))

	defer close(l.registration.toDevice)

	for {
		select {
		case e := <-l.registration.fromDevice:
			if e.msgType != DisconnectedMessage {
				return nil
			}
		case <-time.After(5 * time.Second):
			return fmt.Errorf("timed out waiting to disconnect device %s on port %d", l.registration.class, l.registration.id)
		}
	}
}

// Name returns the ButtonDrivers name
func (l *LegoHatDriver) Name() string { return l.name }

// SetName sets the ButtonDrivers name
func (l *LegoHatDriver) SetName(n string) { l.name = n }

func (l *LegoHatDriver) Type() string { return l.registration.deviceType.String() }

func (l *LegoHatDriver) Connection() string { return l.connection.(gobot.Connection) }
