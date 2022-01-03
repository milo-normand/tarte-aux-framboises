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
	devices              map[LegoHatPortID]*deviceRegistration
	terminateReading     chan bool
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
		terminateReading:     make(chan bool),
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
	port, err := initialize(l.config.serialPath, strings.Replace(version, "\n", "", -1))
	if err != nil {
		return err
	}

	var builder strings.Builder
	for id := range l.devices {
		fmt.Fprintf(&builder, "port %d ; select ;", int(id))
	}
	builder.WriteString("echo 0\r")
	log.Printf("Sending command: %s\n", builder.String())

	_, err = port.Write([]byte(builder.String()))
	if err != nil {
		return err
	}

	_, err = port.Write([]byte("list\r"))
	if err != nil {
		return err
	}

	ready := make(chan error)
	go l.run(port, ready)

	err = <-ready

	if err != nil {
		return err
	}

	for _, d := range l.devices {
		log.Printf("Starting dispatching routine for device on port %d...\n", d.id)
		go l.dispatchInstructions(d.toDevice)
	}

	go l.writeInstructions(port)

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
}

func (l *Adaptor) writeInstructions(port serial.Port) {
	for {
		select {
		case in := <-l.toWrite:
			log.Printf("Writing %s to serial port %s", string(in), l.config.serialPath)
			port.Write(in)
		case <-l.terminateDispatching:
			log.Printf("Received termination signal to stop dispatching")
			break
		}
	}
}

func (l *Adaptor) run(port serial.Port, ready chan error) (err error) {
	defer port.Close()

	lines := make(chan string)
	go ReadPort(port, lines)

	for {
		select {
		case line := <-lines:
			if strings.HasPrefix(line, "P") {
				lineParts := strings.Split(line, ":")
				if len(lineParts) < 2 {
					return fmt.Errorf("unexpected line format with P prefix. should be P<id>: message but didn't have the ':' delimiter: %s", line)
				}
				rawPortID := strings.TrimPrefix(lineParts[0], "P")
				portID, err := strconv.Atoi(rawPortID)
				if err != nil {
					return err
				}

				message := strings.Trim(lineParts[1], " ")

				switch {
				case strings.HasPrefix(message, connectedMessage):
					rawDeviceType := strings.TrimPrefix(message, connectedMessage)
					deviceTypeVal, err := strconv.ParseInt(strings.Trim(rawDeviceType, " "), 16, 64)
					if err != nil {
						return err
					}

					deviceType := DeviceType(deviceTypeVal)
					log.Printf("Device of type %s connected on port %d", deviceType, portID)

					if d, ok := l.devices[LegoHatPortID(portID)]; ok {
						d.deviceType = deviceType
						d.fromDevice <- DeviceEvent{
							msgType: ConnectedMessage,
						}
					}
				case strings.HasPrefix(message, disconnectedMessage):
					log.Printf("Device disconnected on port %d", portID)

					if d, ok := l.devices[LegoHatPortID(portID)]; ok {
						d.fromDevice <- DeviceEvent{
							msgType: DisconnectedMessage,
						}
					}
				case strings.HasPrefix(message, timeoutMessage):
					log.Printf("Device timeout on port %d", portID)

					if d, ok := l.devices[LegoHatPortID(portID)]; ok {
						d.fromDevice <- DeviceEvent{
							msgType: TimeoutMessage,
						}
					}
				}
			}
		case <-l.terminateReading:
			log.Printf("Received termination signal...\n")
			return nil
		}
	}
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

	l.terminateReading <- true
	l.terminateDispatching <- true

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
