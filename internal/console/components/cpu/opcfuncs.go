package cpu

import (
	"strconv"
)

// Unprefixed

func (c *CPU) nop(opc *Opcode) int {
	return opc.Cycles[0]
}

func (c *CPU) call(opc *Opcode) int {
	op0 := opc.Operands[0]

	word := c.fetchWord()

	switch op0.Name {
	case "a16":
		c.pushValue(c.PC)
		c.PC = word
	case "NZ":
		if !c.getZF() {
			c.pushValue(c.PC)
			c.PC = word

			return opc.Cycles[1]
		}
	case "Z":
		if c.getZF() {
			c.pushValue(c.PC)
			c.PC = word

			return opc.Cycles[1]
		}
	case "NC":
		if !c.getCF() {
			c.pushValue(c.PC)
			c.PC = word

			return opc.Cycles[1]
		}
	case "C":
		if c.getCF() {
			c.pushValue(c.PC)
			c.PC = word

			return opc.Cycles[1]
		}
	default:
		panic("unimplemented call for " + op0.Name)
	}

	return opc.Cycles[0]
}

func (c *CPU) ret(opc *Opcode) int {
	if len(opc.Operands) == 0 {
		c.PC = c.popValue()

		return opc.Cycles[0]
	} else {
		op0 := opc.Operands[0]
		switch op0.Name {
		case "NC":
			if !c.getCF() {
				c.PC = c.popValue()
				return opc.Cycles[1]
			}
		case "C":
			if c.getCF() {
				c.PC = c.popValue()
				return opc.Cycles[1]
			}
		case "NZ":
			if !c.getZF() {
				c.PC = c.popValue()
				return opc.Cycles[1]
			}
		case "Z":
			if c.getZF() {
				c.PC = c.popValue()
				return opc.Cycles[1]
			}
		}
	}

	return opc.Cycles[0]
}

func (c *CPU) reti(opc *Opcode) int {
	c.PC = c.popValue()
	c.IME = true

	return opc.Cycles[0]
}

func (c *CPU) rst(opc *Opcode) int {
	c.pushValue(c.PC)

	v, err := strconv.ParseUint(opc.Operands[0].Name[1:], 16, 16)
	if err != nil {
		panic(err)
	}

	c.PC = uint16(v)

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
		v8 = c.fetchByte()
	case "n16":
		v16 = c.fetchWord()
	case "a16":
		v8 = c.Bus.Read(c.fetchWord())
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
		v16 = c.SP

		if len(opc.Operands) == 3 {
			v := int8(c.fetchByte())
			c.setFlags(false, false, (c.SP&0xF)+uint16(v&0xF) > 0xF, c.SP&0xFF+uint16(v)&0xFF > 0xFF)
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
		word := c.fetchWord()
		if op1.Name == "SP" {
			c.Bus.Write(word, uint8(c.SP))
			c.Bus.Write(word+1, uint8(c.SP>>8))
		}

		if op1.Name == "A" {
			c.Bus.Write(word, c.A)
		}
	default:
		panic("unimplemented op0 for load: " + op0.Name)
	}

	return opc.Cycles[0]
}

func (c *CPU) loadH(opc *Opcode) int {
	op0 := opc.Operands[0]
	op1 := opc.Operands[1]

	switch op0.Name {
	case "a8":
		c.Bus.Write(0xFF00|uint16(c.fetchByte()), c.A)
	case "C":
		c.Bus.Write(0xFF00|uint16(c.C), c.A)
	case "A":
		if op1.Name == "a8" {
			c.A = c.Bus.Read(0xFF00 | uint16(c.fetchByte()))
		}

		if op1.Name == "C" {
			c.A = c.Bus.Read(0xFF00 | uint16(c.C))
		}
	default:
		panic("unimplemented loadH for " + op0.Name)
	}

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

	return opc.Cycles[0]
}

func (c *CPU) jumpAbs(opc *Opcode) int {
	switch opc.Operands[0].Name {
	case "a16":
		c.PC = c.fetchWord()

		return opc.Cycles[0]
	case "HL":
		c.PC = c.getDOp("HL")

		return opc.Cycles[0]
	case "NZ":
		word := c.fetchWord()
		if !c.getZF() {
			c.PC = word

			return opc.Cycles[1]
		}
	case "Z":
		word := c.fetchWord()
		if c.getZF() {
			c.PC = word

			return opc.Cycles[1]
		}
	case "NC":
		word := c.fetchWord()
		if !c.getCF() {
			c.PC = word

			return opc.Cycles[1]
		}
	case "C":
		word := c.fetchWord()
		if c.getCF() {
			c.PC = word

			return opc.Cycles[1]
		}
	default:
		panic("unimplemented jump for " + opc.Operands[0].Name)
	}

	return opc.Cycles[0]
}

