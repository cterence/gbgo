package console

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cterence/gbgo/internal/console/components/bus"
	"github.com/cterence/gbgo/internal/console/components/cartridge"
	"github.com/cterence/gbgo/internal/console/components/cpu"
	"github.com/cterence/gbgo/internal/console/components/dma"
	"github.com/cterence/gbgo/internal/console/components/memory"
	"github.com/cterence/gbgo/internal/console/components/ppu"
	"github.com/cterence/gbgo/internal/console/components/serial"
	"github.com/cterence/gbgo/internal/console/components/timer"
	"github.com/cterence/gbgo/internal/console/components/ui"
	"github.com/cterence/gbgo/internal/log"
)

const (
	CPU_FREQ     = 4194304
	FPS          = 60
	FRAME_CYCLES = 70224
	FRAME_TIME   = time.Second / FPS
)

type console struct {
	cpu       *cpu.CPU
	memory    *memory.Memory
	cartridge *cartridge.Cartridge
	bus       *bus.Bus
	timer     *timer.Timer
	ui        *ui.UI
	ppu       *ppu.PPU
	serial    *serial.Serial
	dma       *dma.DMA

	cancel context.CancelFunc

	cpuOptions    []cpu.Option
	busOptions    []bus.Option
	serialOptions []serial.Option

	headless bool
	stopped  bool
	paused   bool
}

type Option func(*console)

func WithHeadless(headless bool) Option {
	return func(c *console) {
		c.headless = headless
	}
}

func WithPrintSerial(printSerial bool) Option {
	return func(c *console) {
		c.serialOptions = append(c.serialOptions, serial.WithPrintSerial(printSerial))
	}
}

func WithBootROM(useBootROM bool) Option {
	return func(c *console) {
		c.busOptions = append(c.busOptions, bus.WithBootROM(useBootROM))
		c.cpuOptions = append(c.cpuOptions, cpu.WithBootROM(useBootROM))
	}
}

func Run(ctx context.Context, romBytes []uint8, options ...Option) error {
	gbCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	gb := console{
		cancel:    cancel,
		cpu:       &cpu.CPU{},
		memory:    &memory.Memory{},
		cartridge: &cartridge.Cartridge{},
		bus:       &bus.Bus{},
		timer:     &timer.Timer{},
		ui:        &ui.UI{},
		ppu:       &ppu.PPU{},
		serial:    &serial.Serial{},
		dma:       &dma.DMA{},
	}

	for _, o := range options {
		o(&gb)
	}

	gb.bus.Cartridge = gb.cartridge
	gb.bus.CPU = gb.cpu
	gb.bus.DMA = gb.dma
	gb.bus.Memory = gb.memory
	gb.bus.PPU = gb.ppu
	gb.bus.Serial = gb.serial
	gb.bus.Timer = gb.timer
	gb.bus.UI = gb.ui
	gb.cpu.Bus = gb.bus
	gb.cpu.Console = &gb
	gb.dma.Bus = gb.bus
	gb.dma.PPU = gb.ppu
	gb.ppu.Bus = gb.bus
	gb.ppu.CPU = gb.cpu
	gb.serial.CPU = gb.cpu
	gb.timer.CPU = gb.cpu
	gb.ui.Bus = gb.bus
	gb.ui.CPU = gb.cpu
	gb.ui.Console = &gb
	gb.ui.PPU = gb.ppu

	err := gb.cartridge.Init(romBytes[0x147], romBytes[0x148])
	if err != nil {
		return fmt.Errorf("failed to init cartridge: %w", err)
	}

	gb.bus.Init(gb.busOptions...)

	err = gb.cpu.Init(gb.cpuOptions...)
	if err != nil {
		return fmt.Errorf("failed to init CPU: %w", err)
	}

	if !gb.headless {
		gb.ui.Init()
	}

	gb.timer.Init()
	gb.ppu.Init()
	gb.serial.Init(gb.serialOptions...)

	for i, b := range romBytes {
		gb.cartridge.Load(uint32(i), b)
	}

	uiCycles := 0

	for {
		cycles := 0

		if !gb.stopped && !gb.paused {
			cycles = gb.cpu.Step()
			gb.timer.Step(cycles)
		}

		if !gb.paused {
			gb.serial.Step(cycles)
			gb.dma.Step(cycles)
			gb.ppu.Step(cycles)
		}

		if gb.paused {
			cycles = 4
		}

		uiCycles += cycles
		if !gb.headless && uiCycles >= FRAME_CYCLES {
			gb.ui.Step()

			uiCycles = 0
		}

		// Check for context every frame cycle
		if uiCycles == 0 {
			select {
			case <-gbCtx.Done():
				return nil
			default:
			}
		}
	}
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

		fmt.Printf("%04X - %s\n", pc, opcode)

		pc += int(opcode.Bytes)
	}

	fmt.Print(sb.String())

	return nil
}

func (gb *console) Shutdown() {
	log.Debug("[console] shutdown")
	gb.cancel()
}

func (gb *console) Pause() {
	gb.paused = !gb.paused
	if gb.paused {
		log.Debug("[console] paused")
	} else {
		log.Debug("[console] unpaused")
	}
}

func (gb *console) Stop() {
	log.Debug("[console] stop")

	gb.stopped = true
}
