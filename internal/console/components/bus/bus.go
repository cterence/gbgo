package bus

import "fmt"

type memory interface {
	Read(addr uint16) uint8
	Write(addr uint16, value uint8)
}

type cartridge interface {
	Read(addr uint16) uint8
}

type Bus struct {
	Memory    memory
	Cartridge cartridge
}

const (
	ROM_BANK_0_END = 0x3fff
	ROM_BANK_1_END = 0x7fff
)

func (b *Bus) Read(addr uint16) uint8 {
	if addr <= ROM_BANK_1_END {
		return b.Cartridge.Read(addr)
	}

	// FIXME: for gameboy doctor
	if addr == 0xFF44 {
		return 0x90
	}

	return b.Memory.Read(addr)
}

func (b *Bus) Write(addr uint16, value uint8) {
	if addr <= ROM_BANK_1_END {
		panic(fmt.Errorf("cannot write to cartridge addr: %x", addr))
	}

	b.Memory.Write(addr, value)
}
