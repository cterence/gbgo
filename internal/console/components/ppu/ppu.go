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

	WIDTH  = 160
	HEIGHT = 144

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

type ppuMode uint8

const (
	HBLANK ppuMode = iota
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

	// Attributes
	priority   bool
	yFlip      bool
	xFlip      bool
	dmgPalette bool
	cgbBank    bool
	cgbPalette uint8
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

	// LDCD
	ppuEnabled    bool
	windowTileMap bool
	windowEnabled bool
	bgwTileData   bool
	bgTileMap     bool
	objSize       bool
	objEnabled    bool
	bgwEnabled    bool

	// STAT
	lycInt    bool
	oamInt    bool
	vblankInt bool
	hblankInt bool
	lycEqLy   bool
	ppuMode   ppuMode

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
	p.scy = 0
	p.scx = 0
	p.ly = 0
	p.lyc = 0
	p.bgp = 0
	p.obp0 = 0
	p.obp1 = 0
	p.wy = 0
	p.wx = 0

	p.ppuMode = OAM_SCAN
}

func (p *PPU) Read(addr uint16) uint8 {
	switch {
	case addr >= VRAM_START && addr <= VRAM_END:
		if p.ppuMode == DRAW {
			return 0xFF
		}

		return p.vram[addr-VRAM_START]
	case addr >= OAM_START && addr <= OAM_END:
		if p.dmaActive || (p.ppuMode == OAM_SCAN || p.ppuMode == DRAW) {
			return 0xFF
		}

		return p.oam[addr-OAM_START]
	default:
		switch addr {
		case LCDC:
			return p.readLCDC()
		case STAT:
			return p.readSTAT()
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
		if !p.dmaActive && p.ppuMode != OAM_SCAN && p.ppuMode != DRAW {
			p.oam[addr-OAM_START] = value
		}
	default:
		switch addr {
		case LCDC:
			p.setLCDC(value)
		case STAT:
			p.setSTAT(value)
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

var tileMapAreas = [2]uint16{0x9800, 0x9C00}

func (p *PPU) GetFrameBuffer() [WIDTH][HEIGHT]uint8 {
	return p.frameBuffer
}

func (p *PPU) Step(cycles int) {
	p.cycles += cycles

	// Disable LCD / PPU
	if !p.ppuEnabled {
		p.ly = 0
		p.ppuMode = HBLANK

		// Clear framebuffer
		for x := range WIDTH {
			for y := range HEIGHT {
				p.frameBuffer[x][y] = 0
			}
		}

		return
	}

	switch p.ppuMode {
	case OAM_SCAN:
		if p.cycles >= OAM_CYCLES {
			i := 0
			p.objectCount = 0

			for i < OAM_SIZE && p.objectCount < 10 {
				y := p.oam[i]

				objSize := uint8(8)
				if p.objSize {
					objSize = 16
				}

				if p.ly+16 >= y && p.ly+16 < objSize+y {
					p.objects[p.objectCount].y = p.oam[i]
					p.objects[p.objectCount].x = p.oam[i+1]
					p.objects[p.objectCount].tileIdx = p.oam[i+2]

					attrs := p.oam[i+3]

					p.objects[p.objectCount].priority = attrs&0x80 != 0
					p.objects[p.objectCount].yFlip = attrs&0x40 != 0
					p.objects[p.objectCount].xFlip = attrs&0x20 != 0
					p.objects[p.objectCount].dmgPalette = attrs&0x10 != 0
					p.objects[p.objectCount].cgbBank = attrs&0x08 != 0
					p.objects[p.objectCount].cgbPalette = attrs & 0x07

					p.objectCount++
				}

				i += 4
			}

			p.ppuMode = DRAW
		}

	case DRAW:
		if p.cycles >= 288 {
			for row := range WIDTH / 8 {
				tileX := uint8(row * 8)
				tileY := p.ly

				// Window not enabled : add scroll
				if !p.windowEnabled {
					tileX += p.scx
					tileY += p.scy
				}

				tilePixelRow := p.getBGWTilePixelRow(tileX, tileY)

				for b := range 8 {
					x := row*8 + b
					p.frameBuffer[x][p.ly] = tilePixelRow[b]
				}
			}

			for i := range p.objectCount {
				objIdx := p.objectCount - 1 - i
				obj := p.objects[objIdx]

				// Skip drawing object if background / window has priority
				// if obj.attrs&0x80 != 0 {
				// 	continue
				// }

				tileIdx := obj.tileIdx

				tileY := p.ly + 16 - obj.y
				if obj.yFlip {
					if tileY < 8 {
						tileIdx &= 0xFE
					} else {
						tileIdx |= 0x01
						tileY -= 8
					}
				}

				tileAddr := TILE_BLOCK_0 + uint16(tileIdx)*16

				tileLo := p.vram[tileAddr+uint16(tileY)*2-VRAM_START]
				tileHi := p.vram[tileAddr+uint16(tileY)*2+1-VRAM_START]

				palette := p.obp0
				if obj.dmgPalette {
					palette = p.obp1
				}

				for b := range 8 {
					pixelIdx := 7 - b
					if obj.xFlip {
						pixelIdx = b
					}

					loPx := (tileLo >> pixelIdx) & 0x1
					hiPx := (tileHi >> pixelIdx) & 0x1
					colorIdx := hiPx<<1 | loPx

					// Transparent pixel
					if colorIdx == 0 {
						continue
					}

					color := (palette >> (colorIdx * 2)) & 0x3

					xDraw := obj.x + uint8(b) - 8
					if xDraw < WIDTH {
						p.frameBuffer[xDraw][p.ly] = color
					}
				}
			}

			p.ppuMode = HBLANK

			if p.hblankInt {
				p.CPU.RequestInterrupt(STAT_INTERRUPT_CODE)
			}
		}

	case HBLANK:
		if p.cycles >= CYCLES_PER_LINE {
			p.cycles = 0

			p.ly++
			if p.ly < HEIGHT {
				p.ppuMode = OAM_SCAN
			} else {
				p.ppuMode = VBLANK
				p.CPU.RequestInterrupt(VBLANK_INTERRUPT_CODE)

				if p.vblankInt {
					p.CPU.RequestInterrupt(STAT_INTERRUPT_CODE)
				}
			}
		}

	case VBLANK:
		if p.cycles >= CYCLES_PER_LINE {
			p.cycles = 0

			p.ly++
			if p.ly == LINES_PER_FRAME {
				p.ppuMode = OAM_SCAN

				p.ly = 0
				if p.oamInt {
					p.CPU.RequestInterrupt(STAT_INTERRUPT_CODE)
				}
			}
		}
	}

	p.lycEqLy = p.ly == p.lyc

	if p.lycEqLy && p.lycInt {
		p.CPU.RequestInterrupt(STAT_INTERRUPT_CODE)
	}
}

func (p *PPU) getBGWTilePixelRow(x, y uint8) [8]uint8 {
	var tilePixelRow [8]uint8

	tileY := y / 8
	tileRow := y % 8
	tileX := x / 8

	// Select BG tile map
	tileMapSelector := btou8(p.bgTileMap)

	// If window enable
	if p.windowEnabled && p.ly >= p.wy && x >= p.wx-7 {
		// Select window tile map
		tileMapSelector = btou8(p.windowTileMap)
	}

	tileMapArea := tileMapAreas[tileMapSelector]
	tileMapIdx := tileMapArea + uint16(tileX) + uint16(tileY)*32
	tileIdx := p.vram[tileMapIdx-VRAM_START]

	// Select BGW tile data area
	tileAddr := TILE_BLOCK_0 + uint16(tileIdx)*16
	if !p.bgwTileData {
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

func (p *PPU) readLCDC() uint8 {
	var value uint8

	value |= btou8(p.ppuEnabled) << 7
	value |= btou8(p.windowTileMap) << 6
	value |= btou8(p.windowEnabled) << 5
	value |= btou8(p.bgwTileData) << 4
	value |= btou8(p.bgTileMap) << 3
	value |= btou8(p.objSize) << 2
	value |= btou8(p.objEnabled) << 1
	value |= btou8(p.bgwEnabled)

	return value
}

func (p *PPU) setLCDC(value uint8) {
	p.ppuEnabled = value&0x80 != 0
	p.windowTileMap = value&0x40 != 0
	p.windowEnabled = value&0x20 != 0
	p.bgwTileData = value&0x10 != 0
	p.bgTileMap = value&0x08 != 0
	p.objSize = value&0x04 != 0
	p.objEnabled = value&0x02 != 0
	p.bgwEnabled = value&0x01 != 0
}

func (p *PPU) readSTAT() uint8 {
	var value uint8

	value |= btou8(p.lycInt) << 6
	value |= btou8(p.oamInt) << 5
	value |= btou8(p.vblankInt) << 4
	value |= btou8(p.hblankInt) << 3
	value |= btou8(p.lycEqLy) << 2
	value |= uint8(p.ppuMode)

	return value
}

func (p *PPU) setSTAT(value uint8) {
	p.lycInt = value&0x40 != 0
	p.oamInt = value&0x20 != 0
	p.vblankInt = value&0x10 != 0
	p.hblankInt = value&0x08 != 0
}

func btou8(b bool) uint8 {
	if b {
		return 1
	}

	return 0
}
