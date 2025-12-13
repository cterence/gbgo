package cartridge

import (
	"fmt"
)

const (
	MAX_BANK_SIZE = 0x4000
)

type Cartridge struct {
	banksN      [][MAX_BANK_SIZE]uint8
	bank0       [MAX_BANK_SIZE]uint8
	currentBank uint8
}

func (c *Cartridge) Init(cartridgeSize int) error {
	bankCount := (cartridgeSize / MAX_BANK_SIZE) - 1

	if bankCount <= 0 || bankCount > 0x200 {
		return fmt.Errorf("unsupported bank count: %d", bankCount)
	}

	c.banksN = make([][MAX_BANK_SIZE]uint8, bankCount)

	return nil
}

func (c *Cartridge) Read(addr uint16) uint8 {
	if addr < MAX_BANK_SIZE {
		return c.bank0[addr]
	}

	if addr >= MAX_BANK_SIZE && addr < MAX_BANK_SIZE*2 {
		bankIndex := (addr / MAX_BANK_SIZE) - 1
		bankAddr := addr - MAX_BANK_SIZE*(bankIndex+1)

		return c.banksN[c.currentBank][bankAddr]
	}

	panic(fmt.Errorf("out of bounds cartridge read: %x", addr))
}

func (c *Cartridge) Write(addr uint16, value uint8) {
	// TODO: implement registers writes and handlers, ex: https://gbdev.io/pandocs/MBC1.html
}

func (c *Cartridge) Load(addr uint32, value uint8) {
	if addr < MAX_BANK_SIZE {
		c.bank0[addr] = value

		return
	}

	if addr >= MAX_BANK_SIZE {
		bankIndex := (addr / MAX_BANK_SIZE) - 1
		bankAddr := uint16(addr - MAX_BANK_SIZE*(bankIndex+1))
		c.banksN[bankIndex][bankAddr] = value

		return
	}

	panic(fmt.Errorf("out of bounds cartridge write: %x", addr))
}
