package ppu

import (
	"encoding/binary"
	"fmt"
)

const (
	VBLANK_INTERRUPT_CODE = 0x1
	STAT_INTERRUPT_CODE   = 0x2

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
	OBP0 = 0xFF48
	OBP1 = 0xFF49
	WY   = 0xFF4A
	WX   = 0xFF4B

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

type cpu interface {
	RequestInterrupt(code uint8)
}

type PPU struct {
	Bus bus
	CPU cpu

	cycles int

	frameBuffer [WIDTH * HEIGHT * 4]uint8

	vram    [VRAM_SIZE]uint8
	oam     [OAM_SIZE]uint8
	objects [10]uint8

	lcdc uint8
	stat uint8
	scy  uint8
	scx  uint8
	ly   uint8
	lyc  uint8
	bgp  uint8
	obp0 uint8
	obp1 uint8
	wy   uint8
	wx   uint8

	dmaActive bool
}

func (p *PPU) Init() {
	p.cycles = 0
	p.lcdc = 0
	p.stat = 0
	p.scy = 0
	p.scx = 0
	p.ly = 0
	p.lyc = 0
	p.bgp = 0
	p.obp0 = 0
	p.obp1 = 0
	p.wy = 0
	p.wx = 0

	p.setSTATMode(OAM_SCAN)
}

func (p *PPU) Read(addr uint16) uint8 {
	switch {
	case addr >= VRAM_START && addr <= VRAM_END:
		if p.getSTATMode() == DRAW {
			return 0xFF
		}

		return p.vram[addr-VRAM_START]
	case addr >= OAM_START && addr <= OAM_END:
		if p.dmaActive || (p.getSTATMode() == OAM_SCAN || p.getSTATMode() == DRAW) {
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
		case OBP0:
			return p.obp0
		case OBP1:
			return p.obp1
		case WY:
			return p.wy
		case WX:
			return p.wx
		default:
			panic(fmt.Errorf("unsupported read for ppu: %x", addr))
		}
	}
}

func (p *PPU) Write(addr uint16, value uint8) {
	switch {
	case addr >= VRAM_START && addr <= VRAM_END:
		if p.getSTATMode() != DRAW {
			p.vram[addr-VRAM_START] = value
		}
	case addr >= OAM_START && addr <= OAM_END:
		if !p.dmaActive && p.getSTATMode() != OAM_SCAN && p.getSTATMode() != DRAW {
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
		case OBP0:
			p.obp0 = value
		case OBP1:
			p.obp1 = value
		case WY:
			p.wy = value
		case WX:
			p.wx = value
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
	return p.frameBuffer
}

func (p *PPU) Step(cycles int) {
	for range cycles / 4 {
		switch p.getSTATMode() {
		case OAM_SCAN:
			if p.cycles == 0 {
				i, objCount := 0, 0

				for i < OAM_SIZE && objCount < 10 {
					y := p.oam[i] - 16

					if p.ly >= y && p.ly <= p.objSize()+y {
						p.objects[objCount] = uint8(i)
						objCount++
					}

					i += 4
				}
			}

			if p.cycles == 80 {
				p.setSTATMode(DRAW)
			} else {
				p.cycles += 4
			}

		case DRAW:
			if p.cycles == 80 {
				bgTileMapArea := bgTileMapAreas[p.lcdc>>3&1]

				bgWindowArea := TILE_BLOCK_1
				if p.lcdc&1<<4 != 0 {
					bgWindowArea = TILE_BLOCK_0
				}

				p.setBGPixels(bgTileMapArea, bgWindowArea)

				for i := range p.objects {
					p.objects[i] = 0
				}
			}

			if p.cycles == 288 {
				p.setSTATMode(HBLANK)

				if p.stat&0x8 != 0 {
					p.CPU.RequestInterrupt(STAT_INTERRUPT_CODE)
				}
			} else {
				p.cycles += 4
			}

		case HBLANK:
			if p.cycles == 456 {
				p.cycles = 0

				p.ly++
				if p.ly < 144 {
					p.setSTATMode(OAM_SCAN)
				} else {
					p.setSTATMode(VBLANK)
					p.CPU.RequestInterrupt(VBLANK_INTERRUPT_CODE)

					if p.stat&0x10 != 0 {
						p.CPU.RequestInterrupt(STAT_INTERRUPT_CODE)
					}
				}
			} else {
				p.cycles += 4
			}

		case VBLANK:
			if p.cycles == 456 {
				p.cycles = 0

				p.ly++
				if p.ly == 154 {
					p.setSTATMode(OAM_SCAN)

					p.ly = 0
					if p.stat&0x20 != 0 {
						p.CPU.RequestInterrupt(STAT_INTERRUPT_CODE)
					}
				}
			} else {
				p.cycles += 4
			}
		}

		if p.ly == p.lyc {
			if p.stat&0x40 != 0 {
				p.setSTATLYC(1)
				p.CPU.RequestInterrupt(STAT_INTERRUPT_CODE)
			}
		} else {
			p.setSTATLYC(0)
		}
	}
}

func (p *PPU) setBGPixels(bgTileMapArea, bgWindowArea uint16) {
	bgY := p.ly + p.scy
	tileY := uint16(bgY / 8)
	fineY := uint16(bgY % 8)

	for row := range WIDTH / 8 {
		tileIdx := p.vram[(bgTileMapArea+uint16(row)+tileY*32)-VRAM_START]

		tileDataAddr := bgWindowArea + uint16(tileIdx)*16 + fineY*2
		if bgWindowArea == TILE_BLOCK_1 {
			tileDataAddr = 0x9000 + uint16(int16(int8(tileIdx))*16) + fineY*2
		}

		tileLo := p.vram[tileDataAddr-VRAM_START]
		tileHi := p.vram[tileDataAddr+1-VRAM_START]

		for b := range 8 {
			x := b + row*8

			loPx := (tileLo >> (7 - b)) & 0x1 // Use b, not bgX%8
			hiPx := (tileHi >> (7 - b)) & 0x1
			pixel := hiPx<<1 | loPx

			offset := (x + int(p.ly)*WIDTH) * PIXEL_BYTES
			binary.LittleEndian.PutUint32(p.frameBuffer[offset:offset+4], palette[pixel])
		}
	}
}

func (p *PPU) getSTATMode() mode {
	return mode(p.stat & 0x3)
}

func (p *PPU) setSTATMode(mode mode) {
	p.stat = (p.stat & 0xFC) | uint8(mode)
}

func (p *PPU) setSTATLYC(value uint8) {
	p.stat = (p.stat & 0xFB) | value&1
}

func (p *PPU) objSize() uint8 {
	if p.lcdc&0x4 == 1 {
		return 16
	}

	return 8
}
