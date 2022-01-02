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
	"gobot.io/x/gobot/drivers/gpio"
)

const (
	firmwareLine   = "Firmware version: "
	bootloaderLine = "BuildHAT bootloader version"
)

type LegoHatPortID int

const (
	PortOne   = LegoHatPortID(0)
	PortTwo   = LegoHatPortID(1)
	PortThree = LegoHatPortID(2)
	PortFour  = LegoHatPortID(3)
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
	name        string
	config      Config
	devices     map[LegoHatPortID]*deviceRegistration
	reader      gpio.DigitalReader
	termination chan bool
}

type Option func(c *Config)

func WithSerialPath(serialPath string) Option {
	return func(c *Config) {
		c.serialPath = serialPath
	}
}

// NewAdaptor returns a new Lego Hat Adaptor.
func NewAdaptor(reader gpio.DigitalReader, opts ...Option) *Adaptor {
	config := Config{
		serialPath: "/dev/serial0",
	}

	for _, apply := range opts {
		apply(&config)
	}

	return &Adaptor{
		name:        gobot.DefaultName("LegoHat"),
		reader:      reader,
		config:      config,
		termination: make(chan bool),
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
	go l.Run(port, ready)

	err = <-ready

	if err != nil {
		return err
	}

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

func (l *Adaptor) Run(port serial.Port, ready chan error) (err error) {
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
					deviceTypeID, err := strconv.ParseInt(strings.Trim(rawDeviceType, " "), 16, 32)
					if err != nil {
						return err
					}

					log.Printf("Device of type %d connected on port %s", deviceTypeID, portID)

					if d, ok := l.devices[LegoHatPortID(portID)]; ok {
						d.deviceType = DeviceType(deviceTypeID)
						d.fromDevice <- DeviceEvent{
							msgType: ConnectedMessage,
						}
					}
				case strings.HasPrefix(message, disconnectedMessage):
					log.Printf("Device disconnected on port %s", portID)

					if d, ok := l.devices[LegoHatPortID(portID)]; ok {
						d.fromDevice <- DeviceEvent{
							msgType: DisconnectedMessage,
						}
					}
				case strings.HasPrefix(message, timeoutMessage):
					log.Printf("Device timeout on port %s", portID)

					if d, ok := l.devices[LegoHatPortID(portID)]; ok {
						d.fromDevice <- DeviceEvent{
							msgType: TimeoutMessage,
						}
					}
				}
			}
		case <-l.termination:
			break
		}
	}

	return nil
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
	if err != nil {
		return err
	}

	l.termination <- true

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
