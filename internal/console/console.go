package console

import (
	"context"
	"fmt"

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
}

func Run(ctx context.Context, romBytes []uint8) error {
	gb := console{
		cpu:       &cpu.CPU{},
		memory:    &memory.Memory{},
		cartridge: &cartridge.Cartridge{},
		bus:       &bus.Bus{},
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

	fmt.Println("[console] loaded cartridge")

	for err == nil {
		err = gb.cpu.Step()
	}

	return err
}
