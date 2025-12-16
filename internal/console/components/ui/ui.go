package ui

import (
	"fmt"

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

type cpu interface {
	RequestInterrupt(code uint8)
}

type buttons struct {
	a   bool
	b   bool
	st  bool
	sel bool
	u   bool
	d   bool
	l   bool
	r   bool
}

type UI struct {
	Console console
	Bus     bus
	PPU     ppu
	CPU     cpu

	buttons buttons

	joypad uint8
}

const (
	WIDTH       = 160
	HEIGHT      = 144
	PIXEL_BYTES = 4
	SCALE       = 4
	FPS         = 60

	INTERRUPT_CODE = 0x10

	JOYPAD = 0xFF00
)

var palette = [4]rl.Color{
	{A: 0xFF, R: 0xFF, G: 0xFF, B: 0xFF},
	{A: 0xFF, R: 0xAA, G: 0xAA, B: 0xAA},
	{A: 0xFF, R: 0x55, G: 0x55, B: 0x55},
	{A: 0xFF, R: 0x00, G: 0x00, B: 0x00},
}

func (ui *UI) Init() {
	rl.SetTraceLogLevel(rl.LogError)
	rl.InitWindow(WIDTH*SCALE, HEIGHT*SCALE, "gbgo")
	rl.SetTargetFPS(FPS)

	ui.joypad = 0xCF
}

func (ui *UI) Read(addr uint16) uint8 {
	switch addr {
	case JOYPAD:
		result := uint8(0xCF) | ui.joypad

		if ui.joypad&0x10 == 0 {
			if ui.buttons.r {
				result &^= 0x1
			}

			if ui.buttons.l {
				result &^= 0x2
			}

			if ui.buttons.u {
				result &^= 0x4
			}

			if ui.buttons.d {
				result &^= 0x8
			}
		}

		if ui.joypad&0x20 == 0 {
			if ui.buttons.a {
				result &^= 0x1
			}

			if ui.buttons.b {
				result &^= 0x2
			}

			if ui.buttons.sel {
				result &^= 0x4
			}

			if ui.buttons.st {
				result &^= 0x8
			}
		}

		return result
	default:
		panic(fmt.Errorf("unsupported read for ui: %x", addr))
	}
}

func (ui *UI) Write(addr uint16, value uint8) {
	switch addr {
	case JOYPAD:
		ui.joypad = value & 0x30
	default:
		panic(fmt.Errorf("unsupported write for ui: %x", addr))
	}
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

	prevA := ui.buttons.a
	prevRight := ui.buttons.r
	prevB := ui.buttons.b
	prevLeft := ui.buttons.l
	prevSelectB := ui.buttons.sel
	prevUp := ui.buttons.u
	prevStart := ui.buttons.st
	prevDown := ui.buttons.d

	ui.buttons.a = rl.IsKeyDown(rl.KeyA)
	ui.buttons.b = rl.IsKeyDown(rl.KeyD)
	ui.buttons.st = rl.IsKeyDown(rl.KeyQ)
	ui.buttons.sel = rl.IsKeyDown(rl.KeyE)
	ui.buttons.u = rl.IsKeyDown(rl.KeyUp)
	ui.buttons.d = rl.IsKeyDown(rl.KeyDown)
	ui.buttons.r = rl.IsKeyDown(rl.KeyRight)
	ui.buttons.l = rl.IsKeyDown(rl.KeyLeft)

	// Trigger interrupt on rising edge
	if (ui.joypad&0x10 == 0 && (ui.buttons.st || !prevRight && ui.buttons.r || !prevLeft && ui.buttons.l || !prevUp && ui.buttons.u || !prevStart && !prevDown && ui.buttons.d)) || (ui.joypad&0x20 == 0 && (!prevA && ui.buttons.a || !prevB && ui.buttons.b || !prevSelectB && ui.buttons.sel)) {
		ui.CPU.RequestInterrupt(INTERRUPT_CODE)
	}
}

func (ui *UI) Close() {
	rl.CloseWindow()
}
