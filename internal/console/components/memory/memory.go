package memory

import (
	"fmt"

	"github.com/cterence/gbgo/internal/console/lib"
)

const (
	MEMORY_SIZE = 0x10000
)

type Memory struct {
	ram [MEMORY_SIZE]uint8
}

func (m *Memory) Read(addr uint16) uint8 {
	lib.Assert(int(addr) < MEMORY_SIZE, fmt.Errorf("out of bounds read: %x", addr))

	return m.ram[addr]
}

func (m *Memory) Write(addr uint16, value uint8) {
	lib.Assert(int(addr) < MEMORY_SIZE, fmt.Errorf("out of bounds write: %x", addr))
	m.ram[addr] = value
}
