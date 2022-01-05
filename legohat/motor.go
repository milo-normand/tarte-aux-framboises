package legohat

import (
	_ "embed"
	"fmt"
	"log"
	"time"

	"gobot.io/x/gobot"
)

const (
	defaultSpeed  = 20
	defaultPLimit = float64(0.7)
	defaultBias   = float64(0.3)
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

	// TODO: include the device specifications like number of modes as device state to handle things like resets and validations accordingly
	l.resetModes()
	l.setPLimit(defaultPLimit)
	l.setBias(defaultBias)
}

func (l *LegoHatMotorDriver) setPLimit(plimit float64) (err error) {
	if plimit < 0 || plimit > 1 {
		return fmt.Errorf("plimit should be between 0 and 1 but was %.2f", plimit)
	}

	l.registration.toDevice <- []byte(fmt.Sprintf("port %d ; plimit %.2f\r", l.registration.id, plimit))
}

func (l *LegoHatMotorDriver) setBias(bias float64) (err error) {
	if bias < 0 || bias > 1 {
		return fmt.Errorf("bias should be between 0 and 1 but was %.2f", bias)
	}

	l.registration.toDevice <- []byte(fmt.Sprintf("port %d ; bias %.2f\r", l.registration.id, bias))

	return nil
}

func (l *LegoHatMotorDriver) setPWM(pwm float64) (err error) {
	if pwm < 0 || pwm > 1 {
		return fmt.Errorf("pwm should be between 0 and 1 but was %.2f", pwm)
	}

	l.registration.toDevice <- []byte(fmt.Sprintf("port %d ; pwm ; set %.2f\r", l.registration.id, pwm))

	return nil
}

func (l *LegoHatMotorDriver) resetModes() (err error) {
	l.registration.toDevice <- []byte(fmt.Sprintf("port %d ; combi 0\r", l.registration.id))
	l.registration.toDevice <- []byte(fmt.Sprintf("port %d ; combi 1\r", l.registration.id))
	l.registration.toDevice <- []byte(fmt.Sprintf("port %d ; combi 2\r", l.registration.id))
	l.registration.toDevice <- []byte(fmt.Sprintf("port %d ; combi 3\r", l.registration.id))
	l.registration.toDevice <- []byte(fmt.Sprintf("port %d ; combi 4\r", l.registration.id))

	return nil
}

// Halt releases the connection to the port
func (l *LegoHatMotorDriver) Halt() (err error) {
	log.Printf("Halting %s (%s)...", l.registration.class, l.registration.deviceType)

	l.TurnOff()

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
	l.registration.toDevice <- []byte(fmt.Sprintf("port %d ; coast\r", l.registration.id))
}

type runSpec struct {
	speed int
}

func (s *runSpec) Validate() (err error) {
	if s.speed < -100 || s.speed > 100 {
		return fmt.Errorf("invalid speed, must be between -100 and 100 but was %d", s.speed)
	}

	return nil
}

type RunOption func(spec *runSpec)

func WithSpeed(speed int) func(spec *runSpec) {
	return func(spec *runSpec) {
		spec.speed = speed
	}
}

func (l *LegoHatMotorDriver) RunForRotations(rotations int, opts ...RunOption) (done chan struct{}, err error) {
	return l.RunForDegrees(rotations*360, opts...)
}

func (l *LegoHatMotorDriver) RunForDegrees(degrees int, opts ...RunOption) (done chan struct{}, err error) {
	done = make(chan struct{})

	runSpec := runSpec{
		speed: defaultSpeed,
	}

	for _, apply := range opts {
		apply(&runSpec)
	}

	err = runSpec.Validate()
	if err != nil {
		return nil, err
	}

	// TODO implement this

	return done, nil
}
