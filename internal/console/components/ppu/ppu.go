package ppu

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"slices"

	"github.com/cterence/gbgo/internal/lib"
)

const (
	OAM_CYCLES       = 80
	CYCLES_PER_LINE  = 456
	LINES_PER_FRAME  = 154
	FRAMEBUFFER_SIZE = 3

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
	Y       uint8
	X       uint8
	TileIdx uint8

	// Attributes
	BGWPriority bool
	YFlip       bool
	XFlip       bool
	DMGPalette  bool
	CGBBank     bool
	CGBPalette  uint8
}

type frameBuffer [WIDTH][HEIGHT]uint8

type PPU struct {
	Bus bus
	CPU cpu
	state
}

type state struct {
	// Pixel FIFO / fetcher
	BackgroundFIFO lib.FIFO[pixel]
	ObjectFIFO     lib.FIFO[pixel]

	LineCycles int

	CurrentFrameBuffer frameBuffer
	CompletedFrame     frameBuffer

	VRAM        [VRAM_SIZE]uint8
	OAM         [OAM_SIZE]uint8
	Objects     [10]object
	ObjectCount uint8

	// Pixel FIFO variables
	FetchedX                  uint8
	PushedX                   uint8
	WindowLineCounter         uint8
	DiscardedPixels           uint8
	FetchedObjects            uint8
	WindowTriggered           bool
	BGScanlineContainedWindow bool

	// LDCD
	PPUEnabled    bool
	WindowTileMap bool
	WindowEnabled bool
	BGWTileData   bool
	BGTileMap     bool
	ObjSize       bool
	ObjEnabled    bool
	BGWEnabled    bool

	// STAT
	LYCInt    bool
	OAMInt    bool
	VBlankInt bool
	HBlankInt bool
	LYCEqLy   bool
	PPUMode   ppuMode

	SCY  uint8
	SCX  uint8
	LY   uint8
	LYC  uint8
	BGP  uint8
	OBP0 uint8
	OBP1 uint8
	WY   uint8
	WX   uint8

	DMAActive bool

	Frames     uint64
	FrameReady bool
}

func (p *PPU) Init() {
	p.LineCycles = 0
	p.setLCDC(0)
	p.setSTAT(0)
	p.SCY = 0
	p.SCX = 0
	p.LY = 0
	p.LYC = 0
	p.BGP = 0
	p.OBP0 = 0
	p.OBP1 = 0
	p.WY = 0
	p.WX = 0
	p.ObjectCount = 0
	p.FetchedX = 0
	p.PushedX = 0
	p.WindowLineCounter = 0
	p.DiscardedPixels = 0
	p.FetchedObjects = 0
	p.WindowTriggered = false
	p.BGScanlineContainedWindow = false
	p.PPUEnabled = false
	p.WindowTileMap = false
	p.WindowEnabled = false
	p.BGWTileData = false
	p.BGTileMap = false
	p.ObjSize = false
	p.ObjEnabled = false
	p.BGWEnabled = false
	p.DMAActive = false
	p.Frames = 0
	p.LYCInt = false
	p.OAMInt = false
	p.VBlankInt = false
	p.HBlankInt = false
	p.LYCEqLy = false
	p.PPUMode = OAM_SCAN
	p.CurrentFrameBuffer = frameBuffer{}
	p.CompletedFrame = frameBuffer{}
	p.VRAM = [VRAM_SIZE]uint8{}
	p.OAM = [OAM_SIZE]uint8{}
	p.Objects = [10]object{}
	p.FrameReady = false

	p.BackgroundFIFO.Init(PIXEL_FIFO_SIZE)
	p.ObjectFIFO.Init(PIXEL_FIFO_SIZE)

	p.PPUMode = OAM_SCAN
}

func (p *PPU) Read(addr uint16) uint8 {
	switch {
	case addr >= VRAM_START && addr <= VRAM_END:
		if p.PPUMode == DRAW {
			return 0xFF
		}

		return p.VRAM[addr-VRAM_START]
	case addr >= OAM_START && addr <= OAM_END:
		if p.DMAActive || (p.PPUMode == OAM_SCAN || p.PPUMode == DRAW) {
			return 0xFF
		}

		return p.OAM[addr-OAM_START]
	default:
		switch addr {
		case LCDC:
			return p.readLCDC()
		case STAT:
			return p.readSTAT()
		case SCY:
			return p.SCY
		case SCX:
			return p.SCX
		case LY:
			return p.LY
		case LYC:
			return p.LYC
		case BGP:
			return p.BGP
		case OBP0:
			return p.OBP0
		case OBP1:
			return p.OBP1
		case WY:
			return p.WY
		case WX:
			return p.WX
		default:
			panic(fmt.Errorf("unsupported read for ppu: %x", addr))
		}
	}
}

