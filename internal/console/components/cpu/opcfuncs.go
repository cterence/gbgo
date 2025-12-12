package cpu

import (
	"strconv"

	"github.com/cterence/gbgo/internal/console/lib"
)

// Unprefixed

func (c *CPU) nop(opc *Opcode) int {
	c.pc += opc.Bytes

	return opc.Cycles[0]
}

func (c *CPU) prefix(opc *Opcode) int {
	c.nextOpcodePrefixed = true
	c.pc += opc.Bytes

	return opc.Cycles[0]
}

func (c *CPU) call(opc *Opcode) int {
	cycles := opc.Cycles[0]
	op0 := opc.Operands[0]

	doCall := func(c *CPU) {
		imm16 := uint16(c.Bus.Read(c.pc+2))<<8 | uint16(c.Bus.Read(c.pc+1))
		c.pushValue(c.pc + opc.Bytes)
		c.pc = imm16
	}

	switch op0.Name {
	case "a16":
		doCall(c)
	case "NZ":
		if !c.getZF() {
			doCall(c)
		} else {
			c.pc += opc.Bytes
		}
	case "Z":
		if c.getZF() {
			doCall(c)
		} else {
			c.pc += opc.Bytes
		}
	case "NC":
		if !c.getCF() {
			doCall(c)
		} else {
			c.pc += opc.Bytes
		}
	case "C":
		if c.getCF() {
			doCall(c)
		} else {
			c.pc += opc.Bytes
		}
	default:
		panic("unimplemented call for " + op0.Name)
	}

	return cycles
}

func (c *CPU) ret(opc *Opcode) int {
	c.pc = c.popValue()

	return opc.Cycles[0]
}

func (c *CPU) load(opc *Opcode) int {
	op0 := opc.Operands[0]
	op1 := opc.Operands[1]

	switch op0.Name {
	case "A", "F", "B", "C", "D", "E", "H", "L":
		switch op1.Name {
		case "A", "F", "B", "C", "D", "E", "H", "L":
			c.setOp(op0.Name, c.getOp(op1.Name))
		case "n8":
			c.setOp(op0.Name, c.Bus.Read(c.pc+1))
		case "BC", "DE", "HL":
			c.setOp(op0.Name, c.Bus.Read(c.getDOp(op1.Name)))

			if op1.Increment {
				c.setDOp(op1.Name, c.getDOp(op1.Name)+1)
			}

			if op1.Decrement {
				c.setDOp(op1.Name, c.getDOp(op1.Name)-1)
			}
		case "a16":
			c.a = c.Bus.Read(uint16(c.Bus.Read(c.pc+2))<<8 | uint16(c.Bus.Read(c.pc+1)))
		default:
			panic("unimplemented load for " + op1.Name)
		}
	case "BC", "DE", "HL", "SP":
		if op0.Immediate {
			imm16 := uint16(c.Bus.Read(c.pc+2))<<8 | uint16(c.Bus.Read(c.pc+1))
			c.setDOp(op0.Name, imm16)
		} else {
			c.Bus.Write(c.getDOp(op0.Name), c.getOp(op1.Name))

			if op0.Increment {
				c.setDOp(op0.Name, c.getDOp(op0.Name)+1)
			}

			if op0.Decrement {
				c.setDOp(op0.Name, c.getDOp(op0.Name)-1)
			}
		}
	case "a16":
		hi := uint16(c.Bus.Read(c.pc + 2))
		lo := uint16(c.Bus.Read(c.pc + 1))

		if op1.Name == "SP" {
			c.Bus.Write(hi, uint8(c.sp>>8))
			c.Bus.Write(lo, uint8(c.sp))
		}

		if op1.Name == "A" {
			c.Bus.Write(hi<<8|lo, c.a)
		}
	default:
		panic("unimplemented load for " + op0.Name)
	}

	c.pc += opc.Bytes

	return opc.Cycles[0]
}

func (c *CPU) loadH(opc *Opcode) int {
	op0 := opc.Operands[0]
	op1 := opc.Operands[1]

	switch op0.Name {
	case "a8":
		c.Bus.Write(0xFF00|uint16(c.Bus.Read(c.pc+1)), c.a)
	case "C":
		c.Bus.Write(0xFF00|uint16(c.Bus.Read(uint16(c.c))), c.a)
	case "A":
		if op1.Name == "a8" {
			c.a = c.Bus.Read(0xFF00 | uint16(c.Bus.Read(c.pc+1)))
		}

		if op1.Name == "C" {
			c.a = c.Bus.Read(0xFF00 | uint16(c.c))
		}
	default:
		panic("unimplemented loadH for " + op0.Name)
	}

	c.pc += opc.Bytes

	return opc.Cycles[0]
}

func (c *CPU) inc(opc *Opcode) int {
	op0 := opc.Operands[0]

	if len(op0.Name) == 1 {
		v := c.getOp(op0.Name)
		res := v + 1
		c.setOp(op0.Name, res)
		c.setFlags(res == 0, false, v&0xF+1 > 0xF, c.getCF())
	}

	if len(op0.Name) == 2 {
		c.setDOp(op0.Name, c.getDOp(op0.Name)+1)
	}

	c.pc += opc.Bytes

	return opc.Cycles[0]
}

func (c *CPU) dec(opc *Opcode) int {
	op0 := opc.Operands[0]

	if len(op0.Name) == 1 {
		v := c.getOp(op0.Name)
		res := v - 1
		c.setOp(op0.Name, res)
		c.setFlags(res == 0, true, v&0xF < v&0xF-1, c.getCF())
	}

	if len(op0.Name) == 2 {
		c.setDOp(op0.Name, c.getDOp(op0.Name)-1)
	}

	c.pc += opc.Bytes

	return opc.Cycles[0]
}

