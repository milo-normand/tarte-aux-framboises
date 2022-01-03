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

	l.registration.toDevice <- []byte(fmt.Sprintf("port %d ; select ; echo 0\r", l.registration.id))
	l.registration.toDevice <- []byte(fmt.Sprintf("list\r"))

	for {
		select {
		case e := <-l.registration.fromDevice:
			log.Printf("Received message on port %d: %v\n", l.registration.id, e)
			if e.msgType == ConnectedMessage {
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

	l.TurnOn(0)

	close(l.registration.toDevice)

	time.Sleep(10 * time.Millisecond)
	log.Printf("Waiting for 10 millis for shutdown write to complete...\n")
	return nil
}

// Name returns the ButtonDrivers name
func (l *LegoHatMotorDriver) Name() string { return l.name }

// SetName sets the ButtonDrivers name
func (l *LegoHatMotorDriver) SetName(n string) { l.name = n }

func (l *LegoHatMotorDriver) Type() string { return l.registration.deviceType.String() }

func (l *LegoHatMotorDriver) Connection() gobot.Connection { return l.connection }

func (l *LegoHatMotorDriver) TurnOn(speed int) (err error) {
	if speed < -100 || speed > 100 {
		return fmt.Errorf("invalid speed, must be between -100 and 100 but was %d", speed)
	}

	l.registration.toDevice <- []byte(fmt.Sprintf("port %d ; combi 0 1 0 2 0 3 0 ; select 0 ; pid %d 0 0 s1 1 0 0.003 0.01 0 100; set %d\r", l.registration.id, l.registration.id, speed))
	return nil
}

func (l *LegoHatMotorDriver) TurnOff() {
	l.registration.toDevice <- []byte(fmt.Sprintf("port %d ; off\r", l.registration.id))
}