func (p *PPU) Write(addr uint16, value uint8) {
	switch {
	case addr >= VRAM_START && addr <= VRAM_END:
		if p.PPUMode != DRAW {
			p.VRAM[addr-VRAM_START] = value
		}
	case addr >= OAM_START && addr <= OAM_END:
		if !p.DMAActive && p.PPUMode != OAM_SCAN && p.PPUMode != DRAW {
			p.OAM[addr-OAM_START] = value
		}
	default:
		switch addr {
		case LCDC:
			p.setLCDC(value)
		case STAT:
			p.setSTAT(value)
		case SCY:
			p.SCY = value
		case SCX:
			p.SCX = value
		case LY:
			p.LY = value
		case LYC:
			p.LYC = value
		case BGP:
			p.BGP = value
		case OBP0:
			p.OBP0 = value
		case OBP1:
			p.OBP1 = value
		case WY:
			p.WY = value
		case WX:
			p.WX = value
		default:
			panic(fmt.Errorf("unsupported write for ppu: %x", addr))
		}
	}
}

func (p *PPU) WriteOAM(addr uint16, value uint8) {
	p.OAM[addr-OAM_START] = value
}

func (p *PPU) ToggleDMAActive(active bool) {
	p.DMAActive = active
}

var tileMapAreas = [2]uint16{0x9800, 0x9C00}

func (p *PPU) Step(cycles int) {
	if !p.PPUEnabled {
		if p.LY != 0 || p.PPUMode != HBLANK {
			p.CurrentFrameBuffer.clear()

			p.LY = 0
			p.LineCycles = 0
			p.PPUMode = HBLANK
		}

		return
	}

	p.LineCycles += cycles

	switch p.PPUMode {
	case OAM_SCAN:
		if p.LineCycles >= OAM_CYCLES {
			p.scanOAM()
		}

	case DRAW:
		for p.FetchedObjects < p.ObjectCount && p.Objects[p.FetchedObjects].X <= p.PushedX+X_OFFSET {
			p.fetchObjPixels()
		}

		p.fetchBGWPixels()
		p.pushPixelToLCD()

		if p.PushedX >= WIDTH {
			p.PPUMode = HBLANK

			if p.HBlankInt {
				p.CPU.RequestInterrupt(STAT_INTERRUPT_CODE)
			}

			break
		}

	case HBLANK:
		if p.LineCycles >= CYCLES_PER_LINE {
			p.LineCycles = 0
			p.LY++

			p.checkLYC()

			if p.BGScanlineContainedWindow {
				p.WindowLineCounter++
				p.BGScanlineContainedWindow = false
			}

			if p.LY < HEIGHT {
				p.PPUMode = OAM_SCAN
			} else {
				p.PPUMode = VBLANK
				p.WindowLineCounter = 0

				copy(p.CompletedFrame[:], p.CurrentFrameBuffer[:])
				p.CurrentFrameBuffer.clear()
				p.Frames++
				p.FrameReady = true

				p.CPU.RequestInterrupt(VBLANK_INTERRUPT_CODE)

				if p.VBlankInt {
					p.CPU.RequestInterrupt(STAT_INTERRUPT_CODE)
				}
			}
		}

	case VBLANK:
		if p.LineCycles >= CYCLES_PER_LINE {
			p.LineCycles = 0
			p.LY++
			p.checkLYC()

			if p.LY == LINES_PER_FRAME {
				p.PPUMode = OAM_SCAN
				p.LY = 0

				if p.OAMInt {
					p.CPU.RequestInterrupt(STAT_INTERRUPT_CODE)
				}
			}
		}
	}
}

func (p *PPU) Load(buf *bytes.Reader) {
	enc := gob.NewDecoder(buf)
	err := enc.Decode(&p.state)

	lib.Assert(err == nil, "failed to decode state: %v", err)
}

