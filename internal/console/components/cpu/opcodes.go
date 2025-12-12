package cpu

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

type Operand struct {
	Name      string `json:"name"`
	Bytes     int    `json:"bytes"`
	Immediate bool   `json:"immediate"`
	Increment bool   `json:"increment"`
	Decrement bool   `json:"decrement"`
}

type Opcode struct {
	Func      OpcodeFunc
	Flags     map[string]string `json:"flags"`
	Mnemonic  string            `json:"mnemonic"`
	Cycles    []int             `json:"cycles"`
	Operands  []Operand         `json:"operands"`
	Bytes     uint16            `json:"bytes"`
	Immediate bool              `json:"immediate"`
}

type Opcodes struct {
	Unprefixed map[string]Opcode `json:"unprefixed"`
	CBPrefixed map[string]Opcode `json:"cbprefixed"`
}

type OpcodeFunc func(*Opcode) int

var (
	//go:embed opcodes.json
	opcodes           []uint8
	UnprefixedOpcodes [256]Opcode
	CBPrefixedOpcodes [256]Opcode
)

func ParseOpcodes() error {
	instructions := Opcodes{}

	err := json.Unmarshal(opcodes, &instructions)
	if err != nil {
		return fmt.Errorf("failed to unmarshal opcode: %w", err)
	}

	for i := range 256 {
		hex := fmt.Sprintf("0x%02X", i)
		UnprefixedOpcodes[i] = instructions.Unprefixed[hex]
		CBPrefixedOpcodes[i] = instructions.CBPrefixed[hex]
	}

	return nil
}

func (c *CPU) bindOpcodeFuncs() {
	opcodeFuncs := map[string]OpcodeFunc{
		"NOP":    c.nop,
		"PREFIX": c.nop,
		"JP":     c.jumpAbs,
		"JR":     c.jumpRel,
		"LD":     c.load,
		"INC":    c.inc,
		"DEC":    c.dec,
		"EI":     c.enableInterrupts,
		"DI":     c.disableInterrupts,
		"LDH":    c.loadH,
		"CALL":   c.call,
		"RET":    c.ret,
		"RETI":   c.reti,
		"RST":    c.rst,
		"ADD":    c.add,
		"ADC":    c.add,
		"SUB":    c.sub,
		"SBC":    c.sub,
		"PUSH":   c.push,
		"POP":    c.pop,
		"AND":    c.and,
		"OR":     c.or,
		"XOR":    c.xor,
		"CP":     c.cp,
		"DAA":    c.daa,
		"RRA":    c.rra,
		"RL":     c.rl,
		"RLA":    c.rl,
		"RR":     c.rr,
		"CPL":    c.cpl,
		"SCF":    c.scf,
		"CCF":    c.ccf,
		"RLC":    c.rlc,
		"RLCA":   c.rlc,
		"RRC":    c.rrc,
		"RRCA":   c.rrc,
		"SLA":    c.sla,
		"SRL":    c.srla,
		"SRA":    c.srla,
		"SWAP":   c.swap,
		"SET":    c.set,
	}

	for i := range 256 {
		UnprefixedOpcodes[i].Func = opcodeFuncs[UnprefixedOpcodes[i].Mnemonic]
		CBPrefixedOpcodes[i].Func = opcodeFuncs[CBPrefixedOpcodes[i].Mnemonic]
	}
}
