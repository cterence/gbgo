package ppu

import (
	"fmt"
)

const (
	OAM_CYCLES      = 80
	CYCLES_PER_LINE = 456
	LINES_PER_FRAME = 154

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
	TILE_BLOCK_1 uint16 = 0x9000
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

type object struct {
	y       uint8
	x       uint8
	tileIdx uint8
	attrs   uint8
}

type PPU struct {
	Bus bus
	CPU cpu

	cycles int

	frameBuffer [WIDTH][HEIGHT]uint8

	vram        [VRAM_SIZE]uint8
	oam         [OAM_SIZE]uint8
	objects     [10]object
	objectCount uint8

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

	p.setPPUMode(OAM_SCAN)
}

func (p *PPU) Read(addr uint16) uint8 {
	switch {
	case addr >= VRAM_START && addr <= VRAM_END:
		if p.getPPUMode() == DRAW {
			return 0xFF
		}

		return p.vram[addr-VRAM_START]
	case addr >= OAM_START && addr <= OAM_END:
		if p.dmaActive || (p.getPPUMode() == OAM_SCAN || p.getPPUMode() == DRAW) {
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
		// TODO: prevent writes when in DRAW mode (produces jumbled pixels now...)
		p.vram[addr-VRAM_START] = value
	case addr >= OAM_START && addr <= OAM_END:
		if !p.dmaActive && p.getPPUMode() != OAM_SCAN && p.getPPUMode() != DRAW {
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

func (p *PPU) GetFrameBuffer() [WIDTH][HEIGHT]uint8 {
	return p.frameBuffer
}

func (p *PPU) Step(cycles int) {
	p.cycles += cycles

	switch p.getPPUMode() {
	case OAM_SCAN:
		if p.cycles >= OAM_CYCLES {
			i := 0
			p.objectCount = 0

			for i < OAM_SIZE && p.objectCount < 10 {
				y := p.oam[i] - 16

				if p.ly >= y && p.ly <= p.getObjHeight()+y {
					p.objects[p.objectCount].y = p.oam[i]
					p.objects[p.objectCount].x = p.oam[i+1]
					p.objects[p.objectCount].tileIdx = p.oam[i+2]
					p.objects[p.objectCount].attrs = p.oam[i+3]

					p.objectCount++
				}

				i += 4
			}

			p.setPPUMode(DRAW)
		}

	case DRAW:
		if p.cycles >= 288 {
			y := p.ly

			for row := range WIDTH / 8 {
				tileX := uint8(row*8) + p.scx
				tileY := y + p.scy
				tilePixelRow := p.getBGTilePixelRow(tileX, tileY)

				for b := range 8 {
					x := row*8 + b
					p.frameBuffer[x][y] = tilePixelRow[b]
				}
			}

			p.setPPUMode(HBLANK)

			if p.stat&0x8 != 0 {
				p.CPU.RequestInterrupt(STAT_INTERRUPT_CODE)
			}
		}

	case HBLANK:
		if p.cycles >= CYCLES_PER_LINE {
			p.cycles = 0

			p.ly++
			if p.ly < HEIGHT {
				p.setPPUMode(OAM_SCAN)
			} else {
				p.setPPUMode(VBLANK)
				p.CPU.RequestInterrupt(VBLANK_INTERRUPT_CODE)

				if p.stat&0x10 != 0 {
					p.CPU.RequestInterrupt(STAT_INTERRUPT_CODE)
				}
			}
		}

	case VBLANK:
		if p.cycles >= CYCLES_PER_LINE {
			p.cycles = 0

			p.ly++
			if p.ly == LINES_PER_FRAME {
				p.setPPUMode(OAM_SCAN)

				p.ly = 0
				if p.stat&0x20 != 0 {
					p.CPU.RequestInterrupt(STAT_INTERRUPT_CODE)
				}
			}
		}
	}

	if p.ly == p.lyc {
		p.setLYEqLYC(1)

		if p.stat&0x40 != 0 {
			p.CPU.RequestInterrupt(STAT_INTERRUPT_CODE)
		}
	} else {
		p.setLYEqLYC(0)
	}
}

func (p *PPU) getPPUMode() mode {
	return mode(p.stat & 0x3)
}

func (p *PPU) setPPUMode(mode mode) {
	p.stat = (p.stat & 0xFC) | uint8(mode)
}

func (p *PPU) setLYEqLYC(value uint8) {
	p.stat = (p.stat & 0xFB) | value<<2
}

func (p *PPU) getBGTilePixelRow(x, y uint8) [8]uint8 {
	var (
		tileAddr     uint16
		tilePixelRow [8]uint8
	)

	tileY := y / 8
	tileRow := y % 8
	tileX := x / 8

	bgTileMapArea := bgTileMapAreas[p.lcdc>>3&1]
	tileMapIdx := bgTileMapArea + uint16(tileX) + uint16(tileY)*32
	tileIdx := p.vram[tileMapIdx-VRAM_START]

	if p.lcdc&0x10 != 0 {
		tileAddr = TILE_BLOCK_0 + uint16(tileIdx)*16
	} else {
		tileAddr = uint16(int32(TILE_BLOCK_1) + int32(int8(tileIdx))*16)
	}

	tileLo := p.vram[tileAddr+uint16(tileRow)*2-VRAM_START]
	tileHi := p.vram[tileAddr+uint16(tileRow)*2+1-VRAM_START]

	for b := range 8 {
		loPx := (tileLo >> (7 - b)) & 0x1
		hiPx := (tileHi >> (7 - b)) & 0x1
		colorIdx := hiPx<<1 | loPx
		tilePixelRow[b] = (p.bgp >> (colorIdx * 2)) & 0x3
	}

	return tilePixelRow
}

func (p *PPU) getObjHeight() uint8 {
	if p.lcdc&0x4 == 1 {
		return 16
	}

	return 8
}
