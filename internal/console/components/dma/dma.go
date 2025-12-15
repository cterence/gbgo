package dma

import (
	"fmt"

	"github.com/cterence/gbgo/internal/log"
)

const (
	DMA_ADDR = 0xFF46

	DMA_BYTES = 0xA0
)

type bus interface {
	Read(addr uint16) uint8
}

type ppu interface {
	WriteOAM(addr uint16, value uint8)
	ToggleDMAActive(active bool)
}

type DMA struct {
	Bus bus
	PPU ppu

	dma       uint8
	dmaActive bool
	nextByte  uint8
}

func (d *DMA) Step(cycles int) {
	if !d.dmaActive {
		return
	}

	for range cycles / 4 {
		srcAddr := uint16(d.dma)<<8 | uint16(d.nextByte)
		destAddr := 0xFE00 | uint16(d.nextByte)

		d.PPU.WriteOAM(destAddr, d.Bus.Read(srcAddr))
		d.nextByte++

		if d.nextByte == DMA_BYTES {
			d.nextByte = 0
			d.PPU.ToggleDMAActive(false)
			d.dmaActive = false

			log.Debug("[dma] inactive")

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
		d.PPU.ToggleDMAActive(d.dmaActive)
		log.Debug("[dma] active")
	default:
		panic(fmt.Errorf("unsupported write for dma: %x", addr))
	}
}
