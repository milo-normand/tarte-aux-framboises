package legohat

import (
	"bufio"
	_ "embed"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"go.bug.st/serial"
	"gobot.io/x/gobot"
	"gobot.io/x/gobot/drivers/gpio"
)

//go:embed data/version
var version string

//go:embed data/firmware.bin
var firmware []byte

//go:embed data/signature.bin
var signature []byte

const (
	firmwareLine   = "Firmware version: "
	bootloaderLine = "BuildHAT bootloader version"
	promptPrefix   = "BHBL>"
	doneLine       = "Done initialising ports"
)

type LegoHatPortID int

const (
	PortA = LegoHatPortID(0)
	PortB = LegoHatPortID(1)
	PortC = LegoHatPortID(2)
	PortD = LegoHatPortID(3)
)

type HatState string

const (
	otherState           = HatState("Other")
	firmwareState        = HatState("Firmware")
	needNewFirmwareState = HatState("NeedNewFirmware")
	bootloaderState      = HatState("Bootloader")
)

// Possible messages received on the serial port
const (
	connectedMessage        = "connected to active ID"
	connectedPassiveMessage = "connected to passive ID"
	disconnectedMessage     = "disconnected"
	timeoutMessage          = "timeout during data phase: disconnecting"
	nothingConnectedMessage = "no device detected"
	pulseDoneMessage        = "pulse done"
	rampDoneMessage         = "ramp done"
)

const (
	resetPinNumber    = "4"
	bootZeroPinNumber = "22"
	pinOff            = byte(0)
	pinOn             = byte(1)
)

type legoDevice interface {
	io.Closer
	Info() Device
}

type Config struct {
	serialPath string
}

// Adaptor represents a connection to a lego hat
type Adaptor struct {
	name                 string
	config               Config
	port                 serial.Port
	devices              map[LegoHatPortID]*deviceRegistration
	terminateDispatching chan bool
	toWrite              chan []byte
	digitalWriter        gpio.DigitalWriter
	eventDispatcher
}

type eventDispatcher struct {
	awaitedEvents map[eventKey]eventRegistration
	input         chan DeviceEvent
}

type eventRegistration struct {
	persistent bool
	conduit    chan DeviceEvent
}

type eventKey struct {
	msgType DeviceMessageType
	portID  LegoHatPortID
}

type Option func(c *Config)

func WithSerialPath(serialPath string) Option {
	return func(c *Config) {
		c.serialPath = serialPath
	}
}

// NewAdaptor returns a new Lego Hat Adaptor.
func NewAdaptor(w gpio.DigitalWriter, opts ...Option) *Adaptor {
	config := Config{
		serialPath: "/dev/serial0",
	}

	for _, apply := range opts {
		apply(&config)
	}

	return &Adaptor{
		name:                 gobot.DefaultName("LegoHat"),
		config:               config,
		devices:              make(map[LegoHatPortID]*deviceRegistration),
		terminateDispatching: make(chan bool),
		toWrite:              make(chan []byte),
		digitalWriter:        w,
		eventDispatcher: eventDispatcher{
			awaitedEvents: make(map[eventKey]eventRegistration),
			input:         make(chan DeviceEvent),
		},
	}
}

// Name returns the Adaptors name
func (l *Adaptor) Name() string { return l.name }

// SetName sets the Adaptors name
func (l *Adaptor) SetName(n string) { l.name = n }

// Connect connects to the joystick
func (l *Adaptor) Connect() (err error) {
	l.port, err = l.initialize(l.config.serialPath, strings.Replace(version, "\n", "", -1))
	if err != nil {
		return err
	}

	go l.inputsToEvents()
	go l.dispatchEvents()

	for _, d := range l.devices {
		log.Printf("Starting dispatching routine for device on port %d...\n", d.id)
		go l.dispatchInstructions(d.toDevice)
	}

	go l.writeInstructions()

	return nil
}

func (l *Adaptor) registerDevice(portID LegoHatPortID, deviceClass DeviceClass) (registration *deviceRegistration) {
	r := deviceRegistration{
		id:       portID,
		class:    deviceClass,
		name:     deviceClass.String(),
		toDevice: make(chan []byte),
	}
	l.devices[portID] = &r

	return &r
}

func (l *Adaptor) dispatchInstructions(in chan []byte) {
	for i := range in {
		log.Printf("Dispatching message to write: %s...\n", string(i))
		l.toWrite <- i
	}

	log.Printf("Terminated dispatching go routine\n")
}

func (l *Adaptor) writeInstructions() {
	defer log.Printf("Terminated goroutine writing instructions\n")

	for {
		select {
		case in := <-l.toWrite:
			log.Printf("Writing to serial port %s:\n\t%s", l.config.serialPath, string(in))
			l.port.Write(in)
		case <-l.terminateDispatching:
			log.Printf("Received termination signal to stop dispatching")
			return
		}
	}
}