func (c *CPU) jumpRel(opc *Opcode) int {
	bytes := c.fetchByte()

	cycles := opc.Cycles[0]

	switch opc.Operands[0].Name {
	case "e8":
		c.PC += uint16(int8(bytes))
	case "NZ":
		if !c.getZF() {
			c.PC += uint16(int8(bytes))
			cycles = opc.Cycles[1]
		}
	case "Z":
		if c.getZF() {
			c.PC += uint16(int8(bytes))
			cycles = opc.Cycles[1]
		}
	case "NC":
		if !c.getCF() {
			c.PC += uint16(int8(bytes))
			cycles = opc.Cycles[1]
		}
	case "C":
		if c.getCF() {
			c.PC += uint16(int8(bytes))
			cycles = opc.Cycles[1]
		}
	default:
		panic("unimplemented jump for " + opc.Operands[0].Name)
	}

	return cycles
}

func (c *CPU) ei(opc *Opcode) int {
	c.IMEScheduled = true

	return opc.Cycles[0]
}

func (c *CPU) di(opc *Opcode) int {
	c.IME = false

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

		res := c.A + v + cy

		c.setFlags(res == 0, false, (c.A&0xF)+(v&0xF+cy) > 0xF, uint16(c.A)+uint16(v)+uint16(cy) > 0xFF)

		c.A = res
	} else {
		switch op0.Name {
		case "HL":
			v1 := c.getDOp(op0.Name)
			v2 := c.getDOp(op1.Name)

			c.setFlags(c.getZF(), false, uint32(v1&0xFFF)+uint32(v2&0xFFF) > 0xFFF, uint32(v1)+uint32(v2) > 0xFFFF)
			c.setDOp(op0.Name, v1+v2)
		case "SP":
			v := int8(c.fetchByte())
			c.setFlags(false, false, (c.SP&0xF)+uint16(v&0xF) > 0xF, c.SP&0xFF+uint16(v)&0xFF > 0xFF)

			c.SP += uint16(v)
		}
	}

	return opc.Cycles[0]
}

func (c *CPU) sub(opc *Opcode) int {
	v := c.getOp(opc.Operands[1].Name)

	cy := uint8(0)

	if opc.Mnemonic == "SBC" && c.getCF() {
		cy = 1
	}

	res := c.A - v - cy

	c.setFlags(res == 0, true, c.A&0xF < v&0xF+cy, uint16(c.A&0xFF) < uint16(v&0xFF)+uint16(cy))

	c.A = res

	return opc.Cycles[0]
}

func (c *CPU) push(opc *Opcode) int {
	c.pushValue(c.getDOp(opc.Operands[0].Name))

	return opc.Cycles[0]
}

func (c *CPU) pop(opc *Opcode) int {
	v := c.popValue()

	if opc.Operands[0].Name == "AF" {
		v &= 0xFFF0
	}

	c.setDOp(opc.Operands[0].Name, v)

	return opc.Cycles[0]
}

func (c *CPU) and(opc *Opcode) int {
	c.A &= c.getOp(opc.Operands[1].Name)

	c.setFlags(c.A == 0, false, true, false)

	return opc.Cycles[0]
}

func (c *CPU) or(opc *Opcode) int {
	c.A |= c.getOp(opc.Operands[1].Name)

	c.setFlags(c.A == 0, false, false, false)

	return opc.Cycles[0]
}

func (c *CPU) xor(opc *Opcode) int {
	c.A ^= c.getOp(opc.Operands[1].Name)

	c.setFlags(c.A == 0, false, false, false)

	return opc.Cycles[0]
}

func (c *CPU) cp(opc *Opcode) int {
	v := c.getOp(opc.Operands[1].Name)
	res := c.A - v

	c.setFlags(res == 0, true, c.A&0xF < v&0xF, c.A < v)

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

		res = c.A - value
	} else {
		if c.A&0x0F > 0x09 || c.getHF() {
			value += 0x06
		}

		if (c.A+value)&0xF0 > 0x90 || cy || c.A > 0x99 {
			value += 0x60
			cy = true
		}

		res = c.A + value
	}

	c.setFlags(res == 0, c.getNF(), false, cy)

	c.A = res

	return opc.Cycles[0]
}

func (c *CPU) rra(opc *Opcode) int {
	v := c.A

	cy := uint8(0)
	if c.getCF() {
		cy = 1
	}

	sb := v & 0x1
	res := v>>1 | cy<<7

	c.A = res
	c.setFlags(false, false, false, sb == 1)

	return opc.Cycles[0]
}

