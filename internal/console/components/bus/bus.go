package bus

import (
	_ "embed"
)

const (
	ROM_BANK_0_END = 0x3FFF
	ROM_BANK_1_END = 0x7FFF

	VRAM_START = 0x8000
	VRAM_END   = 0x9FFF

	OAM_START = 0xFE00
	OAM_END   = 0xFE9F

	EXTERNAL_RAM_START = 0xA000
	EXTERNAL_RAM_END   = 0xBFFF

	WRAM_START = 0xC000
	WRAM_END   = 0xDFFF

	ECHO_START = 0xE000
	ECHO_END   = 0xFDFF

	HRAM_START = 0xFF80
	HRAM_END   = 0xFFFE

	UNUSED_START = 0xFEA0
	UNUSED_END   = 0xFEFF

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
	UI        rw

	bank uint8
}

type Option func(*Bus)

func WithBootROM(useBootROM bool) Option {
	return func(b *Bus) {
		if !useBootROM {
			// Simulate boot
			b.bank = 1
			b.Write(0xFF00, 0xCF)
			b.Write(0xFF01, 0x00)
			b.Write(0xFF02, 0x7E)
			b.Write(0xFF04, 0xAB)
			b.Write(0xFF05, 0x00)
			b.Write(0xFF06, 0x00)
			b.Write(0xFF07, 0xF8)
			b.Write(0xFF0F, 0xE1)
			b.Write(0xFF10, 0x80)
			b.Write(0xFF11, 0xBF)
			b.Write(0xFF12, 0xF3)
			b.Write(0xFF14, 0xBF)
			b.Write(0xFF16, 0x3F)
			b.Write(0xFF17, 0x00)
			b.Write(0xFF18, 0xFF)
			b.Write(0xFF19, 0xBF)
			b.Write(0xFF1A, 0x7F)
			b.Write(0xFF1B, 0xFF)
			b.Write(0xFF1C, 0x9F)
			b.Write(0xFF1D, 0xBF)
			b.Write(0xFF1E, 0xBF)
			b.Write(0xFF20, 0xFF)
			b.Write(0xFF21, 0x00)
			b.Write(0xFF22, 0x00)
			b.Write(0xFF23, 0xBF)
			b.Write(0xFF24, 0x77)
			b.Write(0xFF25, 0xF3)
			b.Write(0xFF26, 0xF1)
			b.Write(0xFF40, 0x91)
			b.Write(0xFF41, 0x85)
			b.Write(0xFF42, 0x00)
			b.Write(0xFF43, 0x00)
			b.Write(0xFF45, 0x00)
			b.Write(0xFF47, 0xFC)
			b.Write(0xFF46, 0xFF)
			b.Write(0xFF48, 0xFF)
			b.Write(0xFF49, 0xFF)
			b.Write(0xFF4A, 0x0)
			b.Write(0xFF4B, 0x0)
			b.Write(0xFFFF, 0x0)
		}
	}
}

func (b *Bus) Init(options ...Option) {
	for _, o := range options {
		o(b)
	}
}

func (b *Bus) Read(addr uint16) uint8 {
	switch {
	case addr <= 0xFF && b.bank == 0:
		return dmgBootRom[addr]
	case addr <= ROM_BANK_1_END || (addr >= EXTERNAL_RAM_START && addr <= EXTERNAL_RAM_END):
		return b.Cartridge.Read(addr)
	case (addr >= VRAM_START && addr <= VRAM_END) || (addr >= OAM_START && addr <= OAM_END) || (addr >= 0xFF40 && addr <= 0xFF4B && addr != 0xFF46):
		return b.PPU.Read(addr)
	case addr == 0xFF46:
		return b.DMA.Read(addr)
	case addr == 0xFF01 || addr == 0xFF02:
		return b.Serial.Read(addr)
	case addr == DIV || addr == TIMA || addr == TMA || addr == TAC:
		return b.Timer.Read(addr)
	case addr == IFF || addr == IE:
		return b.CPU.Read(addr)
	case addr >= WRAM_START && addr <= WRAM_END || addr >= HRAM_START && addr <= HRAM_END:
		return b.Memory.Read(addr)
	case addr >= ECHO_START && addr <= ECHO_END:
		return b.Memory.Read(addr - ECHO_START + WRAM_START)
	case addr >= UNUSED_START && addr <= UNUSED_END:
		return 0xFF
	case addr == 0xFF00:
		return b.UI.Read(addr)
	// TODO: apu
	case addr >= 0xFF10 && addr <= 0xFF3F:
		return 0xFF
	default:
		return 0xFF
	}
}

func (b *Bus) Write(addr uint16, value uint8) {
	switch {
	case addr <= ROM_BANK_1_END || (addr >= EXTERNAL_RAM_START && addr <= EXTERNAL_RAM_END):
		b.Cartridge.Write(addr, value)
	case addr >= VRAM_START && addr <= VRAM_END || (addr >= OAM_START && addr <= OAM_END) || (addr >= 0xFF40 && addr <= 0xFF4B && addr != 0xFF46):
		b.PPU.Write(addr, value)
	case addr == 0xFF46:
		b.DMA.Write(addr, value)
	case addr == 0xFF01 || addr == 0xFF02:
		b.Serial.Write(addr, value)
	case addr == DIV || addr == TIMA || addr == TMA || addr == TAC:
		b.Timer.Write(addr, value)
	case addr == IFF || addr == IE:
		b.CPU.Write(addr, value)
	case addr == 0xFF50:
		b.bank = value
	case addr >= WRAM_START && addr <= WRAM_END || addr >= HRAM_START && addr <= HRAM_END:
		b.Memory.Write(addr, value)
	case addr >= ECHO_START && addr <= ECHO_END:
		b.Memory.Write(addr-ECHO_START+WRAM_START, value)
	case addr >= UNUSED_START && addr <= UNUSED_END:
	case addr == 0xFF00:
		b.UI.Write(addr, value)
	// TODO: apu
	case addr >= 0xFF10 && addr <= 0xFF3F:
	default:
	}
}
