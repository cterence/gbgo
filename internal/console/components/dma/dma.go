package dma

import (
	"fmt"
)

const (
	DMA_ADDR  = 0xFF46
	DMA_BYTES = 0xA0
)

type Bus interface {
	Read(addr uint16) uint8
}

type PPU interface {
	WriteOAM(addr uint16, value uint8)
	ToggleDMAActive(active bool)
}

type DMA struct {
	bus Bus
	ppu PPU

	dma       uint8
	dmaActive bool
	nextByte  uint8
}

func (d *DMA) Init(bus Bus, ppu PPU) {
	d.bus = bus
	d.ppu = ppu
}

func (d *DMA) Step(cycles int) {
	if !d.dmaActive {
		return
	}

	for range cycles / 4 {
		srcAddr := uint16(d.dma)<<8 | uint16(d.nextByte)
		destAddr := 0xFE00 | uint16(d.nextByte)

		d.ppu.WriteOAM(destAddr, d.bus.Read(srcAddr))
		d.nextByte++

		if d.nextByte == DMA_BYTES {
			d.nextByte = 0
			d.ppu.ToggleDMAActive(false)
			d.dmaActive = false

			return
		}
	}
}

func (d *DMA) Read(addr uint16) uint8 {
	switch addr {
	case DMA_ADDR:
		return d.dma
	default:
		panic(fmt.Errorf("unsupported read for dma: %x", addr))
	}
}

func (d *DMA) Write(addr uint16, value uint8) {
	switch addr {
	case DMA_ADDR:
		d.dma = value
		d.dmaActive = true
		d.ppu.ToggleDMAActive(d.dmaActive)
	default:
		panic(fmt.Errorf("unsupported write for dma: %x", addr))
	}
}