// TODO: either return errors on a channel or handle all errors internally
func (l *Adaptor) inputsToEvents() {
	lines := make(chan string)
	go ReadPort(l.port, lines)

	for line := range lines {
		log.Printf("Got line: %s\n", line)
		if strings.HasPrefix(line, "P") {
			lineParts := strings.Split(line, ":")
			if len(lineParts) < 2 {
				log.Printf("unexpected line format with P prefix. should be P<id>: message but didn't have the ':' delimiter: %s\n", line)
				continue
			}
			identification := lineParts[0]
			rawPortID := strings.TrimPrefix(identification, "P")
			portID := rawPortID[0] - '0'
			message := strings.Trim(lineParts[1], " ")

			switch {
			// This is a case of a data message with the mode suffix after the port id
			case len(lineParts[0]) > 2:
				mode := identification[2:]

				if d, ok := l.devices[LegoHatPortID(portID)]; ok {
					log.Printf("Sending data message [%s] to listener %v...\n", DataMessage, d)

					l.eventDispatcher.input <- DeviceEvent{
						msgType: DataMessage,
						mode:    mode,
						data:    []byte(message),
					}
				}
			case strings.HasPrefix(message, connectedMessage):
				rawDeviceType := strings.TrimPrefix(message, connectedMessage)
				deviceTypeVal, err := strconv.ParseInt(strings.Trim(rawDeviceType, " "), 16, 64)
				if err != nil {
					log.Printf("unexpected device type format on line %s: %s\n", line, err.Error())
				}

				deviceType := DeviceType(deviceTypeVal)
				log.Printf("Device of type %s connected on port %d", deviceType, portID)

				if d, ok := l.devices[LegoHatPortID(portID)]; ok {
					d.deviceType = deviceType
				}

				l.eventDispatcher.input <- DeviceEvent{
					msgType: ConnectedMessage,
				}

			case strings.HasPrefix(message, disconnectedMessage):
				log.Printf("Device disconnected on port %d", portID)

				l.eventDispatcher.input <- DeviceEvent{
					msgType: DisconnectedMessage,
				}

			case strings.HasPrefix(message, timeoutMessage):
				log.Printf("Device timeout on port %d", portID)

				l.eventDispatcher.input <- DeviceEvent{
					msgType: TimeoutMessage,
				}

			case strings.HasPrefix(message, pulseDoneMessage):
				log.Printf("Pulse done message on port %d", portID)

				l.eventDispatcher.input <- DeviceEvent{
					msgType: PulseDoneMessage,
				}
			case strings.HasPrefix(message, rampDoneMessage):
				log.Printf("Ramp done message on port %d", portID)

				l.eventDispatcher.input <- DeviceEvent{
					msgType: RampDoneMessage,
				}
			}
		}
	}

	log.Printf("Terminated reading on serial port\n")
}

func ReadPort(port serial.Port, out chan string) {
	scanner := bufio.NewScanner(port)
	for scanner.Scan() {
		line := scanner.Text()

		if len(line) == 0 {
			continue
		}

		out <- line
	}

	close(out)
}

// Finalize closes connection to the lego hat
func (l *Adaptor) Finalize() (err error) {
	// for _, d := range l.devices {
	// 	if d != nil {
	// 		derr := d.Close()
	// 		if derr != nil {
	// 			err = derr
	// 		}
	// 	}
	// }

	// Return the first error encounted on all devices close, if any
	log.Printf("Finalizing adaptor\n")
	// if err != nil {
	// 	return err
	// }

	// Closing serial port
	l.port.Close()

	l.terminateDispatching <- true

	return nil
}

func (l *Adaptor) initialize(devicePath string, version string) (err error) {
	mode := &serial.Mode{
		BaudRate: 115200,
		Parity:   serial.NoParity,
		DataBits: 8,
		StopBits: serial.OneStopBit,
	}

	l.port, err = serial.Open(devicePath, mode)
	if err != nil {
		return err
	}
	l.port.SetReadTimeout(time.Second * 5)

	log.Printf("Checking lego hat version (expecting %s)...\n", version)
	_, err = l.port.Write([]byte("version\r"))
	if err != nil {
		return err
	}

	state := otherState
	detectedVersion := "not available"
	reader := bufio.NewReader(port)
	for retries := 0; retries < 5; retries++ {
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("error reading version: %s, retrying...\n", err.Error())
			continue
		}

		if len(line) == 0 {
			log.Printf("no data returned for version command\n")
		}

		if strings.HasPrefix(line, firmwareLine) {
			rawVersion := strings.TrimPrefix(line, firmwareLine)
			versionParts := strings.Fields(rawVersion)
			if versionParts[0] == version {
				state = firmwareState
				detectedVersion = versionParts[0]
				log.Printf("Correct version detected %s", detectedVersion)
				break
			}

			state = needNewFirmwareState
			break
		}

		if strings.HasPrefix(line, bootloaderLine) {
			state = bootloaderState
			break
		}

		log.Printf("Sending version command...")
		_, err = l.port.Write([]byte("version\r"))
		if err != nil {
			return err
		}
	}

	// TODO: remove once tested
	state = needNewFirmwareState

	if state == needNewFirmwareState {
		l.resetHat()
		err = l.loadFirmware()
		if err != nil {
			return err
		}

		err = l.reboot()
		if err != nil {
			return err
		}

		err = l.waitForText(doneLine)
		if err != nil {
			return err
		}
	} else if state == bootloaderState {
		err = l.loadFirmware()
		if err != nil {
			return err
		}

		err = l.reboot()
		if err != nil {
			return err
		}

		err = l.waitForText(doneLine)
		if err != nil {
			return err
		}
	}

	l.port.SetReadTimeout(time.Second * 1)
	return nil
}

