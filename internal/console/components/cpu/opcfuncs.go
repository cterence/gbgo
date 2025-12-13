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
	if len(opc.Operands) == 0 {
		c.pc = c.popValue()

		return opc.Cycles[0]
	} else {
		op0 := opc.Operands[0]
		switch op0.Name {
		case "NC":
			if !c.getCF() {
				c.pc = c.popValue()
				return opc.Cycles[1]
			}
		case "C":
			if c.getCF() {
				c.pc = c.popValue()
				return opc.Cycles[1]
			}
		case "NZ":
			if !c.getZF() {
				c.pc = c.popValue()
				return opc.Cycles[1]
			}
		case "Z":
			if c.getZF() {
				c.pc = c.popValue()
				return opc.Cycles[1]
			}
		}
	}

	c.pc += opc.Bytes

	return opc.Cycles[0]
}

func (c *CPU) reti(opc *Opcode) int {
	c.pc = c.popValue()
	c.imeScheduled = true

	return opc.Cycles[0]
}

func (c *CPU) rst(opc *Opcode) int {
	c.pushValue(c.pc + opc.Bytes)

	c.pc = uint16(lib.Must(strconv.ParseUint(opc.Operands[0].Name[1:], 16, 16)))

	return opc.Cycles[0]
}

func (c *CPU) load(opc *Opcode) int {
	op0 := opc.Operands[0]
	op1 := opc.Operands[1]
	v8 := uint8(0)
	v16 := uint16(0)

	switch op1.Name {
	case "A", "F", "B", "C", "D", "E", "H", "L":
		v8 = c.getOp(op1.Name)
	case "n8":
		v8 = c.Bus.Read(c.pc + 1)
	case "n16":
		v16 = uint16(c.Bus.Read(c.pc+2))<<8 | uint16(c.Bus.Read(c.pc+1))
	case "a16":
		v8 = c.Bus.Read(uint16(c.Bus.Read(c.pc+2))<<8 | uint16(c.Bus.Read(c.pc+1)))
	case "BC", "DE", "HL":
		if !op1.Immediate {
			v8 = c.Bus.Read(c.getDOp(op1.Name))

			if op1.Increment {
				c.setDOp(op1.Name, c.getDOp(op1.Name)+1)
			}

			if op1.Decrement {
				c.setDOp(op1.Name, c.getDOp(op1.Name)-1)
			}
		} else {
			v16 = c.getDOp(op1.Name)
		}
	case "SP":
		v16 = c.sp

		if len(opc.Operands) == 3 {
			v := int8(c.Bus.Read(c.pc + 1))
			c.setFlags(false, false, (c.sp&0xF)+uint16(v&0xF) > 0xF, c.sp&0xFF+uint16(v)&0xFF > 0xFF)
			v16 += uint16(v)
		}
	default:
		panic("unsupported op1 for load: " + op1.Name)
	}

	switch op0.Name {
	case "A", "F", "B", "C", "D", "E", "H", "L":
		c.setOp(op0.Name, v8)
	case "BC", "DE", "HL", "SP":
		if !op0.Immediate {
			c.Bus.Write(c.getDOp(op0.Name), v8)

			if op0.Increment {
				c.setDOp(op0.Name, c.getDOp(op0.Name)+1)
			}

			if op0.Decrement {
				c.setDOp(op0.Name, c.getDOp(op0.Name)-1)
			}
		} else {
			c.setDOp(op0.Name, v16)
		}
	case "a16":
		hi := uint16(c.Bus.Read(c.pc + 2))
		lo := uint16(c.Bus.Read(c.pc + 1))

		if op1.Name == "SP" {
			c.Bus.Write(hi<<8|lo, uint8(c.sp))
			c.Bus.Write(hi<<8|lo+1, uint8(c.sp>>8))
		}

		if op1.Name == "A" {
			c.Bus.Write(hi<<8|lo, c.a)
		}
	default:
		panic("unimplemented op0 for load: " + op0.Name)
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
		c.Bus.Write(0xFF00|uint16(c.c), c.a)
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
		if op0.Immediate {
			c.setDOp(op0.Name, c.getDOp(op0.Name)+1)
		} else {
			addr := c.getDOp(op0.Name)
			v := c.Bus.Read(addr)
			res := v + 1
			c.Bus.Write(addr, res)
			c.setFlags(res == 0, false, v&0xF+1 > 0xF, c.getCF())
		}
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
		if op0.Immediate {
			c.setDOp(op0.Name, c.getDOp(op0.Name)-1)
		} else {
			addr := c.getDOp(op0.Name)
			v := c.Bus.Read(addr)
			res := v - 1
			c.Bus.Write(addr, res)
			c.setFlags(res == 0, true, v&0xF < v&0xF-1, c.getCF())
		}
	}

	c.pc += opc.Bytes

	return opc.Cycles[0]
}

