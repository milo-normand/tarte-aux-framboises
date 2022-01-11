package legohat

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"gobot.io/x/gobot"
)

// LegoHatLightDriver Represents a lego hat light driver
type LegoHatPowerSensorDriver struct {
	name                 string
	halt                 chan bool
	connection           gobot.Connection
	adaptor              *Adaptor
	notificationInterval time.Duration
	lowPowerThreshold    float64
	device               *deviceRegistration

	gobot.Eventer
}

type PowerSensorEventType string

const (
	PowerFaultEvent  PowerSensorEventType = "power_fault"
	LowPowerEvent    PowerSensorEventType = "low_power"
	PowerUpdateEvent PowerSensorEventType = "power_update"
)

type PowerSensorDriverOption func(driver *LegoHatPowerSensorDriver)

func WithLowPowerThreshold(voltageThreshold float64) PowerSensorDriverOption {
	return func(driver *LegoHatPowerSensorDriver) {
		driver.lowPowerThreshold = voltageThreshold
	}
}

func WithNotificationInterval(interval time.Duration) PowerSensorDriverOption {
	return func(driver *LegoHatPowerSensorDriver) {
		driver.notificationInterval = interval
	}
}

func NewLegoHatPowerSensorDriver(a *Adaptor, opts ...PowerSensorDriverOption) (b *LegoHatPowerSensorDriver) {
	b = &LegoHatPowerSensorDriver{
		name:                 gobot.DefaultName("LegoHatPowerSensor"),
		connection:           a,
		adaptor:              a,
		Eventer:              gobot.NewEventer(),
		notificationInterval: 30 * time.Second,
		lowPowerThreshold:    6.5,
		halt:                 make(chan bool),
	}

	b.device = a.registerDevice(None, Internal)
	for _, apply := range opts {
		apply(b)
	}

	b.AddEvent(string(PowerFaultEvent))
	b.AddEvent(string(PowerUpdateEvent))
	b.AddEvent(string(LowPowerEvent))

	return b
}

func (l *LegoHatPowerSensorDriver) Start() (err error) {
	go l.pollPower()

	registration := l.adaptor.awaitAllMessages(None, PowerFaultMessage)
	go l.watchPowerFaults(registration.conduit)

	return nil
}

func (l *LegoHatPowerSensorDriver) watchPowerFaults(events chan DeviceEvent) {
	for {
		select {
		case <-events:
			log.Printf("Publishing power fault event")
			l.Publish(string(PowerFaultEvent), true)
		case <-l.halt:
			log.Printf("Stop watching for power faults\n")
			return
		}
	}
}

func (l *LegoHatPowerSensorDriver) Halt() (err error) {
	log.Printf("Halting power sensor")
	l.halt <- true

	return nil
}

func (l *LegoHatPowerSensorDriver) pollPower() {
	for {
		select {
		case <-l.halt:
			log.Printf("Stopping polling for power")
			return
		case <-time.After(l.notificationInterval):
			err := l.refreshPowerStatus()
			if err != nil {
				log.Printf("error polling for power voltage: %s", err.Error())
			}
		}
	}
}

func (l *LegoHatPowerSensorDriver) refreshPowerStatus() (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	registration := l.adaptor.awaitMessage(l.device.id, RampDoneMessage)

	l.device.toDevice <- []byte("vin\r")

	data, err := l.device.waitForEventOnDevice(ctx, PowerStatusMessage, registration.conduit)
	if err != nil {
		return err
	}

	val, err := strconv.ParseFloat(string(data), 64)
	if err != nil {
		return fmt.Errorf("can't parse voltage value: %w", err)
	}

	l.Eventer.Publish(string(PowerUpdateEvent), val)

	if val < l.lowPowerThreshold {
		l.Eventer.Publish(string(LowPowerEvent), val)
	}

	return nil
}

func (l *LegoHatPowerSensorDriver) Name() string { return l.name }

func (l *LegoHatPowerSensorDriver) SetName(n string) { l.name = n }

func (l *LegoHatPowerSensorDriver) Connection() gobot.Connection { return l.connection }
