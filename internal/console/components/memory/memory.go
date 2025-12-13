package memory

import "fmt"

const (
	VRAM_START = 0x8000
	VRAM_END   = 0x9FFF
	VRAM_SIZE  = VRAM_END - VRAM_START + 1

	WRAM_START = 0xC000
	WRAM_END   = 0xDFFF
	WRAM_SIZE  = WRAM_END - WRAM_START + 1

	OAM_START = 0xFE00
	OAM_END   = 0xFE9F
	OAM_SIZE  = OAM_END - OAM_START + 1

	IO_START = 0xFF00
	IO_END   = 0xFF7F
	IO_SIZE  = IO_END - IO_START + 1

	HRAM_START = 0xFF80
	HRAM_END   = 0xFFFE
	HRAM_SIZE  = HRAM_END - HRAM_START + 1
)

type Memory struct {
	vram  [VRAM_SIZE]uint8
	wram  [WRAM_SIZE]uint8
	oam   [OAM_SIZE]uint8
	ioTmp [IO_SIZE]uint8
	hram  [HRAM_SIZE]uint8
}

func (m *Memory) Read(addr uint16) uint8 {
	if addr >= VRAM_START && addr <= VRAM_END {
		return m.vram[addr-VRAM_START]
	}

	if addr >= WRAM_START && addr <= WRAM_END {
		return m.wram[addr-WRAM_START]
	}

	if addr >= OAM_START && addr <= OAM_END {
		return m.oam[addr-OAM_START]
	}

	if addr >= IO_START && addr <= IO_END {
		return m.ioTmp[addr-IO_START]
	}

	if addr >= HRAM_START && addr <= HRAM_END {
		return m.hram[addr-HRAM_START]
	}

	panic(fmt.Errorf("unsupported memory read: %04x", addr))
}

func (m *Memory) Write(addr uint16, value uint8) {
	if addr >= VRAM_START && addr <= VRAM_END {
		m.vram[addr-VRAM_START] = value
		return
	}

	if addr >= WRAM_START && addr <= WRAM_END {
		m.wram[addr-WRAM_START] = value
		return
	}

	if addr >= OAM_START && addr <= OAM_END {
		m.oam[addr-OAM_START] = value
		return
	}

	if addr >= IO_START && addr <= IO_END {
		m.ioTmp[addr-IO_START] = value
		return
	}

	if addr >= HRAM_START && addr <= HRAM_END {
		m.hram[addr-HRAM_START] = value
		return
	}

	panic(fmt.Errorf("unsupported memory write: %04x", addr))
}
