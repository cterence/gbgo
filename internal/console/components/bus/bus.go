package bus

import (
	_ "embed"
	"fmt"

	"github.com/cterence/gbgo/internal/log"
)

const (
	ROM_BANK_0_END = 0x3FFF
	ROM_BANK_1_END = 0x7FFF

	VRAM_START = 0x8000
	VRAM_END   = 0x9FFF

	EXTERNAL_RAM_START = 0xA000
	EXTERNAL_RAM_END   = 0xBFFF

	WRAM_START = 0xC000
	WRAM_END   = 0xDFFF
	WRAM_SIZE  = WRAM_END - WRAM_START + 1

	HRAM_START = 0xFF80
	HRAM_END   = 0xFFFE
	HRAM_SIZE  = HRAM_END - HRAM_START + 1

	DIV  = 0xFF04
	TIMA = 0xFF05
	TMA  = 0xFF06
	TAC  = 0xFF07
	IFF  = 0xFF0F
	IE   = 0xFFFF
)

//go:embed dmg.bin
var dmgBootRom []uint8

type rw interface {
	Read(addr uint16) uint8
	Write(addr uint16, value uint8)
}

type Bus struct {
	Memory    rw
	Cartridge rw
	CPU       rw
	Timer     rw
	PPU       rw
	Serial    rw
	DMA       rw

	bank uint8

	gbDoctor bool
}

type Option func(*Bus)

func WithGBDoctor(gbDoctor bool) Option {
	return func(b *Bus) {
		b.gbDoctor = gbDoctor
	}
}

func (b *Bus) Init(options ...Option) {
	for _, o := range options {
		o(b)
	}
}

func (b *Bus) Read(addr uint16) uint8 {
	switch {
	case addr <= 0xFF && b.bank == 0 && !b.gbDoctor:
		return dmgBootRom[addr]
	case addr <= ROM_BANK_1_END || (addr >= EXTERNAL_RAM_START && addr <= EXTERNAL_RAM_END):
		return b.Cartridge.Read(addr)
	case addr >= VRAM_START && addr <= VRAM_END || (addr >= 0xFF40 && addr <= 0xFF4B && addr != 0xFF46):
		return b.PPU.Read(addr)
	case addr == 0xFF46:
		return b.DMA.Read(addr)
	case addr == 0xFF01 || addr == 0xFF02:
		return b.Serial.Read(addr)
	case addr == 0xFF44 && b.gbDoctor:
		return 0x90
	// FIXME: needed for cpu_instrs to pass
	case addr == 0xFF4D:
		return 0xFF
	case addr == DIV || addr == TIMA || addr == TMA || addr == TAC:
		return b.Timer.Read(addr)
	case addr == IFF || addr == IE:
		return b.CPU.Read(addr)
	case addr >= WRAM_START && addr <= WRAM_END || addr >= HRAM_START && addr <= HRAM_END:
		return b.Memory.Read(addr)
	case addr >= 0xFF10 && addr <= 0xFF3F:
		// TODO: apu
		return 0
	default:
		panic(fmt.Errorf("unsupported bus read: %x", addr))
	}
}

func (b *Bus) Write(addr uint16, value uint8) {
	switch {
	case addr <= ROM_BANK_1_END || (addr >= EXTERNAL_RAM_START && addr <= EXTERNAL_RAM_END):
		b.Cartridge.Write(addr, value)
	case addr >= VRAM_START && addr <= VRAM_END || (addr >= 0xFF40 && addr <= 0xFF4B && addr != 0xFF46):
		b.PPU.Write(addr, value)
	case addr == 0xFF46:
		b.DMA.Write(addr, value)
	case addr == 0xFF01 || addr == 0xFF02:
		b.Serial.Write(addr, value)
	case addr == DIV || addr == TIMA || addr == TMA || addr == TAC:
		b.Timer.Write(addr, value)
	case addr == IFF || addr == IE:
		b.CPU.Write(addr, value)
	case addr == 0xFF46:
		log.Debug("DMA")
	case addr == 0xFF50:
		log.Debug("[bus] boot rom disabled\n")

		b.bank = value
	case addr >= WRAM_START && addr <= WRAM_END || addr >= HRAM_START && addr <= HRAM_END:
		b.Memory.Write(addr, value)
	case addr >= 0xFF10 && addr <= 0xFF3F:
		// TODO: apu
	default:
		panic(fmt.Errorf("unsupported bus write: %x", addr))
	}
}
