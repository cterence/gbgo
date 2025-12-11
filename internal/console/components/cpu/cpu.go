package cpu

import (
	"fmt"
	"strings"
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

	nextOpcodePrefixed bool
}

func (c *CPU) Init() error {
	if err := c.parseOpcodes(); err != nil {
		return fmt.Errorf("failed to parse CPU opcodes: %w", err)
	}

	c.pc = 0

	return nil
}

func (c *CPU) Step() error {
	opcodeHex := c.Bus.Read(c.pc)

	var opcode Opcode

	if c.nextOpcodePrefixed {
		opcode = cbprefixedOpcodes[opcodeHex]
		c.nextOpcodePrefixed = false
	} else {
		opcode = unprefixedOpcodes[opcodeHex]
	}

	var opStrSb strings.Builder
	for _, op := range opcode.Operands {
		opStrSb.WriteString(" " + op.Name)
	}

	opStr := opStrSb.String()

	cb := ""
	if c.nextOpcodePrefixed {
		cb = "cb"
	}

	if opcode.Func == nil {
		return fmt.Errorf("unimplemented opcode: 0x%s%02X %s%s (pc:%x)", cb, opcodeHex, opcode.Mnemonic, opStr, c.pc)
	}

	fmt.Printf("executing %s%02X %s%s (pc:%x)\n", cb, opcodeHex, opcode.Mnemonic, opStr, c.pc)

	prevPC := c.pc

	opcode.Func(&opcode)

	if c.pc == prevPC {
		c.pc++
	}

	return nil
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
	case "[HL]":
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
	case "[HL]":
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

// func (c *CPU) getZF() bool {
// 	return c.f>>7&0x1 == 1
// }

// func (c *CPU) getNF() bool {
// 	return c.f>>6&0x1 == 1
// }

// func (c *CPU) getHF() bool {
// 	return c.f>>5&0x1 == 1
// }

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
