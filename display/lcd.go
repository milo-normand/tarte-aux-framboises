package display

import (
	"fmt"
	"strconv"
	"time"

	"gobot.io/x/gobot"
	"gobot.io/x/gobot/drivers/i2c"
)

const (
	// Commands
	CMD_Clear_Display        = 0x01
	CMD_Return_Home          = 0x02
	CMD_Entry_Mode           = 0x04
	CMD_Display_Control      = 0x08
	CMD_Cursor_Display_Shift = 0x10
	CMD_Function_Set         = 0x20
	CMD_CGRAM_Set            = 0x40
	CMD_DDRAM_Set            = 0x80

	// Options
	OPT_Increment = 0x02 // CMD_Entry_Mode
	OPT_Decrement = 0x00
	// OPT_Display_Shift  = 0x01 // CMD_Entry_Mode
	OPT_Enable_Display = 0x04 // CMD_Display_Control
	OPT_Enable_Cursor  = 0x02 // CMD_Display_Control
	OPT_Enable_Blink   = 0x01 // CMD_Display_Control
	OPT_Display_Shift  = 0x08 // CMD_Cursor_Display_Shift
	OPT_Shift_Right    = 0x04 // CMD_Cursor_Display_Shift 0 = Left
	OPT_8Bit_Mode      = 0x10
	OPT_4Bit_Mode      = 0x00
	OPT_2_Lines        = 0x08 // CMD_Function_Set 0 = 1 line
	OPT_1_Lines        = 0x00
	OPT_5x10_Dots      = 0x04 // CMD_Function_Set 0 = 5x7 dots
	OPT_5x8_Dots       = 0x00
)

const (
	PIN_BACKLIGHT byte = 0x08
	PIN_EN        byte = 0x04 // Enable bit
	PIN_RW        byte = 0x02 // Read/Write bit
	PIN_RS        byte = 0x01 // Register select bit
)

type LcdType int

const (
	LCD_UNKNOWN LcdType = iota
	LCD_16x2
	LCD_20x4
)

type LCDDriver struct {
	name          string
	backlight     bool
	lcdType       LcdType
	connector     i2c.Connector
	lcdAddress    int
	lcdConnection i2c.Connection
	i2c.Config
	gobot.Commander
}

func NewLCDDriver(a i2c.Connector, lcdType LcdType, options ...func(i2c.Config)) *LCDDriver {
	d := &LCDDriver{
		name:       gobot.DefaultName("LCDDisplay"),
		connector:  a,
		Config:     i2c.NewConfig(),
		Commander:  gobot.NewCommander(),
		lcdAddress: 0x27,
		backlight:  true,
	}

	for _, option := range options {
		option(d)
	}

	d.AddCommand("Clear", func(params map[string]interface{}) interface{} {
		return d.Clear()
	})
	d.AddCommand("Home", func(params map[string]interface{}) interface{} {
		return d.Home()
	})
	d.AddCommand("Write", func(params map[string]interface{}) interface{} {
		msg := params["msg"].(string)
		return d.Write(msg)
	})
	d.AddCommand("SetPosition", func(params map[string]interface{}) interface{} {
		line, _ := strconv.Atoi(params["line"].(string))
		col, _ := strconv.Atoi(params["col"].(string))
		return d.SetPosition(line, col)
	})

	return d
}

// Name returns the name the LCDDisplay Driver was given when created.
func (d *LCDDriver) Name() string { return d.name }

// SetName sets the name for the LCDDisplay Driver.
func (d *LCDDriver) SetName(n string) { d.name = n }

// Connection returns the driver connection to the device.
func (d *LCDDriver) Connection() gobot.Connection {
	return d.connector.(gobot.Connection)
}

