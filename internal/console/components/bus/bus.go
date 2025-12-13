package bus

import "fmt"

type cpu interface {
	ReadIFF() uint8
	WriteIFF(value uint8)
	ReadIE() uint8
	WriteIE(value uint8)
}

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
	CPU       cpu
}

const (
	ROM_BANK_0_END = 0x3FFF
	ROM_BANK_1_END = 0x7FFF

	IFF = 0xFF0F
	IE  = 0xFFFF
)

func (b *Bus) Read(addr uint16) uint8 {
	if addr <= ROM_BANK_1_END {
		return b.Cartridge.Read(addr)
	}

	// FIXME: for gameboy doctor
	if addr == 0xFF44 {
		return 0x90
	}

	if addr == IFF {
		return b.CPU.ReadIFF()
	}

	if addr == IE {
		return b.CPU.ReadIE()
	}

	return b.Memory.Read(addr)
}

func (b *Bus) Write(addr uint16, value uint8) {
	if addr <= ROM_BANK_1_END {
		panic(fmt.Errorf("cannot write to cartridge addr: %x", addr))
	}

	if addr == IFF {
		b.CPU.WriteIFF(value)

		return
	}

	if addr == IE {
		b.CPU.WriteIE(value)

		return
	}

	b.Memory.Write(addr, value)
}
