package console

import (
	"context"
	"fmt"
	"strings"

	"github.com/cterence/gbgo/internal/console/components/bus"
	"github.com/cterence/gbgo/internal/console/components/cartridge"
	"github.com/cterence/gbgo/internal/console/components/cpu"
	"github.com/cterence/gbgo/internal/console/components/memory"
)

type console struct {
	cpu       *cpu.CPU
	memory    *memory.Memory
	cartridge *cartridge.Cartridge
	bus       *bus.Bus

	stopCPUAfter int
}

type Option func(*console)

func WithStopCPUAfter(stopCPUAfter int) Option {
	return func(c *console) {
		c.stopCPUAfter = stopCPUAfter
	}
}

func Run(ctx context.Context, romBytes []uint8, options ...Option) error {
	gb := console{
		cpu:       &cpu.CPU{},
		memory:    &memory.Memory{},
		cartridge: &cartridge.Cartridge{},
		bus:       &bus.Bus{},
	}

	for _, o := range options {
		o(&gb)
	}

	gb.bus.Memory = gb.memory
	gb.bus.Cartridge = gb.cartridge
	gb.cpu.Bus = gb.bus

	err := gb.cartridge.Init(len(romBytes))
	if err != nil {
		return fmt.Errorf("failed to init cartridge: %w", err)
	}

	err = gb.cpu.Init()
	if err != nil {
		return fmt.Errorf("failed to init CPU: %w", err)
	}

	// Load cartridge
	for i, b := range romBytes {
		gb.cartridge.Write(uint32(i), b)
	}

	// fmt.Println("[console] loaded cartridge")

	cycles, totalCycles := 0, 0

	for err == nil {
		cycles, err = gb.cpu.Step()
		totalCycles += cycles

		if gb.stopCPUAfter > 0 && totalCycles > gb.stopCPUAfter {
			err = fmt.Errorf("stopping CPU after %d cycles", gb.stopCPUAfter)
		}
	}

	return err
}

func Disassemble(romBytes []uint8) error {
	pc := 0

	if err := cpu.ParseOpcodes(); err != nil {
		return fmt.Errorf("failed to parse CPU opcodes: %w", err)
	}

	var sb strings.Builder

	for pc < len(romBytes) {
		opcode := cpu.UnprefixedOpcodes[romBytes[pc]]

		if opcode.Mnemonic == "PREFIX" {
			pc++
			opcode = cpu.CBPrefixedOpcodes[romBytes[pc]]
		}

		operands := ""

		var operandsSb97 strings.Builder
		for _, op := range opcode.Operands {
			operandsSb97.WriteString(fmt.Sprintf(" %-3s", op.Name))
		}

		operands += operandsSb97.String()

		sb.WriteString(fmt.Sprintf("%04X - %-4s%s\n", pc, opcode.Mnemonic, operands))

		pc += int(opcode.Bytes)
	}

	fmt.Print(sb.String())

	return nil
}
