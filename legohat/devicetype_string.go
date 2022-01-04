// Code generated by "stringer -type=DeviceType -linecomment"; DO NOT EDIT.

package legohat

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[lightDevice-8]
	_ = x[tiltSensorDevice-34]
	_ = x[motionSensorDevice-35]
	_ = x[colorDistanceSensorDevice-37]
	_ = x[colorSensorDevice-61]
	_ = x[distanceSensorDevice-62]
	_ = x[forceSensorDevice-63]
	_ = x[matrixDevice-64]
	_ = x[mediumLinearMotorDevice-38]
	_ = x[technicLargeMotorDevice-46]
	_ = x[technicXLargeMotorDevice-47]
	_ = x[spikePrimeMediumMotorDevice-48]
	_ = x[spikePrimeLargeMotorDevice-49]
	_ = x[spikeEssentialAngularMotorDevice-65]
	_ = x[mindstormMotor-75]
	_ = x[motor76Device-76]
}

const (
	_DeviceType_name_0 = "light"
	_DeviceType_name_1 = "tiltSensormotionSensor"
	_DeviceType_name_2 = "colorDistancemediumLinearMotor"
	_DeviceType_name_3 = "technicLargeMotortechnicXLargeMotorspikePrimeMediumMotorspikePrimeLargeMotor"
	_DeviceType_name_4 = "colorSensordistanceSensorforceSensormatrixspikeEssentialAngularMotor"
	_DeviceType_name_5 = "mindstormMotormotor"
)

var (
	_DeviceType_index_1 = [...]uint8{0, 10, 22}
	_DeviceType_index_2 = [...]uint8{0, 13, 30}
	_DeviceType_index_3 = [...]uint8{0, 17, 35, 56, 76}
	_DeviceType_index_4 = [...]uint8{0, 11, 25, 36, 42, 68}
	_DeviceType_index_5 = [...]uint8{0, 14, 19}
)

func (i DeviceType) String() string {
	switch {
	case i == 8:
		return _DeviceType_name_0
	case 34 <= i && i <= 35:
		i -= 34
		return _DeviceType_name_1[_DeviceType_index_1[i]:_DeviceType_index_1[i+1]]
	case 37 <= i && i <= 38:
		i -= 37
		return _DeviceType_name_2[_DeviceType_index_2[i]:_DeviceType_index_2[i+1]]
	case 46 <= i && i <= 49:
		i -= 46
		return _DeviceType_name_3[_DeviceType_index_3[i]:_DeviceType_index_3[i+1]]
	case 61 <= i && i <= 65:
		i -= 61
		return _DeviceType_name_4[_DeviceType_index_4[i]:_DeviceType_index_4[i+1]]
	case 75 <= i && i <= 76:
		i -= 75
		return _DeviceType_name_5[_DeviceType_index_5[i]:_DeviceType_index_5[i+1]]
	default:
		return "DeviceType(" + strconv.FormatInt(int64(i), 10) + ")"
	}
}
