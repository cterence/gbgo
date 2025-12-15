package ppu

import (
	"encoding/binary"
	"fmt"
)

const (
	WIDTH       = 160
	HEIGHT      = 144
	PIXEL_BYTES = 4

	LCDC = 0xFF40
	STAT = 0xFF41
	SCY  = 0xFF42
	SCX  = 0xFF43
	LY   = 0xFF44
	LYC  = 0xFF45
	BGP  = 0xFF47

	VRAM_START = 0x8000
	VRAM_END   = 0x9FFF
	VRAM_SIZE  = VRAM_END - VRAM_START + 1

	OAM_START = 0xFE00
	OAM_END   = 0xFE9F
	OAM_SIZE  = OAM_END - OAM_START + 1

	TILE_MAP_SIZE = 0x400

	TILE_BLOCK_0 uint16 = 0x8000
	TILE_BLOCK_1 uint16 = 0x8800
)

type mode uint8

const (
	HBLANK mode = iota
	VBLANK
	OAM_SCAN
	DRAW
)

type bus interface {
	Read(addr uint16) uint8
}

type PPU struct {
	Bus bus

	vram        [VRAM_SIZE]uint8
	oam         [OAM_SIZE]uint8
	framebuffer [WIDTH * HEIGHT * 4]uint8

	cycles int

	lcdc uint8
	stat uint8
	scy  uint8
	scx  uint8
	ly   uint8
	lyc  uint8
	bgp  uint8

	mode      mode
	dmaActive bool
}

func (p *PPU) Init() {
	p.cycles = 0
	p.lcdc = 0
	p.stat = 0
	p.scy = 0
	p.scx = 0
	p.ly = 0x90 // TODO: set to 0 when ly increment implemented
	p.lyc = 0
	p.bgp = 0
}

func (p *PPU) Read(addr uint16) uint8 {
	switch {
	case addr >= VRAM_START && addr <= VRAM_END:
		if p.mode == DRAW {
			return 0xFF
		}

		return p.vram[addr-VRAM_START]
	case addr >= OAM_START && addr <= OAM_END:
		if p.dmaActive || (p.mode == OAM_SCAN || p.mode == DRAW) {
			return 0xFF
		}

		return p.oam[addr-OAM_START]
	default:
		switch addr {
		case LCDC:
			return p.lcdc
		case STAT:
			return p.stat
		case SCY:
			return p.scy
		case SCX:
			return p.scx
		case LY:
			return p.ly
		case LYC:
			return p.lyc
		case BGP:
			return p.bgp
		default:
			panic(fmt.Errorf("unsupported read for ppu: %x", addr))
		}
	}
}

func (p *PPU) Write(addr uint16, value uint8) {
	switch {
	case addr >= VRAM_START && addr <= VRAM_END:
		if p.mode != DRAW {
			p.vram[addr-VRAM_START] = value
		}
	case addr >= OAM_START && addr <= OAM_END:
		if !p.dmaActive && p.mode != OAM_SCAN && p.mode != DRAW {
			p.oam[addr-OAM_START] = value
		}
	default:
		switch addr {
		case LCDC:
			p.lcdc = value
		case STAT:
			p.stat = value
		case SCY:
			p.scy = value
		case SCX:
			p.scx = value
		case LY:
			p.ly = value
		case LYC:
			p.lyc = value
		case BGP:
			p.bgp = value
		default:
			panic(fmt.Errorf("unsupported write for ppu: %x", addr))
		}
	}
}

func (p *PPU) WriteOAM(addr uint16, value uint8) {
	p.oam[addr-OAM_START] = value
}

func (p *PPU) ToggleDMAActive(active bool) {
	p.dmaActive = active
}

var bgTileMapAreas = [2]uint16{0x9800, 0x9C00}

var palette = [4]uint32{
	0xFFFFFFFF,
	0xFFAAAAAA,
	0xFF555555,
	0xFF000000,
}

func (p *PPU) GetFramebuffer() [WIDTH * HEIGHT * 4]uint8 {
	return p.framebuffer
}

func (p *PPU) Step(cycles int) {
	bgTileMapArea := bgTileMapAreas[p.lcdc>>3&1]

	bgWindowArea := TILE_BLOCK_1
	if p.lcdc&1<<4 != 0 {
		bgWindowArea = TILE_BLOCK_0
	}

	for y := range HEIGHT {
		bgY := uint8(y) + p.scy
		tileY := bgY / 8

		for row := range WIDTH / 8 {
			tileAddr := p.vram[(bgTileMapArea+uint16(row)+uint16(tileY)*32)-VRAM_START]
			tileLo := p.vram[(bgWindowArea+uint16(tileAddr)*16+uint16(y%8*2))-VRAM_START]
			tileHi := p.vram[(bgWindowArea+uint16(tileAddr)*16+uint16(y%8*2+1))-VRAM_START]

			for b := range 8 {
				x := b + row*8
				bgX := uint8(x) + p.scx

				loPx := tileLo >> (7 - bgX%8) & 0x1
				hiPx := tileHi >> (7 - bgX%8) & 0x1
				pixel := hiPx<<1 | loPx
				offset := (x + y*WIDTH) * PIXEL_BYTES

				binary.LittleEndian.PutUint32(p.framebuffer[offset:offset+4], palette[pixel])
			}
		}
	}
}
