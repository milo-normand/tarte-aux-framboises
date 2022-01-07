package legohat

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"go.bug.st/serial"
	"gobot.io/x/gobot"
)

const (
	firmwareLine   = "Firmware version: "
	bootloaderLine = "BuildHAT bootloader version"
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
}

type Option func(c *Config)

func WithSerialPath(serialPath string) Option {
	return func(c *Config) {
		c.serialPath = serialPath
	}
}

// NewAdaptor returns a new Lego Hat Adaptor.
func NewAdaptor(opts ...Option) *Adaptor {
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
	}
}

// Name returns the Adaptors name
func (l *Adaptor) Name() string { return l.name }

// SetName sets the Adaptors name
func (l *Adaptor) SetName(n string) { l.name = n }

// Connect connects to the joystick
func (l *Adaptor) Connect() (err error) {
	l.port, err = initialize(l.config.serialPath, strings.Replace(version, "\n", "", -1))
	if err != nil {
		return err
	}

	go l.run()

	for _, d := range l.devices {
		log.Printf("Starting dispatching routine for device on port %d...\n", d.id)
		go l.dispatchInstructions(d.toDevice)
	}

	go l.writeInstructions()

	return nil
}

func (l *Adaptor) registerDevice(portID LegoHatPortID, deviceClass DeviceClass) (registration *deviceRegistration) {
	r := deviceRegistration{
		id:         portID,
		class:      deviceClass,
		name:       deviceClass.String(),
		toDevice:   make(chan []byte),
		fromDevice: make(chan DeviceEvent),
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
// TODO: Handle data messages
// TODO: implement consumption of all messages and dispatching of only expected/awaited messages
func (l *Adaptor) run() {
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
			rawPortID := strings.TrimPrefix(lineParts[0], "P")
			portID := rawPortID[0] - '0'

			message := strings.Trim(lineParts[1], " ")

			switch {
			case strings.HasPrefix(message, connectedMessage):
				rawDeviceType := strings.TrimPrefix(message, connectedMessage)
				deviceTypeVal, err := strconv.ParseInt(strings.Trim(rawDeviceType, " "), 16, 64)
				if err != nil {
					log.Printf("unexpected device type format on line %s: %s\n", line, err.Error())
				}

				deviceType := DeviceType(deviceTypeVal)
				log.Printf("Device of type %s connected on port %d", deviceType, portID)

				if d, ok := l.devices[LegoHatPortID(portID)]; ok {
					log.Printf("Sending message [%s] to listener %v...\n", ConnectedMessage, d)
					d.deviceType = deviceType
					d.fromDevice <- DeviceEvent{
						msgType: ConnectedMessage,
					}
				}
			case strings.HasPrefix(message, disconnectedMessage):
				log.Printf("Device disconnected on port %d", portID)

				// TODO: figure out what to do with this case. Do we need somethint that receives all
				// messages all the time and only dispatches responses that have been registered and
				// awaited for

				// if d, ok := l.devices[LegoHatPortID(portID)]; ok {
				// 	d.fromDevice <- DeviceEvent{
				// 		msgType: DisconnectedMessage,
				// 	}
				// }
			case strings.HasPrefix(message, timeoutMessage):
				log.Printf("Device timeout on port %d", portID)

				if d, ok := l.devices[LegoHatPortID(portID)]; ok {
					d.fromDevice <- DeviceEvent{
						msgType: TimeoutMessage,
					}
				}
			case strings.HasPrefix(message, pulseDoneMessage):
				log.Printf("Pulse done message on port %d", portID)

				if d, ok := l.devices[LegoHatPortID(portID)]; ok {
					d.fromDevice <- DeviceEvent{
						msgType: PulseDoneMessage,
					}
				}
			case strings.HasPrefix(message, rampDoneMessage):
				log.Printf("Ramp done message on port %d", portID)

				if d, ok := l.devices[LegoHatPortID(portID)]; ok {
					d.fromDevice <- DeviceEvent{
						msgType: RampDoneMessage,
					}
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
		log.Printf("raw line: %s\n", line)

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
	log.Printf("Sending signal to stop reading...\n")

	for _, d := range l.devices {
		close(d.fromDevice)
	}

	return nil
}

func initialize(devicePath string, version string) (port serial.Port, err error) {
	mode := &serial.Mode{
		BaudRate: 115200,
		Parity:   serial.NoParity,
		DataBits: 8,
		StopBits: serial.OneStopBit,
	}

	port, err = serial.Open(devicePath, mode)
	if err != nil {
		return nil, err
	}
	port.SetReadTimeout(time.Second * 5)

	log.Printf("Checking lego hat version (expecting %s)...\n", version)
	_, err = port.Write([]byte("version\r"))
	if err != nil {
		return nil, err
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
			versionParts := strings.Split(rawVersion, " ")
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
		_, err = port.Write([]byte("version\r"))
		if err != nil {
			return nil, err
		}
	}

	if state != firmwareState {
		return nil, fmt.Errorf("expected state [%s] with version [%s] but got state [%s] and version [%s]", firmwareState, version, state, detectedVersion)
	}

	port.SetReadTimeout(time.Second * 1)
	return port, nil
}
