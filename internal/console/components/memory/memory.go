package memory

import "fmt"

const (
	WRAM_START = 0xC000
	WRAM_END   = 0xDFFF
	WRAM_SIZE  = WRAM_END - WRAM_START + 1

	HRAM_START = 0xFF80
	HRAM_END   = 0xFFFE
	HRAM_SIZE  = HRAM_END - HRAM_START + 1
)

type Memory struct {
	wram [WRAM_SIZE]uint8
	hram [HRAM_SIZE]uint8
}

func (m *Memory) Read(addr uint16) uint8 {
	switch {
	case addr >= WRAM_START && addr <= WRAM_END:
		return m.wram[addr-WRAM_START]
	case addr >= HRAM_START && addr <= HRAM_END:
		return m.hram[addr-HRAM_START]
	default:
		panic(fmt.Errorf("unsupported memory read: %04x", addr))
	}
}

func (m *Memory) Write(addr uint16, value uint8) {
	switch {
	case addr >= WRAM_START && addr <= WRAM_END:
		m.wram[addr-WRAM_START] = value
	case addr >= HRAM_START && addr <= HRAM_END:
		m.hram[addr-HRAM_START] = value
	default:
		panic(fmt.Errorf("unsupported memory write: %04x", addr))
	}
}
