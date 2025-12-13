package memory

const (
	MEMORY_SIZE = 0x10000
)

type Memory struct {
	// TODO: real work RAM
	ram [MEMORY_SIZE]uint8
}

func (m *Memory) Read(addr uint16) uint8 {
	return m.ram[addr]
}

func (m *Memory) Write(addr uint16, value uint8) {
	m.ram[addr] = value
}