// Start starts the backlit and the screen and initializes the states.
func (d *LCDDriver) Start() (err error) {
	bus := d.GetBusOrDefault(d.connector.GetDefaultBus())

	if d.lcdConnection, err = d.connector.GetConnection(d.lcdAddress, bus); err != nil {
		return err
	}

	initByteSeq := []byte{
		0x03, 0x03, 0x03, // base initialization
		0x02, // setting up 4-bit transfer mode
		CMD_Function_Set | OPT_2_Lines | OPT_5x8_Dots | OPT_4Bit_Mode,
		CMD_Display_Control | OPT_Enable_Display,
		CMD_Entry_Mode | OPT_Increment,
	}
	for _, b := range initByteSeq {
		err := d.writeByte(b, 0)
		if err != nil {
			return err
		}
	}
	err = d.Clear()
	if err != nil {
		return err
	}
	err = d.Home()
	if err != nil {
		return err
	}

	return nil
}

type rawData struct {
	Data  byte
	Delay time.Duration
}

func (d *LCDDriver) writeByte(data byte, controlPins byte) error {
	err := d.writeDataWithStrobe(data&0xF0 | controlPins)
	if err != nil {
		return err
	}
	err = d.writeDataWithStrobe((data<<4)&0xF0 | controlPins)
	if err != nil {
		return err
	}
	return nil
}

func (d *LCDDriver) writeDataWithStrobe(data byte) error {
	if d.backlight {
		data |= PIN_BACKLIGHT
	}
	seq := []rawData{
		{data, 0},                               // send data
		{data | PIN_EN, 200 * time.Microsecond}, // set strobe
		{data, 30 * time.Microsecond},           // reset strobe
	}
	return d.writeRawDataSeq(seq)
}

func (d *LCDDriver) writeRawDataSeq(seq []rawData) error {
	for _, item := range seq {
		_, err := d.lcdConnection.Write([]byte{item.Data})
		if err != nil {
			return err
		}
		time.Sleep(item.Delay)
	}
	return nil
}

// Home sets the cursor to the origin position on the display.
func (d *LCDDriver) Home() error {
	err := d.writeByte(CMD_Return_Home, 0)
	time.Sleep(3 * time.Millisecond)
	return err
}

// Write displays the passed message on the screen.
func (d *LCDDriver) Write(message string) error {
	buf := []byte(message)
	for _, c := range buf {
		err := d.writeByte(c, PIN_RS)
		if err != nil {
			return err
		}
	}
	return nil
}

// SetPosition sets the line and column position
func (d *LCDDriver) SetPosition(line, col int) error {
	w, h := d.getSize()
	if w != -1 && (col < 0 || col > w-1) {
		return fmt.Errorf("Cursor position %d "+
			"must be within the range [0..%d]", col, w-1)
	}
	if h != -1 && (line < 0 || line > h-1) {
		return fmt.Errorf("Cursor line %d "+
			"must be within the range [0..%d]", line, h-1)
	}
	lineOffset := []byte{0x00, 0x40, 0x14, 0x54}
	var b byte = CMD_DDRAM_Set + lineOffset[line] + byte(col)
	err := d.writeByte(b, 0)
	return err
}

func (d *LCDDriver) getSize() (width, height int) {
	switch d.lcdType {
	case LCD_16x2:
		return 16, 2
	case LCD_20x4:
		return 20, 4
	default:
		return -1, -1
	}
}

func (d *LCDDriver) BacklightOn() error {
	d.backlight = true
	err := d.writeByte(0x00, 0)
	if err != nil {
		return err
	}
	return nil
}

func (d *LCDDriver) BacklightOff() error {
	d.backlight = false
	err := d.writeByte(0x00, 0)
	if err != nil {
		return err
	}
	return nil
}

// Clear clears the text on the lCD display.
func (d *LCDDriver) Clear() error {
	err := d.writeByte(CMD_Clear_Display, 0)
	return err
}

// Halt is a noop function.
func (d *LCDDriver) Halt() error { return nil }

func (d *LCDDriver) Command(cmd byte) error {
	err := d.writeByte(cmd, 0)
	return err
}