func (p *PPU) Save(buf *bytes.Buffer) {
	enc := gob.NewEncoder(buf)
	err := enc.Encode(p.state)

	lib.Assert(err == nil, "failed to encode state: %v", err)
}

func (p *PPU) GetFrame() [WIDTH][HEIGHT]uint8 {
	p.FrameReady = false
	return p.CompletedFrame
}

func (p *PPU) IsFrameReady() bool {
	return p.FrameReady
}

func (p *PPU) scanOAM() {
	i := 0
	p.ObjectCount = 0

	for i < OAM_SIZE && p.ObjectCount < 10 {
		y := p.OAM[i]

		objSize := uint8(8)
		if p.ObjSize {
			objSize = 16
		}

		if p.LY+Y_OFFSET >= y && p.LY+Y_OFFSET < objSize+y {
			p.Objects[p.ObjectCount].Y = p.OAM[i]
			p.Objects[p.ObjectCount].X = p.OAM[i+1]
			p.Objects[p.ObjectCount].TileIdx = p.OAM[i+2]

			attrs := p.OAM[i+3]

			p.Objects[p.ObjectCount].BGWPriority = attrs&0x80 != 0
			p.Objects[p.ObjectCount].YFlip = attrs&0x40 != 0
			p.Objects[p.ObjectCount].XFlip = attrs&0x20 != 0
			p.Objects[p.ObjectCount].DMGPalette = attrs&0x10 != 0
			p.Objects[p.ObjectCount].CGBBank = attrs&0x08 != 0
			p.Objects[p.ObjectCount].CGBPalette = attrs & 0x07

			p.ObjectCount++
		}

		i += 4
	}

	// Sort the objects by x coordinate, stable so that objects scanned first retain priority
	slices.SortStableFunc(p.Objects[:p.ObjectCount], func(a, b object) int {
		return int(a.X) - int(b.X)
	})

	p.DiscardedPixels = 0
	p.FetchedObjects = 0
	p.PushedX = 0
	p.FetchedX = 0
	p.BackgroundFIFO.Clear()
	p.ObjectFIFO.Clear()
	p.WindowTriggered = false

	p.PPUMode = DRAW
}

func (p *PPU) readLCDC() uint8 {
	var value uint8

	value |= btou8(p.PPUEnabled) << 7
	value |= btou8(p.WindowTileMap) << 6
	value |= btou8(p.WindowEnabled) << 5
	value |= btou8(p.BGWTileData) << 4
	value |= btou8(p.BGTileMap) << 3
	value |= btou8(p.ObjSize) << 2
	value |= btou8(p.ObjEnabled) << 1
	value |= btou8(p.BGWEnabled)

	return value
}

func (p *PPU) setLCDC(value uint8) {
	p.PPUEnabled = value&0x80 != 0
	p.WindowTileMap = value&0x40 != 0
	p.WindowEnabled = value&0x20 != 0
	p.BGWTileData = value&0x10 != 0
	p.BGTileMap = value&0x08 != 0
	p.ObjSize = value&0x04 != 0
	p.ObjEnabled = value&0x02 != 0
	p.BGWEnabled = value&0x01 != 0
}

func (p *PPU) readSTAT() uint8 {
	var value uint8

	value |= btou8(p.LYCInt) << 6
	value |= btou8(p.OAMInt) << 5
	value |= btou8(p.VBlankInt) << 4
	value |= btou8(p.HBlankInt) << 3
	value |= btou8(p.LYCEqLy) << 2
	value |= uint8(p.PPUMode)

	return value
}

func (p *PPU) setSTAT(value uint8) {
	p.LYCInt = value&0x40 != 0
	p.OAMInt = value&0x20 != 0
	p.VBlankInt = value&0x10 != 0
	p.HBlankInt = value&0x08 != 0
}

func btou8(b bool) uint8 {
	if b {
		return 1
	}

	return 0
}

func (p *PPU) checkLYC() {
	p.LYCEqLy = p.LY == p.LYC

	if p.LYCEqLy && p.LYCInt {
		p.CPU.RequestInterrupt(STAT_INTERRUPT_CODE)
	}
}

func (f *frameBuffer) clear() {
	for x := range len(f) {
		for y := range len(f[0]) {
			f[x][y] = 0
		}
	}
}
