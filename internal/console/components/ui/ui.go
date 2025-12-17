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

	img     *rl.Image
	pixels  []rl.Color
	texture rl.Texture2D

	buttons buttons

	joypad uint8
}

const (
	WIDTH         = 160
	HEIGHT        = 144
	INITIAL_SCALE = 4
	FPS           = 60

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
	rl.SetConfigFlags(rl.FlagWindowResizable)
	rl.InitWindow(WIDTH*INITIAL_SCALE, HEIGHT*INITIAL_SCALE, "gbgo")
	rl.SetTargetFPS(FPS)

	ui.img = rl.GenImageColor(WIDTH, HEIGHT, rl.Black)
	ui.texture = rl.LoadTextureFromImage(ui.img)
	rl.SetTextureFilter(ui.texture, rl.FilterPoint)
	ui.pixels = make([]rl.Color, WIDTH*HEIGHT)

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
			ui.pixels[y*WIDTH+x] = color
		}
	}

	currentScale := min(float32(rl.GetScreenWidth())/WIDTH, float32(rl.GetScreenHeight())/HEIGHT)
	src := rl.Rectangle{X: 0, Y: 0, Width: WIDTH, Height: HEIGHT}
	dst := rl.Rectangle{X: (float32(rl.GetScreenWidth()) - WIDTH*currentScale) / 2, Y: (float32(rl.GetScreenHeight()) - HEIGHT*currentScale) / 2, Width: WIDTH * currentScale, Height: HEIGHT * currentScale}

	rl.UpdateTexture(ui.texture, ui.pixels[:])
	rl.DrawTexturePro(ui.texture, src, dst, rl.Vector2{X: 0, Y: 0}, 0, rl.White)

	rl.EndDrawing()
}

func (ui *UI) handleEvents() {
	if rl.WindowShouldClose() {
		ui.Console.Shutdown()
	}

	prevA := ui.buttons.a
	prevR := ui.buttons.r
	prevB := ui.buttons.b
	prevL := ui.buttons.l
	prevSel := ui.buttons.sel
	prevU := ui.buttons.u
	prevSt := ui.buttons.st
	prevD := ui.buttons.d

	ui.buttons.a = rl.IsKeyDown(rl.KeyX)
	ui.buttons.b = rl.IsKeyDown(rl.KeyZ)
	ui.buttons.st = rl.IsKeyDown(rl.KeyEnter)
	ui.buttons.sel = rl.IsKeyDown(rl.KeyBackspace)
	ui.buttons.u = rl.IsKeyDown(rl.KeyUp)
	ui.buttons.d = rl.IsKeyDown(rl.KeyDown)
	ui.buttons.l = rl.IsKeyDown(rl.KeyLeft)
	ui.buttons.r = rl.IsKeyDown(rl.KeyRight)

	gamepad := int32(1) // Keyboard is gamepad 0

	if rl.IsGamepadAvailable(gamepad) {
		ui.buttons.a = rl.IsGamepadButtonDown(gamepad, rl.GamepadButtonRightFaceRight) || rl.IsGamepadButtonDown(gamepad, rl.GamepadButtonRightFaceUp)
		ui.buttons.b = rl.IsGamepadButtonDown(gamepad, rl.GamepadButtonRightFaceDown) || rl.IsGamepadButtonDown(gamepad, rl.GamepadButtonRightFaceLeft)
		ui.buttons.st = rl.IsGamepadButtonDown(gamepad, rl.GamepadButtonMiddleRight)
		ui.buttons.sel = rl.IsGamepadButtonDown(gamepad, rl.GamepadButtonMiddleLeft)
		ui.buttons.u = rl.IsGamepadButtonDown(gamepad, rl.GamepadButtonLeftFaceUp) || rl.GetGamepadAxisMovement(gamepad, rl.GamepadAxisLeftY) < -0.5
		ui.buttons.d = rl.IsGamepadButtonDown(gamepad, rl.GamepadButtonLeftFaceDown) || rl.GetGamepadAxisMovement(gamepad, rl.GamepadAxisLeftY) > 0.5
		ui.buttons.l = rl.IsGamepadButtonDown(gamepad, rl.GamepadButtonLeftFaceLeft) || rl.GetGamepadAxisMovement(gamepad, rl.GamepadAxisLeftX) < -0.5
		ui.buttons.r = rl.IsGamepadButtonDown(gamepad, rl.GamepadButtonLeftFaceRight) || rl.GetGamepadAxisMovement(gamepad, rl.GamepadAxisLeftX) > 0.5
	}

	dpadPressed, buttonPressed := false, false

	if ui.joypad&0x10 == 0 {
		dpadPressed = (!prevR && ui.buttons.r) ||
			(!prevL && ui.buttons.l) ||
			(!prevU && ui.buttons.u) ||
			(!prevD && ui.buttons.d)
	}

	if ui.joypad&0x20 == 0 {
		buttonPressed = (!prevA && ui.buttons.a) ||
			(!prevB && ui.buttons.b) ||
			(!prevSel && ui.buttons.sel) ||
			(!prevSt && ui.buttons.st)
	}

	// Trigger interrupt on rising edge
	if dpadPressed || buttonPressed {
		ui.CPU.RequestInterrupt(INTERRUPT_CODE)
	}
}

func (ui *UI) Close() {
	rl.CloseWindow()
}