func (l *Adaptor) reboot() (err error) {
	_, err = l.port.Write([]byte("reboot\r"))
	if err != nil {
		return err
	}

	return nil
}

func (l *Adaptor) resetHat() {
	log.Printf("Resetting hat...\n")
	l.turnPinOff(bootZeroPinNumber)
	l.turnPinOff(resetPinNumber)

	time.Sleep(10 * time.Millisecond)
	l.turnPinOn(resetPinNumber)
	time.Sleep(10 * time.Millisecond)

	time.Sleep(500 * time.Millisecond)
}

func (l *Adaptor) loadFirmware() (err error) {
	log.Printf("Loading firmware...\n")

	log.Printf("Clearing\n")
	_, err = l.port.Write([]byte("clear\r"))
	if err != nil {
		return err
	}

	log.Printf("Waiting for prompt...\n")
	err = l.waitForText(promptPrefix)
	if err != nil {
		return err
	}

	loadCmd := fmt.Sprintf("load %d %d\r", len(firmware), checksum())
	log.Printf("Sending load command %s...\n", loadCmd)
	_, err = l.port.Write([]byte(loadCmd))
	if err != nil {
		return err
	}

	time.Sleep(100 * time.Millisecond)

	log.Printf("Writing firmware...\n")
	_, err = l.port.Write([]byte{byte(0x02)})
	if err != nil {
		return err
	}

	_, err = l.port.Write(firmware)
	if err != nil {
		return err
	}

	_, err = l.port.Write([]byte{byte(0x03), '\r'})
	if err != nil {
		return err
	}

	log.Printf("Waiting for prompt...\n")
	err = l.waitForText(promptPrefix)
	if err != nil {
		return err
	}

	signatureCmd := fmt.Sprintf("signature %d\r", len(signature))
	log.Printf("Writing signature command [%s]...\n", signatureCmd)
	_, err = l.port.Write([]byte(signatureCmd))
	if err != nil {
		return err
	}

	time.Sleep(100 * time.Millisecond)

	log.Printf("Writing signature...\n")
	_, err = l.port.Write([]byte{byte(0x02)})
	if err != nil {
		return err
	}

	_, err = l.port.Write(signature)
	if err != nil {
		return err
	}

	_, err = l.port.Write([]byte{byte(0x03), '\r'})
	if err != nil {
		return err
	}

	log.Printf("Waiting for prompt...\n")
	err = l.waitForText(promptPrefix)
	if err != nil {
		return err
	}

	return nil
}

func checksum() int {
	val := 1

	for _, b := range firmware {
		if val&0x80000000 != 0 {
			val = (val << 1) ^ 0x1d872b41
		} else {
			val = val << 1
		}

		val = (val ^ int(b)) & 0xFFFFFFFF
	}

	return val
}

func (l *Adaptor) waitForText(text string) (err error) {
	promptReceived := make(chan struct{})
	go l.scanForText(text, promptReceived)

	select {
	case <-promptReceived:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timed out waiting for %s prompt from hat", promptPrefix)
	}
}

func (l *Adaptor) scanForText(text string, done chan struct{}) {
	scanner := bufio.NewScanner(l.port)
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), text) {
			done <- struct{}{}
			return
		}
	}
}

func (l *Adaptor) turnPinOff(pin string) {
	l.digitalWriter.DigitalWrite(pin, pinOff)
}

func (l *Adaptor) turnPinOn(pin string) {
	l.digitalWriter.DigitalWrite(pin, pinOn)
}

func (d *eventDispatcher) awaitMessage(portID LegoHatPortID, msgType DeviceMessageType) (registration eventRegistration) {
	receiver := make(chan DeviceEvent)
	registration = eventRegistration{
		conduit: receiver,
	}

	d.awaitedEvents[eventKey{msgType: msgType, portID: portID}] = registration

	return registration
}

func (d *eventDispatcher) awaitAllMessages(portID LegoHatPortID, msgType DeviceMessageType) (registration eventRegistration) {
	receiver := make(chan DeviceEvent)
	registration = eventRegistration{
		persistent: true,
		conduit:    receiver,
	}

	d.awaitedEvents[eventKey{msgType: msgType, portID: portID}] = registration

	return registration
}

func (d *eventDispatcher) dispatchEvents() {
	for e := range d.input {
		key := eventKey{
			msgType: e.msgType,
			portID:  e.portID,
		}

		if r, ok := d.awaitedEvents[key]; ok {
			r.conduit <- e

			// Drop the event listener unless it's persistent
			if !r.persistent {
				log.Printf("Dropping registration")
				delete(d.awaitedEvents, key)
			}
		}
	}
}
