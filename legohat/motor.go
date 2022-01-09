package legohat

import (
	"context"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"gobot.io/x/gobot"
)

const (
	defaultSpeed  = 20
	defaultPLimit = float64(0.7)
	defaultBias   = float64(0.3)
)

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
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	for _, d := range l.devices {
		registration := l.adaptor.awaitMessage(d.id, ConnectedMessage)

		d.toDevice <- []byte(fmt.Sprintf("port %d ; select ; echo 0\r", d.id))
		d.toDevice <- []byte(fmt.Sprintf("list\r"))

		log.Printf("Waiting for %s to connect on port %d...\n", Motor, d.id)

		_, err := d.waitForEventOnDevice(ctx, ConnectedMessage, registration.conduit)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *deviceRegistration) waitForEventOnDevice(ctx context.Context, awaitedMsgType DeviceMessageType, conduit <-chan DeviceEvent) (rawData []byte, err error) {
	select {
	case e := <-conduit:
		log.Printf("Received message on port %d: %v\n", d.id, e)
		switch e.msgType {
		case awaitedMsgType:
			log.Printf("Got awaited message %s on %s device at port %d", awaitedMsgType, d.class, d.id)
			return e.data, nil
		case TimeoutMessage:
			log.Printf("Got awaited message %s on %s device at port %d", timeoutMessage, d.class, d.id)
			return e.data, fmt.Errorf("received timeout from %s device on port %d", d.class, d.id)
		}
	case <-ctx.Done():
		return nil, fmt.Errorf("timed out waiting for message %s for device %s on port %d", awaitedMsgType, d.class, d.id)
	}

	return nil, fmt.Errorf("unreachable code reached")
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
		d.toDevice <- []byte(fmt.Sprintf("port %d ; combi 0 1 0 2 0 3 0 ; pid %d 0 0 s1 1 0 0.003 0.01 0 100; set %d\r", d.id, d.id, speed))
		d.currentMode = 0
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
	done = make(chan struct{}, 1)

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

	direction := 1.0
	if runSpec.speed < 0 {
		runSpec.speed = -1 * runSpec.speed
		direction = -1.0
	}

	position, err := l.GetPosition()
	if err != nil {
		return nil, err
	}

	targetPosition := ((float64(degrees) * direction) + float64(position)) / 360.0
	currentDegree := float64(position) / 360.0

	// TODO: understand where the multiplication factor comes from
	actualSpeedPerSecond := float64(runSpec.speed) * 0.05
	durationInSeconds := math.Abs((targetPosition - currentDegree) / actualSpeedPerSecond)
	log.Printf("Duration of degree rotation: %.2f\n", durationInSeconds)
	timeoutDuration := time.Millisecond * time.Duration((500 + int(math.Ceil(durationInSeconds)*1000)))

	ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()
	defer func() {
		<-time.After(time.Millisecond * 200)
		l.TurnOff()
		done <- struct{}{}
	}()

	for _, d := range l.devices {
		registration := l.adaptor.awaitMessage(d.id, RampDoneMessage)

		d.toDevice <- []byte(fmt.Sprintf("port %d ; combi 0 1 0 2 0 3 0 ; pid %d 0 1 s4 0.0027777778 0 5 0 .1 3 ; set ramp %.2f %.2f %.2f 0\r", d.id, d.id, currentDegree, targetPosition, durationInSeconds))

		_, err := d.waitForEventOnDevice(ctx, RampDoneMessage, registration.conduit)
		if err != nil {
			return nil, err
		}

		d.currentMode = 0
	}

	return done, nil
}

type rotationMethod string

const (
	shortest         rotationMethod = "shortest"
	clockwise        rotationMethod = "clockwise"
	counterClockwise rotationMethod = "counterClockwise"
)

func (l *LegoHatMotorDriver) RunToAngle(angle int, opts ...RunOption) (done chan struct{}, err error) {
	done = make(chan struct{}, 1)

	runSpec := runSpec{
		speed: 100,
	}

	for _, apply := range opts {
		apply(&runSpec)
	}

	err = runSpec.Validate()
	if err != nil {
		return nil, err
	}

	if runSpec.speed < 0 {
		return nil, fmt.Errorf("speed must be between 0 and 100")
	}

	if angle < -180 || angle > 180 {
		return nil, fmt.Errorf("angle must be between -180 and 180")
	}

	return l.runToAngle(angle, shortest, opts...)
}

