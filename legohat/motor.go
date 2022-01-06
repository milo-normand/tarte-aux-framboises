package legohat

import (
	"context"
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
	name       string
	Active     bool
	halt       chan bool
	connection gobot.Connection
	adaptor    *Adaptor
	devices    []*deviceRegistration

	gobot.Eventer
}

type MotorDriverOption func(driver *LegoHatMotorDriver)

func WithAdditionalMotor(portID LegoHatPortID) func(driver *LegoHatMotorDriver) {
	return func(driver *LegoHatMotorDriver) {
		driver.devices = append(driver.devices, driver.adaptor.registerDevice(portID, Motor))
	}
}

// NewLegoMotorDriver returns a new LegoHatDriver
func NewLegoMotorDriver(a *Adaptor, portID LegoHatPortID, opts ...MotorDriverOption) *LegoHatMotorDriver {
	b := &LegoHatMotorDriver{
		name:       gobot.DefaultName(fmt.Sprintf("LegoHat %s", Motor)),
		adaptor:    a,
		connection: a,
		Active:     false,
		Eventer:    gobot.NewEventer(),
		halt:       make(chan bool),
		devices:    make([]*deviceRegistration, 0),
	}

	b.devices = append(b.devices, b.adaptor.registerDevice(portID, Motor))

	for _, apply := range opts {
		apply(b)
	}
	// b.AddEvent(ButtonPush)
	// b.AddEvent(ButtonRelease)
	// b.AddEvent(Error)

	return b
}

// Start starts the LegoHatMotorDriver
func (l *LegoHatMotorDriver) Start() (err error) {
	err = l.waitForConnect()
	if err != nil {
		return err
	}

	// TODO: include the device specifications like number of modes as device state to handle things like resets and validations accordingly
	err = l.resetModes()
	if err != nil {
		return err
	}

	err = l.setPLimit(defaultPLimit)
	if err != nil {
		return err
	}

	err = l.setBias(defaultBias)
	if err != nil {
		return err
	}

	return nil
}

func (l *LegoHatMotorDriver) waitForConnect() (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, d := range l.devices {
		d.toDevice <- []byte(fmt.Sprintf("port %d ; select ; echo 0\r", d.id))
		d.toDevice <- []byte(fmt.Sprintf("list\r"))

		log.Printf("Waiting for %s to connect on port %d...\n", Motor, d.id)

		err := waitForEventOnDevice(ctx, ConnectedMessage, d)
		if err != nil {
			return err
		}
	}

	return nil
}

// TODO figure out how to use context so that we have a single timeout set and then have multiple goroutines waiting for
// a signal
func waitForEventOnDevice(ctx context.Context, messageType DeviceMessageType, d *deviceRegistration) (err error) {
	select {
	case e := <-d.fromDevice:
		log.Printf("Received message on port %d: %v\n", d.id, e)
		if e.msgType == messageType {
			log.Printf("Got awaited message %s on %s device at port %d", messageType, d.class, d.id)
			return nil
		}
	case <-ctx.Done():
		return fmt.Errorf("timed out waiting for message %s for device %s on port %d", messageType, d.class, d.id)
	}

	return nil
}

func (l *LegoHatMotorDriver) setPLimit(plimit float64) (err error) {
	if plimit < 0 || plimit > 1 {
		return fmt.Errorf("plimit should be between 0 and 1 but was %.2f", plimit)
	}

	for _, d := range l.devices {
		d.toDevice <- []byte(fmt.Sprintf("port %d ; plimit %.2f\r", d.id, plimit))
	}

	return nil
}

func (l *LegoHatMotorDriver) setBias(bias float64) (err error) {
	if bias < 0 || bias > 1 {
		return fmt.Errorf("bias should be between 0 and 1 but was %.2f", bias)
	}

	for _, d := range l.devices {
		d.toDevice <- []byte(fmt.Sprintf("port %d ; bias %.2f\r", d.id, bias))
	}

	return nil
}

func (l *LegoHatMotorDriver) setPWM(pwm float64) (err error) {
	if pwm < 0 || pwm > 1 {
		return fmt.Errorf("pwm should be between 0 and 1 but was %.2f", pwm)
	}

	for _, d := range l.devices {
		d.toDevice <- []byte(fmt.Sprintf("port %d ; pwm ; set %.2f\r", d.id, pwm))
	}

	return nil
}

func (l *LegoHatMotorDriver) resetModes() (err error) {
	for _, d := range l.devices {
		d.toDevice <- []byte(fmt.Sprintf("port %d ; combi 1\r", d.id))
		d.toDevice <- []byte(fmt.Sprintf("port %d ; combi 2\r", d.id))
		d.toDevice <- []byte(fmt.Sprintf("port %d ; combi 3\r", d.id))
	}

	return nil
}

// Halt releases the connection to the port
func (l *LegoHatMotorDriver) Halt() (err error) {
	for _, d := range l.devices {
		log.Printf("Halting %s (%s)...", d.class, d.deviceType)

		l.TurnOff()

		close(d.toDevice)
	}

	time.Sleep(10 * time.Millisecond)
	log.Printf("Waiting for 10 millis for shutdown write to complete...\n")
	return nil
}

// Name returns the ButtonDrivers name
func (l *LegoHatMotorDriver) Name() string { return l.name }

// SetName sets the ButtonDrivers name
func (l *LegoHatMotorDriver) SetName(n string) { l.name = n }

// DeviceType returns the device type for the primary device only
func (l *LegoHatMotorDriver) DeviceType() string { return l.devices[0].deviceType.String() }

func (l *LegoHatMotorDriver) Connection() gobot.Connection { return l.connection }

func (l *LegoHatMotorDriver) TurnOn(speed int) (err error) {
	if speed < -100 || speed > 100 {
		return fmt.Errorf("invalid speed, must be between -100 and 100 but was %d", speed)
	}

	for _, d := range l.devices {
		d.toDevice <- []byte(fmt.Sprintf("port %d ; combi 0 1 0 2 0 3 0 ; select 0 ; pid %d 0 0 s1 1 0 0.003 0.01 0 100; set %d\r", d.id, d.id, speed))
	}

	return nil
}

func (l *LegoHatMotorDriver) TurnOff() {
	for _, d := range l.devices {
		d.toDevice <- []byte(fmt.Sprintf("port %d ; coast\r", d.id))
	}
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

func (l *LegoHatMotorDriver) RunForDuration(duration time.Duration, opts ...RunOption) (done chan struct{}, err error) {
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

	ctx, cancel := context.WithTimeout(context.Background(), duration+time.Second*1)
	defer cancel()
	defer l.TurnOff()

	for _, d := range l.devices {
		d.toDevice <- []byte(fmt.Sprintf("port %d ; combi 0 1 0 2 0 3 0 ; select 0 ; pid %d 0 0 s1 1 0 0.003 0.01 0 100; set pulse %d 0.0 %.2f 0\r", d.id, d.id, runSpec.speed, duration.Seconds()))

		err := waitForEventOnDevice(ctx, PulseDoneMessage, d)
		if err != nil {
			return nil, err
		}
	}

	return done, nil
}
