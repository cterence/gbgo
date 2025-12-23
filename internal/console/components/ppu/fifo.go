package ppu

const (
	X_OFFSET        = 8
	Y_OFFSET        = 16
	TILE_BYTE_SIZE  = 16
	PIXEL_FIFO_SIZE = 8
)

type pixel struct {
	colorIdx    uint8
	color       uint8
	bgwPriority bool
}

type fifo[T any] struct {
	elements []T
	head     int
	tail     int
	count    int
}

func (f *fifo[T]) push(p T) {
	if f.count == len(f.elements) {
		return
	}

	f.elements[f.tail] = p
	f.tail = (f.tail + 1) % len(f.elements)
	f.count++
}

func (f *fifo[T]) pop() (T, bool) {
	if f.count == 0 {
		var zero T
		return zero, false
	}

	p := f.elements[f.head]
	f.head = (f.head + 1) % len(f.elements)
	f.count--

	return p, true
}

func (f *fifo[T]) clear() {
	f.count = 0
	f.head = 0
	f.tail = 0
}

func (p *PPU) fetchBGWPixels() {
	// Only fetch if fifo empty
	if p.backgroundFIFO.count > 0 {
		return
	}

	// 1. Fetch tile idx
	var (
		tileMapOffset uint16
		tileRow       uint16
	)

	tileMapSelector := btou8(p.bgTileMap)

	// Check if we should be fetching a window pixel
	if !p.windowTriggered && p.windowEnabled && p.wx <= 166 && p.ly >= p.wy && p.fetchedX+7 >= p.wx {
		p.windowTriggered = true
		p.bgScanlineContainedWindow = true
	}

	if p.windowTriggered {
		tileMapSelector = btou8(p.windowTileMap)
		windowX := (p.fetchedX + 7 - p.wx) / 8
		tileMapOffset = uint16(windowX) + 32*(uint16(p.windowLineCounter)/8)
		tileRow = uint16(p.windowLineCounter) % 8
	} else {
		tileX := uint16(((p.fetchedX / 8) + (p.scx / 8)) & 0x1f)
		tileY := uint16(((p.ly + p.scy) & 0xFF) / 8)
		tileMapOffset = tileX + 32*tileY
		tileRow = (uint16(p.ly) + uint16(p.scy)) % 8
	}

	tileMapOffset &= 0x3FF
	tileMapArea := tileMapAreas[tileMapSelector]
	tileIdx := p.vram[tileMapArea+tileMapOffset-VRAM_START]

	// Select BGW tile data area
	tileAddr := TILE_BLOCK_0 + uint16(tileIdx)*TILE_BYTE_SIZE
	if !p.bgwTileData {
		tileAddr = uint16(int32(TILE_BLOCK_1) + (int32(int8(tileIdx)))*TILE_BYTE_SIZE)
	}

	// 2. & 3. Fetch tile data
	tileLo := p.vram[tileAddr+tileRow*2-VRAM_START]
	tileHi := p.vram[tileAddr+tileRow*2+1-VRAM_START]

	// 4. Push to FIFO
	for b := range 8 {
		loPx := (tileLo >> (7 - b)) & 0x1
		hiPx := (tileHi >> (7 - b)) & 0x1
		colorIdx := hiPx<<1 | loPx
		pixel := pixel{
			colorIdx: colorIdx,
			color:    (p.bgp >> (colorIdx * 2)) & 0x3,
		}

		p.backgroundFIFO.push(pixel)
	}

	p.fetchedX += 8
}

func (p *PPU) fetchObjPixels() {
	// Only fetch if fifo empty
	obj := p.objects[p.fetchedObjects]

	size := uint8(8)
	if p.objSize {
		size = 16
	}

	tileIdx := obj.tileIdx

	tileY := p.ly + Y_OFFSET - obj.y
	if obj.yFlip {
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

	tileLo := p.vram[tileAddr+uint16(tileY)*2-VRAM_START]
	tileHi := p.vram[tileAddr+uint16(tileY)*2+1-VRAM_START]

	obp := p.obp0
	if obj.dmgPalette {
		obp = p.obp1
	}

	startPixel := 0

	if obj.x < X_OFFSET {
		startPixel = int(X_OFFSET - obj.x)
	}

	for px := startPixel; px < 8; px++ {
		pixelIdx := 7 - px
		if obj.xFlip {
			pixelIdx = px
		}

		loPx := (tileLo >> pixelIdx) & 0x1
		hiPx := (tileHi >> pixelIdx) & 0x1
		colorIdx := hiPx<<1 | loPx

		newPixel := pixel{
			colorIdx:    colorIdx,
			color:       (obp >> (colorIdx * 2)) & 0x3,
			bgwPriority: obj.bgwPriority,
		}

		fifoIdx := px - startPixel
		if fifoIdx < p.objectFIFO.count {
			// Only overwrite if existing pixel is transparent
			if p.objectFIFO.elements[fifoIdx].colorIdx == 0 {
				p.objectFIFO.elements[fifoIdx] = newPixel
			}
		} else {
			p.objectFIFO.push(newPixel)
		}
	}

	p.fetchedObjects++
}

func (p *PPU) pushPixelToLCD() {
	bgPixel, ok := p.backgroundFIFO.pop()
	if !ok {
		return
	}

	if !p.bgwEnabled {
		bgPixel.color = 0
		bgPixel.colorIdx = 0
	}

	finalPixel := bgPixel

	objPixel, ok := p.objectFIFO.pop()
	if ok {
		if p.objEnabled && objPixel.colorIdx != 0 && (!objPixel.bgwPriority || bgPixel.colorIdx == 0) {
			finalPixel = objPixel
		}
	}

	if p.discardedPixels < p.scx%8 {
		p.discardedPixels++
		return
	}

	p.currentFrameBuffer[p.pushedX][p.ly] = finalPixel.color
	p.pushedX++
}
