package cpu

import (
	"fmt"
)

type bus interface {
	Read(addr uint16) uint8
	Write(addr uint16, value uint8)
}

type CPU struct {
	Bus bus

	pc uint16
	sp uint16

	a uint8
	f uint8
	b uint8
	c uint8
	d uint8
	e uint8
	h uint8
	l uint8

	ime          bool
	imeScheduled bool
	iff          uint8 // 0xFF0F
	ie           uint8 // 0xFFFF

	halted bool

	gbDoctor bool
}

type Option func(*CPU)

const (
	INTERRUPTS_START_ADDR = 0x40
	IFF                   = 0xFF0F
	IE                    = 0xFFFF
)

func WithGBDoctor(gbDoctor bool) Option {
	return func(c *CPU) {
		c.gbDoctor = gbDoctor
	}
}

func (c *CPU) Init(options ...Option) error {
	for _, o := range options {
		o(c)
	}

	if err := ParseOpcodes(); err != nil {
		return fmt.Errorf("failed to parse CPU opcodes: %w", err)
	}

	c.bindOpcodeFuncs()

	// Registers after boot
	c.pc = 0x0100
	c.sp = 0xFFFE
	c.a = 0x01
	c.f = 0xB0
	c.b = 0x00
	c.c = 0x13
	c.d = 0x00
	c.e = 0xD8
	c.h = 0x01
	c.l = 0x4D

	return nil
}

func (c *CPU) Step() (int, error) {
	cycles := 4

	if c.halted {
		if c.ie&c.iff&0x1F != 0 { // Interrupt pending?
			c.halted = false
		}

		return cycles, nil // Return base CPU cycle
	}

	if c.gbDoctor {
		c.printGBDoctorLog()
	}

	if c.ime && (c.ie&c.iff) != 0 {
		cycles = c.handleInterrupts()
	}

	opcode := c.getOpcode(c.Bus.Read(c.pc))
	if opcode.Func == nil {
		return 0, fmt.Errorf("nil opcode func: %v", opcode)
	}

	cycles += opcode.Func(opcode)

	if c.imeScheduled {
		c.ime = true
		c.imeScheduled = false
	}

	return cycles, nil
}

func (c *CPU) Read(addr uint16) uint8 {
	if addr == IFF {
		return c.iff
	}

	if addr == IE {
		return c.ie
	}

	panic(fmt.Errorf("unsupported read on cpu: %x", addr))
}

func (c *CPU) Write(addr uint16, value uint8) {
	if addr == IFF {
		c.iff = value

		return
	}

	if addr == IE {
		c.ie = value

		return
	}

	panic(fmt.Errorf("unsupported write on cpu: %x", addr))
}

func (c *CPU) RequestInterrupt(code uint8) {
	c.iff |= code
}

func (c *CPU) printGBDoctorLog() {
	fmt.Printf("A:%02X F:%02X B:%02X C:%02X D:%02X E:%02X H:%02X L:%02X SP:%04X PC:%04X PCMEM:%02X,%02X,%02X,%02X\n", c.a, c.f, c.b, c.c, c.d, c.e, c.h, c.l, c.sp, c.pc, c.Bus.Read(c.pc), c.Bus.Read(c.pc+1), c.Bus.Read(c.pc+2), c.Bus.Read(c.pc+3))
}

func (c *CPU) getOpcode(opcodeHex uint8) *Opcode {
	opcode := &UnprefixedOpcodes[opcodeHex]

	if opcode.Mnemonic == "PREFIX" {
		opcodeHex = c.Bus.Read(c.pc + 1)
		opcode = &CBPrefixedOpcodes[opcodeHex]
	}

	return opcode
}

func (c *CPU) getOp(op string) uint8 {
	switch op {
	case "A":
		return c.a
	case "F":
		return c.f
	case "B":
		return c.b
	case "C":
		return c.c
	case "D":
		return c.d
	case "E":
		return c.e
	case "H":
		return c.h
	case "L":
		return c.l
	case "n8":
		return c.Bus.Read(c.pc + 1)
	case "HL":
		return c.Bus.Read(uint16(c.h)<<8 | uint16(c.l))
	default:
		panic("unsupported operand for getOp: " + op)
	}
}

func (c *CPU) setOp(op string, value uint8) {
	switch op {
	case "A":
		c.a = value
	case "F":
		c.f = value
	case "B":
		c.b = value
	case "C":
		c.c = value
	case "D":
		c.d = value
	case "E":
		c.e = value
	case "H":
		c.h = value
	case "L":
		c.l = value
	case "HL":
		c.Bus.Write(uint16(c.h)<<8|uint16(c.l), value)
	default:
		panic("unsupported operand for setOp: " + op)
	}
}

func (c *CPU) getDOp(op string) uint16 {
	switch op {
	case "AF":
		return uint16(c.a)<<8 | uint16(c.f)
	case "BC":
		return uint16(c.b)<<8 | uint16(c.c)
	case "DE":
		return uint16(c.d)<<8 | uint16(c.e)
	case "HL":
		return uint16(c.h)<<8 | uint16(c.l)
	case "SP":
		return c.sp
	default:
		panic("unsupported operand for getDOp: " + op)
	}
}

func (c *CPU) setDOp(op string, value uint16) {
	switch op {
	case "AF":
		c.a = uint8(value >> 8)
		c.f = uint8(value)
	case "BC":
		c.b = uint8(value >> 8)
		c.c = uint8(value)
	case "DE":
		c.d = uint8(value >> 8)
		c.e = uint8(value)
	case "HL":
		c.h = uint8(value >> 8)
		c.l = uint8(value)
	case "SP":
		c.sp = value
	default:
		panic("unsupported operand for getDOp: " + op)
	}
}

func (c *CPU) getZF() bool {
	return c.f>>7&0x1 == 1
}

func (c *CPU) getNF() bool {
	return c.f>>6&0x1 == 1
}

func (c *CPU) getHF() bool {
	return c.f>>5&0x1 == 1
}

func (c *CPU) getCF() bool {
	return c.f>>4&0x1 == 1
}

func (c *CPU) setZF(cond bool) {
	if cond {
		c.f |= 0x80
	} else {
		c.f &= 0x7F
	}
}

func (c *CPU) setNF(cond bool) {
	if cond {
		c.f |= 0x40
	} else {
		c.f &= 0xBF
	}
}

func (c *CPU) setHF(cond bool) {
	if cond {
		c.f |= 0x20
	} else {
		c.f &= 0xDF
	}
}

func (c *CPU) setCF(cond bool) {
	if cond {
		c.f |= 0x10
	} else {
		c.f &= 0xEF
	}
}

func (c *CPU) setFlags(zf, nf, hf, cf bool) {
	c.setZF(zf)
	c.setNF(nf)
	c.setHF(hf)
	c.setCF(cf)
}

func (c *CPU) handleInterrupts() int {
	c.ime = false
	c.imeScheduled = false

	for i := range 5 {
		mask := uint8(1 << i)
		if c.ie&c.iff&mask != 0 {
			c.iff &= ^mask
			c.pushValue(c.pc)
			c.pc = INTERRUPTS_START_ADDR + uint16(i*8)

			return 20
		}
	}

	return 0
}
