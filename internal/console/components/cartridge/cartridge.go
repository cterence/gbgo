package cartridge

import (
	"fmt"

	"github.com/cterence/gbgo/internal/log"
)

const (
	MAX_BANK_SIZE = 0x4000

	EXTERNAL_RAM_START = 0xA000
	EXTERNAL_RAM_END   = 0xBFFF
	EXTERNAL_RAM_SIZE  = EXTERNAL_RAM_END - EXTERNAL_RAM_START + 1
)

type Cartridge struct {
	banks       [][MAX_BANK_SIZE]uint8
	currentBank uint8
	ram         [EXTERNAL_RAM_SIZE]uint8
}

func (c *Cartridge) Init(cartridgeType, romSize uint8) error {
	c.currentBank = 1

	bankCount := 1 << (romSize + 1)

	if bankCount <= 0 || bankCount > 0x200 {
		return fmt.Errorf("unsupported bank count: %d", bankCount)
	}

	c.banks = make([][MAX_BANK_SIZE]uint8, bankCount)

	log.Debug("[cartridge] type: %d\n", cartridgeType)
	log.Debug("[cartridge] bank count: %d\n", bankCount)

	return nil
}

func (c *Cartridge) Read(addr uint16) uint8 {
	switch {
	case addr < MAX_BANK_SIZE:
		return c.banks[0][addr]
	case addr >= MAX_BANK_SIZE && addr < MAX_BANK_SIZE*2:
		bankIndex := (addr / MAX_BANK_SIZE) - 1
		bankAddr := addr - MAX_BANK_SIZE*(bankIndex+1)

		return c.banks[c.currentBank][bankAddr]
	case addr >= EXTERNAL_RAM_START && addr <= EXTERNAL_RAM_END:
		return c.ram[addr]

	default:
		panic(fmt.Errorf("out of bounds cartridge read: %x", addr))
	}
}

func (c *Cartridge) Write(addr uint16, value uint8) {
	switch {
	case addr >= 0x2000 && addr <= 0x3FFF:
		if value == 0 {
			value = 1
		}

		c.currentBank = value
		log.Debug("[cartridge] selected bank: %x\n", value)
	case addr >= EXTERNAL_RAM_START && addr <= EXTERNAL_RAM_END:
		c.ram[addr-EXTERNAL_RAM_START] = value
	}
}

func (c *Cartridge) Load(addr uint32, value uint8) {
	bankIndex := (addr / MAX_BANK_SIZE)
	bankAddr := uint16(addr - MAX_BANK_SIZE*bankIndex)
	c.banks[bankIndex][bankAddr] = value
}
