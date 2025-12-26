package console

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

type serializable interface {
	Load(*bytes.Reader)
	Save(*bytes.Buffer)
}

type state struct {
	Bytes [][]uint8
}

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

	romPath  string
	stateDir string

	cpuOptions    []cpu.Option
	busOptions    []bus.Option
	serialOptions []serial.Option

	headless bool
	stopped  bool
	paused   bool
	noState  bool
}

type Option func(*console)

func WithHeadless() Option {
	return func(c *console) {
		c.headless = true
	}
}

func WithPrintSerial() Option {
	return func(c *console) {
		c.serialOptions = append(c.serialOptions, serial.WithPrintSerial())
	}
}

func WithNoState() Option {
	return func(c *console) {
		c.noState = true
	}
}

func WithBootROM(bootRom []uint8) Option {
	return func(c *console) {
		c.busOptions = append(c.busOptions, bus.WithBootROM(bootRom))
		c.cpuOptions = append(c.cpuOptions, cpu.WithBootROM())
	}
}

func Run(ctx context.Context, romBytes []uint8, romPath, stateDir string, options ...Option) error {
	gbCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	gb := console{
		cancel:    cancel,
		romPath:   romPath,
		stateDir:  stateDir,
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

	err := gb.cartridge.Init(romPath, stateDir, romBytes[0x147], romBytes[0x148], romBytes[0x149])
	if err != nil {
		return fmt.Errorf("failed to init cartridge: %w", err)
	}

	gb.Reset()

	for i, b := range romBytes {
		gb.cartridge.Load(uint32(i), b)
	}

	if !gb.noState {
		gb.loadState()
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

			for range cycles {
				gb.ppu.Step(1)
			}
		}

		if gb.paused {
			cycles = 4
		}

		uiCycles += cycles
		if !gb.headless && uiCycles >= FRAME_CYCLES {
			gb.ui.Step(cycles)

			uiCycles -= FRAME_CYCLES
		}

		// Check context every frame cycle
		if uiCycles == 0 {
			select {
			case <-gbCtx.Done():
				return nil
			default:
			}
		}
	}
}

func (gb *console) Reset() {
	gb.cpu.Init(gb.cpuOptions...)
	gb.memory.Init()
	gb.bus.Init(gb.busOptions...)
	gb.timer.Init()
	gb.ppu.Init()
	gb.serial.Init(gb.serialOptions...)

	if !gb.headless {
		gb.ui.Init(gb.romPath)
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
	gb.cancel()
	gb.cartridge.Close()

	if !gb.noState {
		gb.saveState()
	}

	log.Debug("[console] shutdown")
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

func (gb *console) getSerializables() []serializable {
	return []serializable{gb.cpu, gb.memory, gb.ppu, gb.timer}
}

func (gb *console) loadState() {
	saveStatePath := strings.ReplaceAll(filepath.Base(gb.romPath), filepath.Ext(gb.romPath), ".state")

	stateBytes, err := os.ReadFile(filepath.Join(gb.stateDir, saveStatePath))
	if err != nil {
		if !strings.Contains(err.Error(), "no such file or directory") {
			fmt.Printf("failed to open save state file: %v\n", err)
		}

		return
	}

	var st state

	r := bytes.NewReader(stateBytes)
	dec := gob.NewDecoder(r)

	err = dec.Decode(&st)
	if err != nil {
		fmt.Printf("failed to decode save state file: %v\n", err)
		return
	}

	ser := gb.getSerializables()

	for i, s := range st.Bytes {
		ser[i].Load(bytes.NewReader(s))
	}

	log.Debug("[console] loaded state from %s", saveStatePath)
}

func (gb *console) saveState() {
	var st state

	for _, s := range gb.getSerializables() {
		buf := bytes.NewBuffer(nil)
		s.Save(buf)

		state, err := io.ReadAll(buf)
		if err != nil {
			fmt.Printf("failed to read save state buffer: %v\n", err)
		}

		st.Bytes = append(st.Bytes, state)
	}

	saveStatePath := strings.ReplaceAll(filepath.Base(gb.romPath), filepath.Ext(gb.romPath), ".state")

	f, err := os.Create(filepath.Join(gb.stateDir, saveStatePath))
	if err != nil {
		fmt.Printf("failed to create save state file: %v\n", err)

		return
	}

	defer func() {
		if err := f.Close(); err != nil {
			fmt.Printf("failed to close state file: %v\n", err)
		}
	}()

	enc := gob.NewEncoder(f)

	err = enc.Encode(st)
	if err != nil {
		fmt.Printf("failed to encode save state: %v\n", err)
	}

	log.Debug("[console] saved state to %s", saveStatePath)
}
