package legohat

import "fmt"

type Device struct {
	ID   int
	Type DeviceType
}

type deviceRegistration struct {
	id         LegoHatPortID
	class      DeviceClass
	deviceType DeviceType
	name       string
	fromDevice chan DeviceEvent
	toDevice   chan []byte
}

type DeviceMessageType string

const (
	ConnectedMessage    DeviceMessageType = "connected"
	DisconnectedMessage DeviceMessageType = "disconnected"
	TimeoutMessage      DeviceMessageType = "timeout"
	PulseDoneMessage    DeviceMessageType = "pulseDone"
	RampDoneMessage     DeviceMessageType = "rampDone"
	DataMessage         DeviceMessageType = "data"
)

type DeviceEvent struct {
	msgType DeviceMessageType
	data    []byte
}

type DeviceType int

const (
	lightDevice                      DeviceType = 0x08 // light
	tiltSensorDevice                 DeviceType = 0x22 // tiltSensor
	motionSensorDevice               DeviceType = 0x23 // motionSensor
	colorDistanceSensorDevice        DeviceType = 0x25 // colorDistance
	colorSensorDevice                DeviceType = 0x3d // colorSensor
	distanceSensorDevice             DeviceType = 0x3e // distanceSensor
	forceSensorDevice                DeviceType = 0x3f // forceSensor
	matrixDevice                     DeviceType = 0x40 // matrix
	mediumLinearMotorDevice          DeviceType = 0x26 // mediumLinearMotor
	technicLargeMotorDevice          DeviceType = 0x2e // technicLargeMotor
	technicXLargeMotorDevice         DeviceType = 0x2f // technicXLargeMotor
	spikePrimeMediumMotorDevice      DeviceType = 0x30 // spikePrimeMediumMotor
	spikePrimeLargeMotorDevice       DeviceType = 0x31 // spikePrimeLargeMotor
	spikeEssentialAngularMotorDevice DeviceType = 0x41 // spikeEssentialAngularMotor
	mindstormMotor                   DeviceType = 0x4B // mindstormMotor
	motor76Device                    DeviceType = 0x4C // motor
)

type DeviceClass int

const (
	Unknown DeviceClass = iota
	Light
	TiltSensor
	MotionSensor
	ColorDistanceSensor
	ColorSensor
	DistanceSensor
	ForceSensor
	Matrix
	Motor
)

func getDeviceClassForType(deviceType DeviceType) (class DeviceClass, err error) {
	switch deviceType {
	case lightDevice:
		return Light, nil
	case tiltSensorDevice:
		return TiltSensor, nil
	case motionSensorDevice:
		return MotionSensor, nil
	case colorDistanceSensorDevice:
		return ColorDistanceSensor, nil
	case colorSensorDevice:
		return ColorSensor, nil
	case distanceSensorDevice:
		return DistanceSensor, nil
	case forceSensorDevice:
		return ForceSensor, nil
	case matrixDevice:
		return Matrix, nil
	case mediumLinearMotorDevice, technicLargeMotorDevice, technicXLargeMotorDevice, spikePrimeMediumMotorDevice, spikePrimeLargeMotorDevice, spikeEssentialAngularMotorDevice, mindstormMotor, motor76Device:
		return Motor, nil
	}

	return Unknown, fmt.Errorf("unknown device class for type [%d] (%s)", int(deviceType), deviceType)
}