func (c *CPU) jumpAbs(opc *Opcode) int {
	var addr uint16

	switch opc.Operands[0].Name {
	case "a16":
		addr = uint16(c.Bus.Read(c.pc+2))<<8 | uint16(c.Bus.Read(c.pc+1))
	default:
		panic("unimplemented jump for " + opc.Operands[0].Name)
	}

	c.pc = addr

	return opc.Cycles[0]
}

func (c *CPU) jumpRel(opc *Opcode) int {
	var bytes int8

	cycles := opc.Cycles[0]

	switch opc.Operands[0].Name {
	case "e8":
		bytes = int8(c.Bus.Read(c.pc + 1))
	case "NZ":
		if !c.getZF() {
			bytes = int8(c.Bus.Read(c.pc + 1))
			cycles = opc.Cycles[1]
		}
	case "Z":
		if c.getZF() {
			bytes = int8(c.Bus.Read(c.pc + 1))
			cycles = opc.Cycles[1]
		}
	case "NC":
		if !c.getCF() {
			bytes = int8(c.Bus.Read(c.pc + 1))
			cycles = opc.Cycles[1]
		}
	case "C":
		if c.getCF() {
			bytes = int8(c.Bus.Read(c.pc + 1))
			cycles = opc.Cycles[1]
		}
	default:
		panic("unimplemented jump for " + opc.Operands[0].Name)
	}

	c.pc += opc.Bytes + uint16(bytes)

	return cycles
}

func (c *CPU) enableInterrupts(opc *Opcode) int {
	c.ime = true
	c.pc += opc.Bytes

	return opc.Cycles[0]
}

func (c *CPU) disableInterrupts(opc *Opcode) int {
	c.ime = false
	c.pc += opc.Bytes

	return opc.Cycles[0]
}

func (c *CPU) add(opc *Opcode) int {
	v := c.getOp(opc.Operands[1].Name)
	res := c.a + v

	c.setFlags(res == 0, false, (c.a&0xF)+(v&0xF) > 0xF, uint16(c.a)+uint16(v) > 0xFF)

	c.a = res
	c.pc += opc.Bytes

	return opc.Cycles[0]
}

func (c *CPU) sub(opc *Opcode) int {
	v := c.getOp(opc.Operands[1].Name)
	res := c.a - v

	c.setFlags(res == 0, true, c.a&0xF < v&0xF, c.a < v)

	c.a = res
	c.pc += opc.Bytes

	return opc.Cycles[0]
}

func (c *CPU) push(opc *Opcode) int {
	c.pushValue(c.getDOp(opc.Operands[0].Name))

	c.pc += opc.Bytes

	return opc.Cycles[0]
}

func (c *CPU) pop(opc *Opcode) int {
	c.setDOp(opc.Operands[0].Name, c.popValue())

	c.pc += opc.Bytes

	return opc.Cycles[0]
}

func (c *CPU) and(opc *Opcode) int {
	c.a &= c.getOp(opc.Operands[1].Name)

	c.setFlags(c.a == 0, false, true, false)

	c.pc += opc.Bytes

	return opc.Cycles[0]
}

func (c *CPU) or(opc *Opcode) int {
	c.a |= c.getOp(opc.Operands[1].Name)

	c.setFlags(c.a == 0, false, false, false)

	c.pc += opc.Bytes

	return opc.Cycles[0]
}

func (c *CPU) xor(opc *Opcode) int {
	c.a ^= c.getOp(opc.Operands[1].Name)

	c.setFlags(c.a == 0, false, false, false)

	c.pc += opc.Bytes

	return opc.Cycles[0]
}

func (c *CPU) cp(opc *Opcode) int {
	v := c.getOp(opc.Operands[1].Name)
	res := c.a - v

	c.setFlags(res == 0, true, c.a&0xF < v&0xF, c.a < v)

	c.pc += opc.Bytes

	return opc.Cycles[0]
}

// CB prefixed

func (c *CPU) rlc(opc *Opcode) int {
	v := c.getOp(opc.Operands[0].Name)

	sb := v & 0x80 >> 7
	res := v<<1 | sb

	c.setOp(opc.Operands[0].Name, res)
	c.setFlags(res == 0, false, false, sb == 1)

	c.pc += opc.Bytes

	return opc.Cycles[0]
}

func (c *CPU) srl(opc *Opcode) int {
	v := c.getOp(opc.Operands[0].Name)

	sb := v & 0x1
	res := v >> 1

	c.setOp(opc.Operands[0].Name, res)
	c.setFlags(res == 0, false, false, sb == 1)

	c.pc += opc.Bytes

	return opc.Cycles[0]
}

func (c *CPU) set(opc *Opcode) int {
	op0 := opc.Operands[0]
	op1 := opc.Operands[1]

	c.setOp(op1.Name, c.getOp(op1.Name)|1<<lib.Must(strconv.Atoi(op0.Name)))

	c.pc += opc.Bytes

	return opc.Cycles[0]
}

// Helpers

func (c *CPU) pushValue(value uint16) {
	c.Bus.Write(c.sp, uint8(value>>8))
	c.sp--
	c.Bus.Write(c.sp, uint8(value))
	c.sp--
}

func (c *CPU) popValue() uint16 {
	c.sp++
	lo := c.Bus.Read(c.sp)
	c.sp++
	hi := c.Bus.Read(c.sp)

	return uint16(hi)<<8 | uint16(lo)
}
