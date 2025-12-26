package cpu

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"github.com/cterence/gbgo/internal/lib"
	"github.com/cterence/gbgo/internal/log"
)

type bus interface {
	Read(addr uint16) uint8
	Write(addr uint16, value uint8)
}

type console interface {
	Stop()
}

type CPU struct {
	Bus     bus
	Console console
	state
}

type state struct {
	CurrentOpcode *Opcode

	PC uint16
	SP uint16

	A uint8
	F uint8
	B uint8
	C uint8
	D uint8
	E uint8
	H uint8
	L uint8

	IME          bool
	IMEScheduled bool
	IFF          uint8 // 0xFF0F
	IE           uint8 // 0xFFFF
	Halted       bool
	HaltBug      bool

	// Emulator
	Debug      bool
	UseBootROM bool
}

type Option func(*CPU)

const (
	INTERRUPTS_START_ADDR = 0x40
	IFF                   = 0xFF0F
	IE                    = 0xFFFF
)

func WithDebug(debug bool) Option {
	return func(c *CPU) {
		c.Debug = debug
	}
}

func WithBootROM() Option {
	return func(c *CPU) {
		c.UseBootROM = true
	}
}

func (c *CPU) String() string {
	// return fmt.Sprintf("%-12s - A:%02X F:%02X B:%02X C:%02X D:%02X E:%02X H:%02X L:%02X SP:%04X PC:%04X PCMEM:%02X,%02X,%02X,%02X", c.CurrentOpcode, c.A, c.F, c.B, c.C, c.D, c.E, c.H, c.L, c.SP, c.PC, c.Bus.Read(c.PC), c.Bus.Read(c.PC+1), c.Bus.Read(c.PC+2), c.Bus.Read(c.PC+3))
	return fmt.Sprintf(" %04x: %02x %02x %02x  A:%02x F:%02x B:%02x C:%02x D:%02x E:%02x H:%02x L:%02x SP:%04x",
		c.PC, c.Bus.Read(c.PC), c.Bus.Read(c.PC+1), c.Bus.Read(c.PC+2), c.A, c.F, c.B, c.C, c.D, c.E, c.H, c.L, c.SP)
}

func (c *CPU) Init(options ...Option) {
	for _, o := range options {
		o(c)
	}

	if err := ParseOpcodes(); err != nil {
		panic(fmt.Errorf("failed to parse CPU opcodes: %w", err))
	}

	c.bindOpcodeFuncs()

	c.PC = 0x0100
	c.SP = 0xFFFE
	c.A = 0x01
	c.F = 0xB0
	c.B = 0x00
	c.C = 0x13
	c.D = 0x00
	c.E = 0xD8
	c.H = 0x01
	c.L = 0x4D

	if c.UseBootROM {
		c.PC = 0
		c.SP = 0
		c.A = 0
		c.F = 0
		c.B = 0
		c.C = 0
		c.D = 0
		c.E = 0
		c.H = 0
		c.L = 0
	}

	c.IME = false
	c.IMEScheduled = false
	c.IFF = 0
	c.IE = 0
	c.HaltBug = false
	c.Halted = false
}

func (c *CPU) Step() int {
	cycles := 0

	if c.Halted {
		if c.IE&c.IFF&0x1F != 0 { // Interrupt pending?
			c.Halted = false
		} else {
			return 4 // Return base CPU cycle
		}
	}

	if c.IME && (c.IE&c.IFF) != 0 {
		cycles = c.handleInterrupts()
	}

	if log.DebugEnabled {
		fmt.Println(c)
	}

	opcode := c.getOpcode()

	cycles += opcode.Func(opcode)

	if c.IMEScheduled {
		c.IME = true
		c.IMEScheduled = false
	}

	return cycles
}

func (c *CPU) Read(addr uint16) uint8 {
	switch addr {
	case IFF:
		return c.IFF | 0xE0
	case IE:
		return c.IE
	default:
		panic(fmt.Errorf("unsupported read on cpu: %x", addr))
	}
}

func (c *CPU) Write(addr uint16, value uint8) {
	switch addr {
	case IFF:
		c.IFF = value
	case IE:
		c.IE = value
	default:
		panic(fmt.Errorf("unsupported write on cpu: %x", addr))
	}
}

func (c *CPU) RequestInterrupt(code uint8) {
	c.IFF |= code
}

func (c *CPU) Load(buf *bytes.Reader) {
	enc := gob.NewDecoder(buf)
	err := enc.Decode(&c.state)

	lib.Assert(err == nil, "failed to decode CPU state: %v", err)
}

