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
	Bytes     int               `json:"bytes"`
	Immediate bool              `json:"immediate"`
}

type Opcodes struct {
	Unprefixed map[string]Opcode `json:"unprefixed"`
	CBPrefixed map[string]Opcode `json:"cbprefixed"`
}

type OpcodeFunc func(*Opcode)

var (
	//go:embed opcodes.json
	opcodes           []uint8
	unprefixedOpcodes [256]Opcode
	cbprefixedOpcodes [256]Opcode
)

func (c *CPU) parseOpcodes() error {
	instructions := Opcodes{}

	opcodeFuncs := map[string]OpcodeFunc{
		"NOP":    c.nop,
		"PREFIX": c.prefix,
		"RLC":    c.rlc,
		"JP":     c.jump,
		"LD":     c.load,
		"INC":    c.inc,
	}

	err := json.Unmarshal(opcodes, &instructions)
	if err != nil {
		return fmt.Errorf("failed to unmarshal opcode: %w", err)
	}

	for i := range 256 {
		hex := fmt.Sprintf("0x%02X", i)
		unprefixedOpcodes[i] = instructions.Unprefixed[hex]
		unprefixedOpcodes[i].Func = opcodeFuncs[unprefixedOpcodes[i].Mnemonic]
		cbprefixedOpcodes[i] = instructions.CBPrefixed[hex]
		cbprefixedOpcodes[i].Func = opcodeFuncs[cbprefixedOpcodes[i].Mnemonic]
	}

	return nil
}