func (l *LegoHatMotorDriver) runToAngle(angle int, method rotationMethod, opts ...RunOption) (done chan struct{}, err error) {
	done = make(chan struct{}, 1)
	runSpec := runSpec{
		speed: defaultSpeed,
	}

	for _, apply := range opts {
		apply(&runSpec)
	}

	state, err := l.GetState()
	if err != nil {
		return nil, err
	}

	log.Printf("Current state is %s\n", state)

	angleDiff := (angle-state.absolutePosition+180)%360 - 180
	newPosition := float64(state.position+angleDiff) / 360.0

	// clockwiseDiff := (angle - state.absolutePosition) % 360
	// counterDiff := (state.absolutePosition - angle) % 360

	// direction := 1
	// if angleDiff > 0 {
	// 	direction = 1
	// }

	// TODO: implement clockwise and counterclockwise
	// switch method {
	// case clockwise:
	// 	newPosition = (state.position + clockwiseDiff)
	// }

	pos := state.position / 360.0
	speed := float64(runSpec.speed) * 0.05
	durationInSeconds := math.Abs((float64(newPosition) - float64(state.position)) / speed)
	timeoutDuration := time.Millisecond * time.Duration((500 + int(math.Ceil(durationInSeconds)*1000)))

	ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()
	defer func() {
		l.TurnOff()
		done <- struct{}{}
	}()

	for _, d := range l.devices {
		registration := l.adaptor.awaitMessage(d.id, RampDoneMessage)

		d.toDevice <- []byte(fmt.Sprintf("port %d ; combi 0 1 0 2 0 3 0 ; pid %d 0 1 s4 0.0027777778 0 5 0 .1 3 ; set ramp %d %d %.2f 0\r", d.id, d.id, pos, newPosition, durationInSeconds))

		_, err := d.waitForEventOnDevice(ctx, RampDoneMessage, registration.conduit)
		if err != nil {
			return nil, err
		}

		d.currentMode = 0
	}

	return done, nil
}

func (l *LegoHatMotorDriver) GetPosition() (pos int, err error) {
	s, err := l.GetState()
	if err != nil {
		return 0, err
	}

	return s.position, nil
}

func (l *LegoHatMotorDriver) GetAbsolutePosition() (absPos int, err error) {
	s, err := l.GetState()
	if err != nil {
		return 0, err
	}

	return s.absolutePosition, nil
}

func (l *LegoHatMotorDriver) GetSpeed() (speed int, err error) {
	s, err := l.GetState()
	if err != nil {
		return 0, err
	}

	return s.speed, nil
}

type MotorState struct {
	speed            int
	absolutePosition int
	position         int
}

func (m *MotorState) String() string {
	return fmt.Sprintf("speed: %d, absolutePosition: %d, position: %d", m.speed, m.absolutePosition, m.position)
}

func (l *LegoHatMotorDriver) GetState() (state *MotorState, err error) {
	// TODO: validate the current mode before running this. But what's a simple mode?
	primary := l.devices[0]

	registration := l.adaptor.awaitMessage(primary.id, DataMessage)

	// TODO: review the combi stuff since that's more of a hack/guess at the moment
	primary.toDevice <- []byte(fmt.Sprintf("port %d ; combi 0 1 0 2 0 3 0 ; selonce %d\r", primary.id, primary.currentMode))

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
	defer cancel()

	rawData, err := primary.waitForEventOnDevice(ctx, DataMessage, registration.conduit)
	if err != nil {
		return nil, err
	}

	data := strings.Trim(string(rawData), " ")
	parts := strings.Fields(data)

	if len(parts) != 3 {
		return nil, fmt.Errorf("expected 3 integer values but got %d: %s", len(parts), data)
	}

	speed, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse speed from %s: %s", data, err.Error())
	}

	position, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse position from %s: %s", data, err.Error())
	}

	absPos, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse absolute position from %s: %s", data, err.Error())
	}

	state = &MotorState{
		speed:            int(speed),
		absolutePosition: int(absPos),
		position:         int(position),
	}

	return state, nil
}

func (l *LegoHatMotorDriver) RunForDuration(duration time.Duration, opts ...RunOption) (done chan struct{}, err error) {
	done = make(chan struct{}, 1)

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
	defer func() {
		l.TurnOff()
		done <- struct{}{}
	}()

	for _, d := range l.devices {
		registration := l.adaptor.awaitMessage(d.id, PulseDoneMessage)

		d.toDevice <- []byte(fmt.Sprintf("port %d ; combi 0 1 0 2 0 3 0 ; pid %d 0 0 s1 1 0 0.003 0.01 0 100; set pulse %d 0.0 %.2f 0\r", d.id, d.id, runSpec.speed, duration.Seconds()))

		_, err := d.waitForEventOnDevice(ctx, PulseDoneMessage, registration.conduit)
		if err != nil {
			return nil, err
		}

		d.currentMode = 0
	}

	return done, nil
}
