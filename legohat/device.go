package legohat

import "fmt"

type Device struct {
	ID        int
	Name      string
	Type      DeviceType
	Listeners []chan []byte
}

type DeviceType int

const (
	lightDevice               DeviceType = 8  // light
	tiltSensorDevice          DeviceType = 34 // tiltSensor
	motionSensorDevice        DeviceType = 35 // motionSensor
	colorDistanceSensorDevice DeviceType = 37 // colorDistance
	colorSensorDevice         DeviceType = 61 // colorSensor
	distanceSensorDevice      DeviceType = 62 // distanceSensor
	forceSensorDevice         DeviceType = 63 // forceSensor
	matrixDevice              DeviceType = 64 // motor
	motor38Device             DeviceType = 38 // motor
	motor46Device             DeviceType = 46 // motor
	motor47Device             DeviceType = 47 // motor
	motor48Device             DeviceType = 48 // motor
	motor49Device             DeviceType = 49 // motor
	motor65Device             DeviceType = 65 // motor
	motor75Device             DeviceType = 75 // motor
	motor76Device             DeviceType = 76 // motor
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
	case motor38Device, motor46Device, motor47Device, motor48Device, motor49Device, motor65Device, motor75Device, motor76Device:
		return Motor, nil
	}

	return Unknown, fmt.Errorf("unknown device class for type [%d] (%s)", int(deviceType), deviceType)
}
