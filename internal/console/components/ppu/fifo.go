package ppu

const (
	X_OFFSET        = 8
	Y_OFFSET        = 16
	TILE_BYTE_SIZE  = 16
	PIXEL_FIFO_SIZE = 8
)

type pixel struct {
	ColorIdx    uint8
	Color       uint8
	BGWPriority bool
}

func (p *PPU) fetchBGWPixels() {
	// Only fetch if fifo empty
	if p.BackgroundFIFO.GetCount() > 0 {
		return
	}

	// 1. Fetch tile idx
	var (
		tileMapOffset uint16
		tileRow       uint16
	)

	tileMapSelector := btou8(p.BGTileMap)

	// Check if we should be fetching a window pixel
	if !p.WindowTriggered && p.WindowEnabled && p.WX <= 166 && p.LY >= p.WY && p.FetchedX+7 >= p.WX {
		p.WindowTriggered = true
		p.BGScanlineContainedWindow = true
	}

	if p.WindowTriggered {
		tileMapSelector = btou8(p.WindowTileMap)
		windowX := (p.FetchedX + 7 - p.WX) / 8
		tileMapOffset = uint16(windowX) + 32*(uint16(p.WindowLineCounter)/8)
		tileRow = uint16(p.WindowLineCounter) % 8
	} else {
		tileX := uint16(((p.FetchedX / 8) + (p.SCX / 8)) & 0x1f)
		tileY := uint16(((p.LY + p.SCY) & 0xFF) / 8)
		tileMapOffset = tileX + 32*tileY
		tileRow = (uint16(p.LY) + uint16(p.SCY)) % 8
	}

	tileMapOffset &= 0x3FF
	tileMapArea := tileMapAreas[tileMapSelector]
	tileIdx := p.VRAM[tileMapArea+tileMapOffset-VRAM_START]

	// Select BGW tile data area
	tileAddr := TILE_BLOCK_0 + uint16(tileIdx)*TILE_BYTE_SIZE
	if !p.BGWTileData {
		tileAddr = uint16(int32(TILE_BLOCK_1) + (int32(int8(tileIdx)))*TILE_BYTE_SIZE)
	}

	// 2. & 3. Fetch tile data
	tileLo := p.VRAM[tileAddr+tileRow*2-VRAM_START]
	tileHi := p.VRAM[tileAddr+tileRow*2+1-VRAM_START]

	// 4. Push to FIFO
	for b := range 8 {
		loPx := (tileLo >> (7 - b)) & 0x1
		hiPx := (tileHi >> (7 - b)) & 0x1
		colorIdx := hiPx<<1 | loPx
		pixel := pixel{
			ColorIdx: colorIdx,
			Color:    (p.BGP >> (colorIdx * 2)) & 0x3,
		}

		p.BackgroundFIFO.Push(pixel)
	}

	p.FetchedX += 8
}

func (p *PPU) fetchObjPixels() {
	// Only fetch if fifo empty
	obj := p.Objects[p.FetchedObjects]

	size := uint8(8)
	if p.ObjSize {
		size = 16
	}

	tileIdx := obj.TileIdx

	tileY := p.LY + Y_OFFSET - obj.Y
	if obj.YFlip {
		tileY = size - 1 - tileY
	}

	if size == 16 {
		if tileY < 8 {
			tileIdx &= 0xFE
		} else {
			tileIdx |= 0x01
			tileY -= 8
		}
	}

	tileAddr := TILE_BLOCK_0 + uint16(tileIdx)*TILE_BYTE_SIZE

	tileLo := p.VRAM[tileAddr+uint16(tileY)*2-VRAM_START]
	tileHi := p.VRAM[tileAddr+uint16(tileY)*2+1-VRAM_START]

	obp := p.OBP0
	if obj.DMGPalette {
		obp = p.OBP1
	}

	startPixel := 0

	if obj.X < X_OFFSET {
		startPixel = int(X_OFFSET - obj.X)
	}

	for px := startPixel; px < 8; px++ {
		pixelIdx := 7 - px
		if obj.XFlip {
			pixelIdx = px
		}

		loPx := (tileLo >> pixelIdx) & 0x1
		hiPx := (tileHi >> pixelIdx) & 0x1
		colorIdx := hiPx<<1 | loPx

		newPixel := pixel{
			ColorIdx:    colorIdx,
			Color:       (obp >> (colorIdx * 2)) & 0x3,
			BGWPriority: obj.BGWPriority,
		}

		fifoIdx := px - startPixel
		if fifoIdx < p.ObjectFIFO.GetCount() {
			// Overwrite existing object in FIFO if it's transparent
			if p.ObjectFIFO.Peek(fifoIdx).ColorIdx == 0 {
				p.ObjectFIFO.Replace(fifoIdx, newPixel)
			}
		} else {
			p.ObjectFIFO.Push(newPixel)
		}
	}

	p.FetchedObjects++
}

func (p *PPU) pushPixelToLCD() {
	bgPixel, ok := p.BackgroundFIFO.Pop()
	if !ok {
		return
	}

	if !p.BGWEnabled {
		bgPixel.Color = 0
		bgPixel.ColorIdx = 0
	}

	finalPixel := bgPixel

	objPixel, ok := p.ObjectFIFO.Pop()
	if ok {
		if p.ObjEnabled && objPixel.ColorIdx != 0 && (!objPixel.BGWPriority || bgPixel.ColorIdx == 0) {
			finalPixel = objPixel
		}
	}

	if p.DiscardedPixels < p.SCX%8 {
		p.DiscardedPixels++
		return
	}

	p.CurrentFrameBuffer[p.PushedX][p.LY] = finalPixel.Color
	p.PushedX++
}
