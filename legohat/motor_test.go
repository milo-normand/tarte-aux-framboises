package legohat

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAngleCommand(t *testing.T) {
	testCases := []struct {
		name        string
		state       MotorState
		angle       int
		expectedCmd string
	}{
		{
			name: "0 to -60",
			state: MotorState{
				speed:            0,
				absolutePosition: -31,
				position:         0,
			},
			angle:       -60,
			expectedCmd: "port 1 ; combi 0 1 0 2 0 3 0 ; pid 1 0 1 s4 0.0027777778 0 5 0 .1 3 ; set ramp 0.000000 -0.080556 0.016111 0\r",
		},
		{
			name: "-60 to 60",
			state: MotorState{
				speed:            0,
				absolutePosition: -55,
				position:         -24,
			},
			angle:       60,
			expectedCmd: "port 1 ; combi 0 1 0 2 0 3 0 ; pid 1 0 1 s4 0.0027777778 0 5 0 .1 3 ; set ramp 0.9583333333333334 1.2277777777777779 0.053888888888888896 0\r",
		},
		{
			name: "60 to 0",
			state: MotorState{
				speed:            0,
				absolutePosition: 47,
				position:         77,
			},
			angle:       0,
			expectedCmd: "port 1 ; combi 0 1 0 2 0 3 0 ; pid 1 0 1 s4 0.0027777778 0 5 0 .1 3 ; set ramp -0.066667 0.252778 0.063889 0\r",
		},
		{
			name: "reset to 0",
			state: MotorState{
				speed:            0,
				absolutePosition: -27,
				position:         3,
			},
			angle:       0,
			expectedCmd: "port 1 ; combi 0 1 0 2 0 3 0 ; pid 1 0 1 s4 0.0027777778 0 5 0 .1 3 ; set ramp 0.008333 0.083333 0.015000 0\r",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := toAngleCommand(1, tc.state, tc.angle, 100, shortest)
			assert.Equal(t, tc.expectedCmd, cmd)
		})
	}
}
