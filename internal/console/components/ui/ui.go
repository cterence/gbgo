package ui

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

type console interface {
	Shutdown()
}

type bus interface {
	Read(addr uint16) uint8
}

type ppu interface {
	GetFrameBuffer() [WIDTH][HEIGHT]uint8
}

type UI struct {
	Console console
	Bus     bus
	PPU     ppu
}

const (
	WIDTH       = 160
	HEIGHT      = 144
	PIXEL_BYTES = 4
	SCALE       = 4
)

var palette = [4]rl.Color{
	{A: 0xFF, R: 0xFF, G: 0xFF, B: 0xFF},
	{A: 0xFF, R: 0xAA, G: 0xAA, B: 0xAA},
	{A: 0xFF, R: 0x55, G: 0x55, B: 0x55},
	{A: 0xFF, R: 0x00, G: 0x00, B: 0x00},
}

func (ui *UI) Init() error {
	rl.SetTraceLogLevel(rl.LogError)
	rl.InitWindow(WIDTH*SCALE, HEIGHT*SCALE, "gbgo")
	rl.SetTargetFPS(60)

	return nil
}

func (ui *UI) Step() {
	ui.drawFrameBuffer()
	ui.handleEvents()
}

func (ui *UI) drawFrameBuffer() {
	framebuffer := ui.PPU.GetFrameBuffer()

	rl.BeginDrawing()

	for y := range HEIGHT {
		for x := range WIDTH {
			color := palette[framebuffer[x][y]]
			rl.DrawRectangle(int32(x)*SCALE, int32(y)*SCALE, SCALE, SCALE, color)
		}
	}

	rl.EndDrawing()
}

func (ui *UI) handleEvents() {
	if rl.WindowShouldClose() {
		ui.Console.Shutdown()
	}
}

func (ui *UI) Close() {
	rl.CloseWindow()
}
