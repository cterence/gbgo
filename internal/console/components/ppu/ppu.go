package ppu

import (
	"fmt"
	"slices"
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

type ui interface {
	DrawFrameBuffer([WIDTH][HEIGHT]uint8)
}

type object struct {
	y       uint8
	x       uint8
	tileIdx uint8

	// Attributes
	bgwPriority bool
	yFlip       bool
	xFlip       bool
	dmgPalette  bool
	cgbBank     bool
	cgbPalette  uint8
}

type PPU struct {
	Bus bus
	CPU cpu
	UI  ui

	// Pixel FIFO / fetcher
	backgroundFIFO fifo
	objectFIFO     fifo

	lineCycles int

	frameBuffer [WIDTH][HEIGHT]uint8

	vram        [VRAM_SIZE]uint8
	oam         [OAM_SIZE]uint8
	objects     [10]object
	objectCount uint8

	// Pixel FIFO variables
	fetchedX                  uint8
	pushedX                   uint8
	windowLineCounter         uint8
	discardedPixels           uint8
	fetchedObjects            uint8
	windowTriggered           bool
	bgScanlineContainedWindow bool

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
	p.lineCycles = 0
	p.setLCDC(0)
	p.setSTAT(0)
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
		if p.ppuMode != DRAW {
			p.vram[addr-VRAM_START] = value
		}
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

func (p *PPU) Step(cycles int) {
	p.lineCycles += cycles

	if !p.ppuEnabled {
		if p.ly != 0 || p.ppuMode != HBLANK {
			// Clear framebuffer
			for x := range WIDTH {
				for y := range HEIGHT {
					p.frameBuffer[x][y] = 0
				}
			}
		}

		p.ly = 0
		p.ppuMode = HBLANK

		return
	}

	switch p.ppuMode {
	case OAM_SCAN:
		if p.lineCycles >= OAM_CYCLES {
			p.scanOAM()
		}

	case DRAW:
		for p.fetchedObjects < p.objectCount && p.objects[p.fetchedObjects].x <= p.pushedX+X_OFFSET {
			p.fetchObjPixels()
		}

		p.fetchBGWPixels()
		p.pushPixelToLCD()

		if p.pushedX >= WIDTH {
			p.ppuMode = HBLANK

			if p.hblankInt {
				p.CPU.RequestInterrupt(STAT_INTERRUPT_CODE)
			}

			break
		}

	case HBLANK:
		if p.lineCycles >= CYCLES_PER_LINE {
			p.lineCycles = 0
			p.ly++

			p.checkLYC()

			if p.bgScanlineContainedWindow {
				p.windowLineCounter++
				p.bgScanlineContainedWindow = false
			}

			if p.ly < HEIGHT {
				p.ppuMode = OAM_SCAN
			} else {
				p.ppuMode = VBLANK
				p.windowLineCounter = 0
				p.CPU.RequestInterrupt(VBLANK_INTERRUPT_CODE)

				if p.vblankInt {
					p.CPU.RequestInterrupt(STAT_INTERRUPT_CODE)
				}
			}
		}

	case VBLANK:
		if p.lineCycles >= CYCLES_PER_LINE {
			p.lineCycles = 0
			p.ly++
			p.checkLYC()

			if p.ly == LINES_PER_FRAME {
				p.ppuMode = OAM_SCAN
				// TODO: probably shoud not be done here, but we're sure that frame is finished
				p.UI.DrawFrameBuffer(p.frameBuffer)

				p.ly = 0
				if p.oamInt {
					p.CPU.RequestInterrupt(STAT_INTERRUPT_CODE)
				}
			}
		}
	}
}

func (p *PPU) scanOAM() {
	i := 0
	p.objectCount = 0

	for i < OAM_SIZE && p.objectCount < 10 {
		y := p.oam[i]

		objSize := uint8(8)
		if p.objSize {
			objSize = 16
		}

		if p.ly+Y_OFFSET >= y && p.ly+Y_OFFSET < objSize+y {
			p.objects[p.objectCount].y = p.oam[i]
			p.objects[p.objectCount].x = p.oam[i+1]
			p.objects[p.objectCount].tileIdx = p.oam[i+2]

			attrs := p.oam[i+3]

			p.objects[p.objectCount].bgwPriority = attrs&0x80 != 0
			p.objects[p.objectCount].yFlip = attrs&0x40 != 0
			p.objects[p.objectCount].xFlip = attrs&0x20 != 0
			p.objects[p.objectCount].dmgPalette = attrs&0x10 != 0
			p.objects[p.objectCount].cgbBank = attrs&0x08 != 0
			p.objects[p.objectCount].cgbPalette = attrs & 0x07

			p.objectCount++
		}

		i += 4
	}

	// Sort the objects by x coordinate, stable so that objects scanned first retain priority
	slices.SortStableFunc(p.objects[:p.objectCount], func(a, b object) int {
		return int(a.x) - int(b.x)
	})

	p.lineCycles = 0
	p.discardedPixels = 0
	p.fetchedObjects = 0
	p.pushedX = 0
	p.fetchedX = 0
	p.backgroundFIFO.clear()
	p.objectFIFO.clear()
	p.windowTriggered = false

	p.ppuMode = DRAW
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

func (p *PPU) checkLYC() {
	p.lycEqLy = p.ly == p.lyc

	if p.lycEqLy && p.lycInt {
		p.CPU.RequestInterrupt(STAT_INTERRUPT_CODE)
	}
}
