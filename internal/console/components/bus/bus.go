package bus

import "github.com/cterence/gbgo/internal/log"

const (
	ROM_BANK_0_END = 0x3FFF
	ROM_BANK_1_END = 0x7FFF

	VRAM_START = 0x8000
	VRAM_END   = 0x9FFF

	DIV  = 0xFF04
	TIMA = 0xFF05
	TMA  = 0xFF06
	TAC  = 0xFF07
	IFF  = 0xFF0F
	IE   = 0xFFFF
)

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
	if addr <= ROM_BANK_1_END {
		return b.Cartridge.Read(addr)
	}

	if addr >= VRAM_START && addr <= VRAM_END {
		return b.PPU.Read(addr)
	}

	if addr == 0xFF44 && b.gbDoctor {
		return 0x90
	}

	if addr == 0xFF4D {
		return 0xFF
	}

	if addr == DIV || addr == TIMA || addr == TMA || addr == TAC {
		return b.Timer.Read(addr)
	}

	if addr == IFF || addr == IE {
		return b.CPU.Read(addr)
	}

	return b.Memory.Read(addr)
}

func (b *Bus) Write(addr uint16, value uint8) {
	if addr <= ROM_BANK_1_END {
		b.Cartridge.Write(addr, value)
		return
	}

	if addr >= VRAM_START && addr <= VRAM_END {
		b.PPU.Write(addr, value)
		return
	}

	if addr == DIV || addr == TIMA || addr == TMA || addr == TAC {
		b.Timer.Write(addr, value)
		return
	}

	if addr == IFF || addr == IE {
		b.CPU.Write(addr, value)
		return
	}

	if addr == 0xFF46 {
		log.Debug("DMA")
	}

	b.Memory.Write(addr, value)
}
