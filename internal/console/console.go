package console

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/Zyko0/go-sdl3/bin/binsdl"
	"github.com/cterence/gbgo/internal/console/components/bus"
	"github.com/cterence/gbgo/internal/console/components/cartridge"
	"github.com/cterence/gbgo/internal/console/components/cpu"
	"github.com/cterence/gbgo/internal/console/components/memory"
	"github.com/cterence/gbgo/internal/console/components/ppu"
	"github.com/cterence/gbgo/internal/console/components/serial"
	"github.com/cterence/gbgo/internal/console/components/timer"
	"github.com/cterence/gbgo/internal/console/components/ui"
	"github.com/cterence/gbgo/internal/log"
)

const (
	CPU_FREQ = 4194304
	UI_FREQ  = 60
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

	cancel context.CancelFunc

	cpuOptions    []cpu.Option
	busOptions    []bus.Option
	serialOptions []serial.Option

	stopCPUAfter int
	headless     bool
	stopped      bool
}

type Option func(*console)

func WithStopCPUAfter(stopCPUAfter int) Option {
	return func(c *console) {
		c.stopCPUAfter = stopCPUAfter
	}
}

func WithGBDoctor(gbDoctor bool) Option {
	return func(c *console) {
		c.cpuOptions = append(c.cpuOptions, cpu.WithGBDoctor(gbDoctor))
		c.busOptions = append(c.busOptions, bus.WithGBDoctor(gbDoctor))
	}
}

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
	}

	for _, o := range options {
		o(&gb)
	}

	gb.bus.Cartridge = gb.cartridge
	gb.bus.CPU = gb.cpu
	gb.bus.Memory = gb.memory
	gb.bus.PPU = gb.ppu
	gb.bus.Serial = gb.serial
	gb.bus.Timer = gb.timer
	gb.cpu.Bus = gb.bus
	gb.cpu.Console = &gb
	gb.ppu.Bus = gb.bus
	gb.serial.CPU = gb.cpu
	gb.timer.CPU = gb.cpu
	gb.ui.Bus = gb.bus
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
		defer binsdl.Load().Unload()
		defer gb.ui.Close()

		if err := gb.ui.Init(); err != nil {
			return fmt.Errorf("failed to init UI: %w", err)
		}

		trapSigInt(cancel)
	}

	gb.timer.Init()
	gb.ppu.Init()
	gb.serial.Init(gb.serialOptions...)

	for i, b := range romBytes {
		gb.cartridge.Load(uint32(i), b)
	}

	cycles, totalCycles, uiCycles := 0, 0, 0

	for err == nil {
		select {
		default:
			if !gb.stopped {
				cycles, err = gb.cpu.Step()
				gb.timer.Step(cycles)
			}

			uiCycles += cycles
			if !gb.headless && uiCycles >= CPU_FREQ/UI_FREQ {
				gb.ppu.Step()
				gb.ui.Step()

				uiCycles -= CPU_FREQ / UI_FREQ
			}

			gb.serial.Step()

			totalCycles += cycles
			if gb.stopCPUAfter > 0 && totalCycles > gb.stopCPUAfter {
				err = fmt.Errorf("stopping CPU after %d cycles", gb.stopCPUAfter)
			}
		case <-gbCtx.Done():
			return nil
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

func (gb *console) Shutdown() {
	log.Debug("[console] shutdown")
	gb.cancel()
}

func (gb *console) Stop() {
	log.Debug("[console] stop")

	gb.stopped = true
}

func trapSigInt(cancel context.CancelFunc) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		cancel()
	}()
}
