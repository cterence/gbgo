package ui

import (
	"encoding/binary"
	"fmt"

	"github.com/Zyko0/go-sdl3/sdl"
)

type console interface {
	Shutdown()
}

type bus interface {
	Read(addr uint16) uint8
}

type UI struct {
	Console console
	Bus     bus

	window   *sdl.Window
	renderer *sdl.Renderer
	texture  *sdl.Texture
	surface  *sdl.Surface

	framebuffer [WIDTH * HEIGHT * PIXEL_BYTES]uint8
}

const (
	WIDTH       = 144
	HEIGHT      = 160
	PIXEL_BYTES = 4
	SCALE       = 4

	VRAM_BLOCK_0_START = 0x8000
	VRAM_BLOCK_0_END   = 0x87FF

	VRAM_BLOCK_1_START = 0x8800
	VRAM_BLOCK_1_END   = 0x8FFF

	VRAM_BLOCK_2_START = 0x9000
	VRAM_BLOCK_2_END   = 0x97FF

	VRAM_SIZE = VRAM_BLOCK_2_END - VRAM_BLOCK_0_START + 1
)

var palette = [4]uint32{0xFFFFFFFF, 0xFFAAAAAA, 0xFF555555, 0xFF000000}

func (ui *UI) Init() error {
	err := sdl.Init(sdl.INIT_VIDEO)
	if err != nil {
		return fmt.Errorf("failed to init sdl: %w", err)
	}

	if ui.window == nil && ui.renderer == nil {
		ui.window, ui.renderer, err = sdl.CreateWindowAndRenderer("gbgo", WIDTH*SCALE, HEIGHT*SCALE, sdl.WINDOW_RESIZABLE)
		if err != nil {
			return fmt.Errorf("failed to create window and renderer: %w", err)
		}
	}

	if ui.texture == nil {
		ui.texture, err = ui.renderer.CreateTexture(sdl.PIXELFORMAT_ARGB8888, sdl.TEXTUREACCESS_STREAMING, WIDTH, HEIGHT)
		if err != nil {
			return fmt.Errorf("failed to create texture: %w", err)
		}

		if err := ui.texture.SetScaleMode(sdl.SCALEMODE_NEAREST); err != nil {
			return fmt.Errorf("failed to set texture scale mode: %w", err)
		}
	}

	if ui.surface == nil {
		ui.surface, err = sdl.CreateSurface(WIDTH, HEIGHT, sdl.PIXELFORMAT_ARGB8888)
		if err != nil {
			panic("failed to create surface: " + err.Error())
		}
	}

	return nil
}

func (ui *UI) Step() {
	ui.drawVRAM()
	ui.handleEvents()
}

func (ui *UI) getTile(idx uint16) {
	for i := range 8 {
		sl := ui.Bus.Read(uint16(i*2) + idx*16 + VRAM_BLOCK_0_START)
		sh := ui.Bus.Read(uint16(i*2+1) + idx*16 + VRAM_BLOCK_0_START)

		for b := range 8 {
			lo := sl >> (7 - b) & 0x1
			hi := sh >> (7 - b) & 0x1
			pixel := hi<<1 | lo
			color := palette[pixel]
			x := int32(b) + int32((8*idx)%WIDTH)
			y := int32(i) + int32(8*(idx/(WIDTH/8)))
			fbIdx := (x + y*WIDTH) * PIXEL_BYTES
			binary.LittleEndian.PutUint32(ui.framebuffer[fbIdx:fbIdx+PIXEL_BYTES], color)
		}
	}
}

func (ui *UI) drawVRAM() {
	for i := range 256 {
		ui.getTile(uint16(i))
	}

	// yDraw := 0

	// for y := range 8 {
	// 	for x := range 8 {
	// 		binary.LittleEndian.PutUint32(ui.framebuffer[x*PIXEL_BYTES+yDraw:x*PIXEL_BYTES+PIXEL_BYTES+yDraw], tile[x+y])
	// 	}
	// 	yDraw += WIDTH * PIXEL_BYTES
	// }

	if err := ui.texture.Update(nil, ui.framebuffer[:], ui.surface.Pitch); err != nil {
		panic("failed to update texture: " + err.Error())
	}

	if err := ui.renderer.Clear(); err != nil {
		panic("failed to clear renderer: " + err.Error())
	}

	if err := ui.renderer.RenderTexture(ui.texture, nil, nil); err != nil {
		panic("failed to render texture: " + err.Error())
	}

	if err := ui.renderer.Present(); err != nil {
		panic("failed to present UI: " + err.Error())
	}
}

func (ui *UI) handleEvents() {
	var event sdl.Event

	for sdl.PollEvent(&event) {
		switch event.Type {
		case sdl.EVENT_QUIT, sdl.EVENT_WINDOW_DESTROYED:
			ui.Console.Shutdown()
		}
	}
}

func (ui *UI) Close() {
	ui.surface.Destroy()
	ui.texture.Destroy()
	ui.renderer.Destroy()
	ui.window.Destroy()
}
