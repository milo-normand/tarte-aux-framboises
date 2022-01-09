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
	halt       chan bool
	connection gobot.Connection
	adaptor    *Adaptor
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
		halt:       make(chan bool),
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

func (l *LegoHatLightDriver) TurnOn() {
	for _, d := range l.devices {
		d.toDevice <- []byte(fmt.Sprintf("port %d ; plimit 1 ; on\r", d.id))
	}
}

func (l *LegoHatLightDriver) TurnOff() {
	for _, d := range l.devices {
		d.toDevice <- []byte(fmt.Sprintf("port %d ; plimit 1 ; off\r", d.id))
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