func (c *CPU) cpl(opc *Opcode) int {
	c.A = ^c.A

	c.setFlags(c.getZF(), true, true, c.getCF())

	return opc.Cycles[0]
}

func (c *CPU) scf(opc *Opcode) int {
	c.setFlags(c.getZF(), false, false, true)

	return opc.Cycles[0]
}

func (c *CPU) ccf(opc *Opcode) int {
	c.setFlags(c.getZF(), false, false, !c.getCF())

	return opc.Cycles[0]
}

func (c *CPU) halt(opc *Opcode) int {
	if !c.IME && (c.IE&c.IFF&0x1F != 0) {
		c.HaltBug = true
	} else {
		c.Halted = true
	}

	return opc.Cycles[0]
}

func (c *CPU) stop(opc *Opcode) int {
	c.fetchByte()
	c.Console.Stop()

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

	return opc.Cycles[0]
}

func (c *CPU) rl(opc *Opcode) int {
	v := uint8(0)
	if opc.Mnemonic == "RLA" {
		v = c.A
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
		c.A = res
		c.setFlags(false, false, false, sb == 1)
	} else {
		c.setOp(opc.Operands[0].Name, res)
		c.setFlags(res == 0, false, false, sb == 1)
	}

	return opc.Cycles[0]
}

func (c *CPU) rlc(opc *Opcode) int {
	v := uint8(0)
	if opc.Mnemonic == "RLCA" {
		v = c.A
	} else {
		v = c.getOp(opc.Operands[0].Name)
	}

	sb := v & 0x80 >> 7
	res := v<<1 | sb

	if opc.Mnemonic == "RLCA" {
		c.A = res
		c.setFlags(false, false, false, sb == 1)
	} else {
		c.setOp(opc.Operands[0].Name, res)
		c.setFlags(res == 0, false, false, sb == 1)
	}

	return opc.Cycles[0]
}

func (c *CPU) rrc(opc *Opcode) int {
	v := uint8(0)
	if opc.Mnemonic == "RRCA" {
		v = c.A
	} else {
		v = c.getOp(opc.Operands[0].Name)
	}

	sb := v & 0x1
	res := v>>1 | sb<<7

	if opc.Mnemonic == "RRCA" {
		c.A = res
		c.setFlags(false, false, false, sb == 1)
	} else {
		c.setOp(opc.Operands[0].Name, res)
		c.setFlags(res == 0, false, false, sb == 1)
	}

	return opc.Cycles[0]
}

func (c *CPU) sla(opc *Opcode) int {
	v := c.getOp(opc.Operands[0].Name)

	sb := v & 0x80 >> 7
	res := v << 1

	c.setOp(opc.Operands[0].Name, res)
	c.setFlags(res == 0, false, false, sb == 1)

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

	return opc.Cycles[0]
}

func (c *CPU) swap(opc *Opcode) int {
	v := c.getOp(opc.Operands[0].Name)
	v = v<<4 | v>>4

	c.setOp(opc.Operands[0].Name, v)
	c.setFlags(v == 0, false, false, false)

	return opc.Cycles[0]
}

func (c *CPU) set(opc *Opcode) int {
	op0 := opc.Operands[0]
	op1 := opc.Operands[1]

	bit, err := strconv.Atoi(op0.Name)
	if err != nil {
		panic(err)
	}

	c.setOp(op1.Name, c.getOp(op1.Name)|1<<bit)

	return opc.Cycles[0]
}

func (c *CPU) res(opc *Opcode) int {
	op0 := opc.Operands[0]
	op1 := opc.Operands[1]

	bit, err := strconv.Atoi(op0.Name)
	if err != nil {
		panic(err)
	}

	c.setOp(op1.Name, c.getOp(op1.Name)&^(1<<bit))

	return opc.Cycles[0]
}

func (c *CPU) bit(opc *Opcode) int {
	op0 := opc.Operands[0]
	op1 := opc.Operands[1]

	bit, err := strconv.Atoi(op0.Name)
	if err != nil {
		panic(err)
	}

	mask := uint8(1 << bit)
	v := c.getOp(op1.Name)
	res := v & mask >> bit

	c.setFlags(res == 0, false, true, c.getCF())

	return opc.Cycles[0]
}

// Helpers

func (c *CPU) pushValue(value uint16) {
	c.SP--
	c.Bus.Write(c.SP, uint8(value>>8))
	c.SP--
	c.Bus.Write(c.SP, uint8(value))
}

func (c *CPU) popValue() uint16 {
	lo := c.Bus.Read(c.SP)
	c.SP++
	hi := c.Bus.Read(c.SP)
	c.SP++

	return uint16(hi)<<8 | uint16(lo)
}
