package memory

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"github.com/cterence/gbgo/internal/lib"
)

const (
	WRAM_START = 0xC000
	WRAM_END   = 0xDFFF
	WRAM_SIZE  = WRAM_END - WRAM_START + 1

	HRAM_START = 0xFF80
	HRAM_END   = 0xFFFE
	HRAM_SIZE  = HRAM_END - HRAM_START + 1
)

type Memory struct {
	state
}

type state struct {
	WRAM [WRAM_SIZE]uint8
	HRAM [HRAM_SIZE]uint8
}

func (m *Memory) Init() {
	m.WRAM = [WRAM_SIZE]uint8{}
	m.HRAM = [HRAM_SIZE]uint8{}
}

func (m *Memory) Read(addr uint16) uint8 {
	switch {
	case addr >= WRAM_START && addr <= WRAM_END:
		return m.WRAM[addr-WRAM_START]
	case addr >= HRAM_START && addr <= HRAM_END:
		return m.HRAM[addr-HRAM_START]
	default:
		panic(fmt.Errorf("unsupported memory read: %04x", addr))
	}
}

func (m *Memory) Write(addr uint16, value uint8) {
	switch {
	case addr >= WRAM_START && addr <= WRAM_END:
		m.WRAM[addr-WRAM_START] = value
	case addr >= HRAM_START && addr <= HRAM_END:
		m.HRAM[addr-HRAM_START] = value
	default:
		panic(fmt.Errorf("unsupported memory write: %04x", addr))
	}
}

func (m *Memory) Load(buf *bytes.Reader) {
	enc := gob.NewDecoder(buf)
	err := enc.Decode(&m.state)

	lib.Assert(err == nil, "failed to decode state: %v", err)
}

func (m *Memory) Save(buf *bytes.Buffer) {
	enc := gob.NewEncoder(buf)
	err := enc.Encode(m.state)

	lib.Assert(err == nil, "failed to encode state: %v", err)
}