func (c *CPU) jumpAbs(opc *Opcode) int {
	switch opc.Operands[0].Name {
	case "a16":
		c.pc = uint16(c.Bus.Read(c.pc+2))<<8 | uint16(c.Bus.Read(c.pc+1))

		return opc.Cycles[0]
	case "HL":
		c.pc = c.getDOp("HL")

		return opc.Cycles[0]
	case "NZ":
		if !c.getZF() {
			c.pc = uint16(c.Bus.Read(c.pc+2))<<8 | uint16(c.Bus.Read(c.pc+1))

			return opc.Cycles[1]
		}
	case "Z":
		if c.getZF() {
			c.pc = uint16(c.Bus.Read(c.pc+2))<<8 | uint16(c.Bus.Read(c.pc+1))

			return opc.Cycles[1]
		}
	case "NC":
		if !c.getCF() {
			c.pc = uint16(c.Bus.Read(c.pc+2))<<8 | uint16(c.Bus.Read(c.pc+1))

			return opc.Cycles[1]
		}
	case "C":
		if c.getCF() {
			c.pc = uint16(c.Bus.Read(c.pc+2))<<8 | uint16(c.Bus.Read(c.pc+1))

			return opc.Cycles[1]
		}
	default:
		panic("unimplemented jump for " + opc.Operands[0].Name)
	}

	c.pc += opc.Bytes

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

func (c *CPU) ei(opc *Opcode) int {
	c.imeScheduled = true
	c.pc += opc.Bytes

	return opc.Cycles[0]
}

func (c *CPU) di(opc *Opcode) int {
	c.ime = false
	c.pc += opc.Bytes

	return opc.Cycles[0]
}

func (c *CPU) add(opc *Opcode) int {
	op0 := opc.Operands[0]
	op1 := opc.Operands[1]

	if len(op0.Name) == 1 {
		v := c.getOp(op1.Name)
		cy := uint8(0)

		if opc.Mnemonic == "ADC" && c.getCF() {
			cy = 1
		}

		res := c.a + v + cy

		c.setFlags(res == 0, false, (c.a&0xF)+(v&0xF+cy) > 0xF, uint16(c.a)+uint16(v)+uint16(cy) > 0xFF)

		c.a = res
	} else {
		switch op0.Name {
		case "HL":
			v1 := c.getDOp(op0.Name)
			v2 := c.getDOp(op1.Name)

			c.setFlags(c.getZF(), false, uint32(v1&0xFFF)+uint32(v2&0xFFF) > 0xFFF, uint32(v1)+uint32(v2) > 0xFFFF)
			c.setDOp(op0.Name, v1+v2)
		case "SP":
			v := int8(c.Bus.Read(c.pc + 1))
			c.setFlags(false, false, (c.sp&0xF)+uint16(v&0xF) > 0xF, c.sp&0xFF+uint16(v)&0xFF > 0xFF)

			c.sp += uint16(v)
		}
	}

	c.pc += opc.Bytes

	return opc.Cycles[0]
}

func (c *CPU) sub(opc *Opcode) int {
	v := c.getOp(opc.Operands[1].Name)

	cy := uint8(0)

	if opc.Mnemonic == "SBC" && c.getCF() {
		cy = 1
	}

	res := c.a - v - cy

	c.setFlags(res == 0, true, c.a&0xF < v&0xF+cy, uint16(c.a&0xFF) < uint16(v&0xFF)+uint16(cy))

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
	v := c.popValue()

	if opc.Operands[0].Name == "AF" {
		v &= 0xFFF0
	}

	c.setDOp(opc.Operands[0].Name, v)

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

func (c *CPU) daa(opc *Opcode) int {
	cy := c.getCF()
	value := uint8(0)
	res := uint8(0)

	if c.getNF() {
		if c.getHF() {
			value += 0x06
		}

		if cy {
			value += 0x60
		}

		res = c.a - value
	} else {
		if c.a&0x0F > 0x09 || c.getHF() {
			value += 0x06
		}

		if (c.a+value)&0xF0 > 0x90 || cy || c.a > 0x99 {
			value += 0x60
			cy = true
		}

		res = c.a + value
	}

	c.setFlags(res == 0, c.getNF(), false, cy)

	c.a = res
	c.pc += opc.Bytes

	return opc.Cycles[0]
}

func (c *CPU) rra(opc *Opcode) int {
	v := c.a

	cy := uint8(0)
	if c.getCF() {
		cy = 1
	}

	sb := v & 0x1
	res := v>>1 | cy<<7

	c.a = res
	c.setFlags(false, false, false, sb == 1)

	c.pc += opc.Bytes

	return opc.Cycles[0]
}

func (c *CPU) cpl(opc *Opcode) int {
	c.a = ^c.a
	c.pc += opc.Bytes

	c.setFlags(c.getZF(), true, true, c.getCF())

	return opc.Cycles[0]
}

func (c *CPU) scf(opc *Opcode) int {
	c.setFlags(c.getZF(), false, false, true)
	c.pc += opc.Bytes

	return opc.Cycles[0]
}

func (c *CPU) ccf(opc *Opcode) int {
	c.setFlags(c.getZF(), false, false, !c.getCF())
	c.pc += opc.Bytes

	return opc.Cycles[0]
}

func (c *CPU) halt(opc *Opcode) int {
	c.halted = true
	c.pc += opc.Bytes

	return opc.Cycles[0]
}

// CB prefixed

func (c *CPU) rr(opc *Opcode) int {
	v := c.getOp(opc.Operands[0].Name)

	cy := uint8(0)
	if c.getCF() {
		cy = 1
	}

	sb := v & 0x1
	res := v>>1 | cy<<7

	c.setOp(opc.Operands[0].Name, res)
	c.setFlags(res == 0, false, false, sb == 1)

	c.pc += opc.Bytes

	return opc.Cycles[0]
}

func (c *CPU) rl(opc *Opcode) int {
	v := uint8(0)
	if opc.Mnemonic == "RLA" {
		v = c.a
	} else {
		v = c.getOp(opc.Operands[0].Name)
	}

	cy := uint8(0)
	if c.getCF() {
		cy = 1
	}

	sb := v & 0x80 >> 7
	res := v<<1 | cy

	if opc.Mnemonic == "RLA" {
		c.a = res
		c.setFlags(false, false, false, sb == 1)
	} else {
		c.setOp(opc.Operands[0].Name, res)
		c.setFlags(res == 0, false, false, sb == 1)
	}

	c.pc += opc.Bytes

	return opc.Cycles[0]
}

func (c *CPU) rlc(opc *Opcode) int {
	v := uint8(0)
	if opc.Mnemonic == "RLCA" {
		v = c.a
	} else {
		v = c.getOp(opc.Operands[0].Name)
	}

	sb := v & 0x80 >> 7
	res := v<<1 | sb

	if opc.Mnemonic == "RLCA" {
		c.a = res
		c.setFlags(false, false, false, sb == 1)
	} else {
		c.setOp(opc.Operands[0].Name, res)
		c.setFlags(res == 0, false, false, sb == 1)
	}

	c.pc += opc.Bytes

	return opc.Cycles[0]
}

func (c *CPU) rrc(opc *Opcode) int {
	v := uint8(0)
	if opc.Mnemonic == "RRCA" {
		v = c.a
	} else {
		v = c.getOp(opc.Operands[0].Name)
	}

	sb := v & 0x1
	res := v>>1 | sb<<7

	if opc.Mnemonic == "RRCA" {
		c.a = res
		c.setFlags(false, false, false, sb == 1)
	} else {
		c.setOp(opc.Operands[0].Name, res)
		c.setFlags(res == 0, false, false, sb == 1)
	}

	c.pc += opc.Bytes

	return opc.Cycles[0]
}

func (c *CPU) sla(opc *Opcode) int {
	v := c.getOp(opc.Operands[0].Name)

	sb := v & 0x80 >> 7
	res := v << 1

	c.setOp(opc.Operands[0].Name, res)
	c.setFlags(res == 0, false, false, sb == 1)

	c.pc += opc.Bytes

	return opc.Cycles[0]
}

func (c *CPU) srla(opc *Opcode) int {
	v := c.getOp(opc.Operands[0].Name)

	sb := v & 0x1
	b7 := v & 0x80
	res := v >> 1

	if opc.Mnemonic == "SRA" {
		res |= b7
	}

	c.setOp(opc.Operands[0].Name, res)
	c.setFlags(res == 0, false, false, sb == 1)

	c.pc += opc.Bytes

	return opc.Cycles[0]
}

func (c *CPU) swap(opc *Opcode) int {
	v := c.getOp(opc.Operands[0].Name)
	v = v<<4 | v>>4

	c.setOp(opc.Operands[0].Name, v)
	c.setFlags(v == 0, false, false, false)

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

func (c *CPU) res(opc *Opcode) int {
	op0 := opc.Operands[0]
	op1 := opc.Operands[1]

	c.setOp(op1.Name, c.getOp(op1.Name)&^(1<<lib.Must(strconv.Atoi(op0.Name))))

	c.pc += opc.Bytes

	return opc.Cycles[0]
}

func (c *CPU) bit(opc *Opcode) int {
	op0 := opc.Operands[0]
	op1 := opc.Operands[1]

	bit := lib.Must(strconv.Atoi(op0.Name))
	mask := uint8(1 << bit)
	v := c.getOp(op1.Name)
	res := v & mask >> bit

	c.setFlags(res == 0, false, true, c.getCF())

	c.pc += opc.Bytes

	return opc.Cycles[0]
}

// Helpers

func (c *CPU) pushValue(value uint16) {
	c.sp--
	c.Bus.Write(c.sp, uint8(value>>8))
	c.sp--
	c.Bus.Write(c.sp, uint8(value))
}

func (c *CPU) popValue() uint16 {
	lo := c.Bus.Read(c.sp)
	c.sp++
	hi := c.Bus.Read(c.sp)
	c.sp++

	return uint16(hi)<<8 | uint16(lo)
}
