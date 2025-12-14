package ui

import (
	"fmt"

	"github.com/Zyko0/go-sdl3/sdl"
)

type console interface {
	Shutdown()
}

type bus interface {
	Read(addr uint16) uint8
}

type ppu interface {
	GetFramebuffer() [WIDTH * HEIGHT * PIXEL_BYTES]uint8
}

type UI struct {
	Console console
	Bus     bus
	PPU     ppu

	window   *sdl.Window
	renderer *sdl.Renderer
	texture  *sdl.Texture
	surface  *sdl.Surface
}

const (
	WIDTH       = 160
	HEIGHT      = 144
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

func (ui *UI) drawVRAM() {
	framebuffer := ui.PPU.GetFramebuffer()

	if err := ui.texture.Update(nil, framebuffer[:], ui.surface.Pitch); err != nil {
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
