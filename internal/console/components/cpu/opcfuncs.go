package cpu

func (c *CPU) nop(*Opcode) {}

func (c *CPU) prefix(*Opcode) {
	c.nextOpcodePrefixed = true
}

func (c *CPU) load(opc *Opcode) {
	op0 := opc.Operands[0]
	op1 := opc.Operands[1]

	switch op0.Name {
	case "A", "F", "B", "C", "D", "E", "H", "L":
		switch op1.Name {
		case "A", "F", "B", "C", "D", "E", "H", "L":
			c.setOp(op0.Name, c.getOp(op1.Name))
		case "n8":
			c.setOp(op0.Name, c.Bus.Read(c.pc+1))
		case "HL":
			c.setOp(op0.Name, c.Bus.Read(c.getDOp("HL")))

			if op1.Increment {
				c.setDOp("HL", c.getDOp("HL")+1)
			}

			if op1.Decrement {
				c.setDOp("HL", c.getDOp("HL")-1)
			}
		default:
			panic("unimplemented load for " + op1.Name)
		}
	case "BC", "DE", "HL", "SP":
		imm16 := uint16(c.Bus.Read(c.pc+2))<<8 | uint16(c.Bus.Read(c.pc+1))
		c.setDOp(op0.Name, imm16)
	default:
		panic("unimplemented load for " + op0.Name)
	}

	c.pc += uint16(opc.Operands[1].Bytes) + 1
}

func (c *CPU) inc(opc *Opcode) {
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
}

func (c *CPU) jump(opc *Opcode) {
	var addr uint16

	switch opc.Operands[0].Name {
	case "a16":
		addr = uint16(c.Bus.Read(c.pc+2))<<8 | uint16(c.Bus.Read(c.pc+1))
	default:
		panic("unimplemented jump for " + opc.Operands[0].Name)
	}

	c.pc = addr
}

func (c *CPU) rlc(opc *Opcode) {
	v := c.getOp(opc.Operands[0].Name)

	sb := v & 0x80 >> 7
	res := v<<1 | sb

	c.setOp(opc.Operands[0].Name, res)
	c.setFlags(res == 0, false, false, sb == 1)

	c.pc += uint16(opc.Operands[0].Bytes) + 1
}