func (c *CPU) Save(buf *bytes.Buffer) {
	enc := gob.NewEncoder(buf)
	err := enc.Encode(c.state)

	lib.Assert(err == nil, "failed to encode CPU state: %v", err)
}

func (c *CPU) fetchByte() uint8 {
	val := c.Bus.Read(c.PC)

	if c.HaltBug {
		c.HaltBug = false
	} else {
		c.PC++
	}

	return val
}

func (c *CPU) fetchWord() uint16 {
	lo := c.fetchByte()
	hi := c.fetchByte()

	return uint16(hi)<<8 | uint16(lo)
}

func (c *CPU) getOpcode() *Opcode {
	opcode := &UnprefixedOpcodes[c.fetchByte()]

	if opcode.Mnemonic == "PREFIX" {
		opcode = &CBPrefixedOpcodes[c.fetchByte()]
	}

	c.CurrentOpcode = opcode

	return opcode
}

func (c *CPU) getOp(op string) uint8 {
	switch op {
	case "A":
		return c.A
	case "F":
		return c.F
	case "B":
		return c.B
	case "C":
		return c.C
	case "D":
		return c.D
	case "E":
		return c.E
	case "H":
		return c.H
	case "L":
		return c.L
	case "n8":
		return c.fetchByte()
	case "HL":
		return c.Bus.Read(uint16(c.H)<<8 | uint16(c.L))
	default:
		panic("unsupported operand for getOp: " + op)
	}
}

func (c *CPU) setOp(op string, value uint8) {
	switch op {
	case "A":
		c.A = value
	case "F":
		c.F = value
	case "B":
		c.B = value
	case "C":
		c.C = value
	case "D":
		c.D = value
	case "E":
		c.E = value
	case "H":
		c.H = value
	case "L":
		c.L = value
	case "HL":
		c.Bus.Write(uint16(c.H)<<8|uint16(c.L), value)
	default:
		panic("unsupported operand for setOp: " + op)
	}
}

func (c *CPU) getDOp(op string) uint16 {
	switch op {
	case "AF":
		return uint16(c.A)<<8 | uint16(c.F)
	case "BC":
		return uint16(c.B)<<8 | uint16(c.C)
	case "DE":
		return uint16(c.D)<<8 | uint16(c.E)
	case "HL":
		return uint16(c.H)<<8 | uint16(c.L)
	case "SP":
		return c.SP
	default:
		panic("unsupported operand for getDOp: " + op)
	}
}

func (c *CPU) setDOp(op string, value uint16) {
	switch op {
	case "AF":
		c.A = uint8(value >> 8)
		c.F = uint8(value)
	case "BC":
		c.B = uint8(value >> 8)
		c.C = uint8(value)
	case "DE":
		c.D = uint8(value >> 8)
		c.E = uint8(value)
	case "HL":
		c.H = uint8(value >> 8)
		c.L = uint8(value)
	case "SP":
		c.SP = value
	default:
		panic("unsupported operand for getDOp: " + op)
	}
}

func (c *CPU) getZF() bool {
	return c.F>>7&0x1 == 1
}

func (c *CPU) getNF() bool {
	return c.F>>6&0x1 == 1
}

func (c *CPU) getHF() bool {
	return c.F>>5&0x1 == 1
}

func (c *CPU) getCF() bool {
	return c.F>>4&0x1 == 1
}

func (c *CPU) setZF(cond bool) {
	if cond {
		c.F |= 0x80
	} else {
		c.F &= 0x7F
	}
}

func (c *CPU) setNF(cond bool) {
	if cond {
		c.F |= 0x40
	} else {
		c.F &= 0xBF
	}
}

func (c *CPU) setHF(cond bool) {
	if cond {
		c.F |= 0x20
	} else {
		c.F &= 0xDF
	}
}

func (c *CPU) setCF(cond bool) {
	if cond {
		c.F |= 0x10
	} else {
		c.F &= 0xEF
	}
}

func (c *CPU) setFlags(zf, nf, hf, cf bool) {
	c.setZF(zf)
	c.setNF(nf)
	c.setHF(hf)
	c.setCF(cf)
}

func (c *CPU) handleInterrupts() int {
	c.IME = false
	c.IMEScheduled = false

	for i := range 5 {
		mask := uint8(1 << i)
		if c.IE&c.IFF&mask != 0 {
			c.IFF &= ^mask
			c.pushValue(c.PC)
			c.PC = INTERRUPTS_START_ADDR + uint16(i*8)

			return 20
		}
	}

	return 0
}
