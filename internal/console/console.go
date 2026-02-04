package console

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/cterence/gbgo/internal/console/components/bus"
	"github.com/cterence/gbgo/internal/console/components/cartridge"
	"github.com/cterence/gbgo/internal/console/components/cpu"
	"github.com/cterence/gbgo/internal/console/components/debugger"
	"github.com/cterence/gbgo/internal/console/components/dma"
	"github.com/cterence/gbgo/internal/console/components/joypad"
	"github.com/cterence/gbgo/internal/console/components/memory"
	"github.com/cterence/gbgo/internal/console/components/ppu"
	"github.com/cterence/gbgo/internal/console/components/serial"
	"github.com/cterence/gbgo/internal/console/components/timer"
	"github.com/cterence/gbgo/internal/console/components/ui"
	"github.com/cterence/gbgo/internal/log"
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
	joypad    *joypad.Joypad
	ui        *ui.UI
	ppu       *ppu.PPU
	serial    *serial.Serial
	dma       *dma.DMA
	debugger  *debugger.Debugger

	romPath  string
	stateDir string

	cpuOptions    []cpu.Option
	busOptions    []bus.Option
	serialOptions []serial.Option

	headless    bool
	stopped     bool
	paused      bool
	noState     bool
	shouldClose bool
	debug       bool
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

func WithDebug() Option {
	return func(c *console) {
		c.debug = true
		c.cpuOptions = append(c.cpuOptions, cpu.WithDebug())
	}
}

func WithBootROM(bootRom []uint8) Option {
	return func(c *console) {
		c.busOptions = append(c.busOptions, bus.WithBootROM(bootRom))
		c.cpuOptions = append(c.cpuOptions, cpu.WithBootROM())
	}
}

func Run(romBytes []uint8, romPath, stateDir string, options ...Option) error {
	gb := console{
		romPath:   romPath,
		stateDir:  stateDir,
		cpu:       &cpu.CPU{},
		memory:    &memory.Memory{},
		cartridge: &cartridge.Cartridge{},
		bus:       &bus.Bus{},
		timer:     &timer.Timer{},
		joypad:    &joypad.Joypad{},
		ui:        &ui.UI{},
		ppu:       &ppu.PPU{},
		serial:    &serial.Serial{},
		dma:       &dma.DMA{},
		debugger:  &debugger.Debugger{},
	}

	for _, o := range options {
		o(&gb)
	}

	err := gb.cartridge.Init(romPath, stateDir, romBytes[0x147], romBytes[0x148], romBytes[0x149])
	if err != nil {
		return fmt.Errorf("failed to init cartridge: %w", err)
	}
	defer gb.cartridge.Close()

	gb.Reset()

	if !gb.headless {
		defer gb.ui.Close()
	}

	for i, b := range romBytes {
		gb.cartridge.Load(uint32(i), b)
	}

	if !gb.noState {
		gb.loadState()
		defer gb.saveState()
	}

	totalCycles := uint64(0)

	for !gb.shouldClose {
		cycles := 4

		if !gb.paused {
			if !gb.stopped {
				cycles = gb.cpu.Step()
				gb.timer.Step(cycles)
			}

			gb.serial.Step(cycles)
			gb.dma.Step(cycles)

			for range cycles / 2 {
				gb.ppu.Step(2)
			}
		}

		if !gb.headless && (gb.ppu.IsFrameReady() || gb.paused) {
			gb.ui.HandleEvents()
			gb.ui.DrawFrame()
		}

		totalCycles += uint64(cycles)
	}

	return nil
}

func (gb *console) Reset() {
	gb.cpu.Init(gb.bus, gb, gb.debugger, gb.cpuOptions...)
	gb.memory.Init()
	gb.bus.Init(gb.memory, gb.cartridge, gb.cpu, gb.timer, gb.ppu, gb.serial, gb.dma, gb.joypad, gb.busOptions...)
	gb.timer.Init(gb.cpu)
	gb.ppu.Init(gb.bus, gb.cpu)
	gb.serial.Init(gb.cpu, gb.serialOptions...)
	gb.dma.Init(gb.bus, gb.ppu)
	gb.joypad.Init(gb.cpu)

	if gb.debug {
		gb.debugger.Init(os.Stdout)
	}

	if !gb.headless {
		gb.ui.Init(gb, gb.joypad, gb.ppu, gb.romPath)
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
	gb.shouldClose = true
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
