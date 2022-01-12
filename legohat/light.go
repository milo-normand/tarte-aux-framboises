package legohat

import (
	"fmt"
	"log"
	"time"

	"gobot.io/x/gobot"
)

// LegoHatLightDriver Represents a lego hat light driver
type LegoHatLightDriver struct {
	name       string
	connection gobot.Connection
	deviceDriver

	gobot.Eventer
}

type LightDriverOption func(driver *LegoHatLightDriver)

// NewLegoLightDriver returns a new LegoHatDriver
func NewLegoLightDriver(a *Adaptor, portID LegoHatPortID, opts ...LightDriverOption) (b *LegoHatLightDriver) {
	b = &LegoHatLightDriver{
		name:       gobot.DefaultName(fmt.Sprintf("LegoHat %s", Light)),
		connection: a,
		Eventer:    gobot.NewEventer(),
		deviceDriver: deviceDriver{
			adaptor: a,
			devices: make([]*deviceRegistration, 0),
		},
	}

	b.devices = append(b.devices, b.adaptor.registerDevice(portID, Light))

	for _, apply := range opts {
		apply(b)
	}

	return b
}

func (l *LegoHatLightDriver) Start() (err error) {
	err = l.waitForConnect()
	if err != nil {
		return err
	}

	return nil
}

func (l *LegoHatLightDriver) Blink(interval time.Duration, duration time.Duration) (done chan struct{}) {
	done = make(chan struct{}, 1)

	go func() {
		count := int64(duration / interval)
		if duration%interval > 0 {
			count++
		}

		for i := int64(0); i < count; i++ {
			l.blinkOnce(interval)
		}

		done <- struct{}{}
	}()

	return done
}

func (l *LegoHatLightDriver) blinkOnce(duration time.Duration, opts ...LightOption) (err error) {
	err = l.TurnOn(opts...)
	if err != nil {
		return err
	}
	time.Sleep(duration / 2)
	l.TurnOff()
	time.Sleep(duration / 2)

	return nil
}

type lightActivationSpec struct {
	level float32
}

type LightOption func(s *lightActivationSpec)

func WithLightLevel(level float32) LightOption {
	return func(s *lightActivationSpec) {
		s.level = level
	}
}

func (s *lightActivationSpec) Validate() (err error) {
	if s.level < 0 || s.level > 1.0 {
		return fmt.Errorf("light level should be between 0 and 1 but was %.2f", s.level)
	}

	return nil
}

func (l *LegoHatLightDriver) TurnOn(opts ...LightOption) (err error) {
	light := lightActivationSpec{
		level: 1.0,
	}

	for _, apply := range opts {
		apply(&light)
	}

	err = light.Validate()
	if err != nil {
		return err
	}

	for _, d := range l.devices {
		d.toDevice <- []byte(fmt.Sprintf("port %d ; plimit 1 ; set -%.4f\r", d.id, light.level))
	}

	return nil
}

func (l *LegoHatLightDriver) TurnOff() {
	for _, d := range l.devices {
		d.toDevice <- []byte(fmt.Sprintf("port %d ; plimit 1 ; set 0\r", d.id))
	}
}

func (l *LegoHatLightDriver) Halt() (err error) {
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
func (l *LegoHatLightDriver) Name() string { return l.name }

// SetName sets the ButtonDrivers name
func (l *LegoHatLightDriver) SetName(n string) { l.name = n }

// DeviceType returns the device type for the primary device only
func (l *LegoHatLightDriver) DeviceType() string { return l.devices[0].deviceType.String() }

func (l *LegoHatLightDriver) Connection() gobot.Connection { return l.connection }
