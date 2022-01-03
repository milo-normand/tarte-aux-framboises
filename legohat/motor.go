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

// LegoHatMotorDriver Represents a lego hat motor driver
type LegoHatMotorDriver struct {
	name         string
	Active       bool
	halt         chan bool
	connection   gobot.Connection
	adaptor      *Adaptor
	registration *deviceRegistration

	gobot.Eventer
}

// NewLegoMotorDriver returns a new LegoHatDriver
func NewLegoMotorDriver(a *Adaptor, portID LegoHatPortID) *LegoHatMotorDriver {
	b := &LegoHatMotorDriver{
		name:       gobot.DefaultName(fmt.Sprintf("LegoHat %s", Motor)),
		adaptor:    a,
		connection: a,
		Active:     false,
		Eventer:    gobot.NewEventer(),
		halt:       make(chan bool),
	}

	r := b.adaptor.registerDevice(portID, Motor)
	b.registration = r
	// b.AddEvent(ButtonPush)
	// b.AddEvent(ButtonRelease)
	// b.AddEvent(Error)

	return b
}

// Start starts the LegoHatMotorDriver
func (l *LegoHatMotorDriver) Start() (err error) {
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
func (l *LegoHatMotorDriver) Halt() (err error) {
	log.Printf("Halting %s (%s)...", l.registration.class, l.registration.deviceType)

	l.registration.toDevice <- []byte(fmt.Sprintf("port %d ; pwm ; coast ; off \r", l.registration.id))
	l.registration.toDevice <- []byte(fmt.Sprintf("port %d ; select ; echo 0\r", l.registration.id))

	close(l.registration.toDevice)

	log.Printf("Waiting for disconnection or timeout...\n")
	for {
		select {
		case e := <-l.registration.fromDevice:
			if e.msgType != DisconnectedMessage {
				log.Printf("Disconnected device %s successfully", l.registration.class)
				return nil
			}
		case <-time.After(5 * time.Second):
			log.Printf("timed out after 5 seconds...")
			return fmt.Errorf("timed out waiting to disconnect device %s on port %d", l.registration.class, l.registration.id)
		}
	}
}

// Name returns the ButtonDrivers name
func (l *LegoHatMotorDriver) Name() string { return l.name }

// SetName sets the ButtonDrivers name
func (l *LegoHatMotorDriver) SetName(n string) { l.name = n }

func (l *LegoHatMotorDriver) Type() string { return l.registration.deviceType.String() }

func (l *LegoHatMotorDriver) Connection() gobot.Connection { return l.connection }
