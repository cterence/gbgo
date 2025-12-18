package ui

import (
	"fmt"
	"strconv"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	WIDTH         = 160
	HEIGHT        = 144
	INITIAL_SCALE = 4
	FPS           = 60

	AXIS_TRIGGER = 0.5

	INTERRUPT_CODE = 0x10

	JOYPAD = 0xFF00
)

type console interface {
	Shutdown()
	Pause()
}

type bus interface {
	Read(addr uint16) uint8
}

type cpu interface {
	RequestInterrupt(code uint8)
}

type buttonState struct {
	keyboardKeys       []int32
	gamepadButtons     []int32
	gamepadAxis        []int32
	gamepadAxisTrigger float32
	justPressed        bool
	currentlyPressed   bool
}

type button uint8

const (
	// Console buttons
	A button = iota
	B
	START
	SELECT
	UP
	DOWN
	LEFT
	RIGHT
	// Other buttons
	TURBO
	PAUSE
	SLOWMO
)

type UI struct {
	Console console
	Bus     bus
	CPU     cpu

	img         *rl.Image
	windowTitle string
	pixels      []rl.Color

	cycles    uint64
	cpuCycles uint64
	texture   rl.Texture2D

	currentFPS int32

	joypad uint8

	paused bool
}

var palette = [4]rl.Color{
	{A: 0xFF, R: 0xFF, G: 0xFF, B: 0xFF},
	{A: 0xFF, R: 0xAA, G: 0xAA, B: 0xAA},
	{A: 0xFF, R: 0x55, G: 0x55, B: 0x55},
	{A: 0xFF, R: 0x00, G: 0x00, B: 0x00},
}

var buttons = []buttonState{
	// A
	{
		keyboardKeys:   []int32{rl.KeyX},
		gamepadButtons: []int32{rl.GamepadButtonRightFaceRight, rl.GamepadButtonRightFaceUp},
	},
	// B
	{
		keyboardKeys:   []int32{rl.KeyZ},
		gamepadButtons: []int32{rl.GamepadButtonRightFaceLeft, rl.GamepadButtonRightFaceDown},
	},
	// START
	{
		keyboardKeys:   []int32{rl.KeyEnter},
		gamepadButtons: []int32{rl.GamepadButtonMiddleRight},
	},
	// SELECT
	{
		keyboardKeys:   []int32{rl.KeyBackspace},
		gamepadButtons: []int32{rl.GamepadButtonMiddleLeft},
	},
	// UP
	{
		keyboardKeys:       []int32{rl.KeyUp},
		gamepadButtons:     []int32{rl.GamepadButtonLeftFaceUp},
		gamepadAxis:        []int32{rl.GamepadAxisLeftY},
		gamepadAxisTrigger: -AXIS_TRIGGER,
	},
	// DOWN
	{
		keyboardKeys:       []int32{rl.KeyDown},
		gamepadButtons:     []int32{rl.GamepadButtonLeftFaceDown},
		gamepadAxis:        []int32{rl.GamepadAxisLeftY},
		gamepadAxisTrigger: AXIS_TRIGGER,
	},
	// LEFT
	{
		keyboardKeys:       []int32{rl.KeyLeft},
		gamepadButtons:     []int32{rl.GamepadButtonLeftFaceLeft},
		gamepadAxis:        []int32{rl.GamepadAxisLeftX},
		gamepadAxisTrigger: -AXIS_TRIGGER,
	},
	// RIGHT
	{
		keyboardKeys:       []int32{rl.KeyRight},
		gamepadButtons:     []int32{rl.GamepadButtonLeftFaceRight},
		gamepadAxis:        []int32{rl.GamepadAxisLeftX},
		gamepadAxisTrigger: AXIS_TRIGGER,
	},
	// TURBO
	{
		keyboardKeys:   []int32{rl.KeySpace},
		gamepadButtons: []int32{rl.GamepadButtonRightTrigger2},
	},
	// PAUSE
	{
		keyboardKeys:   []int32{rl.KeyRightShift},
		gamepadButtons: []int32{rl.GamepadButtonMiddle},
	},
	// SLOWMO
	{
		keyboardKeys:   []int32{rl.KeyLeftShift},
		gamepadButtons: []int32{rl.GamepadButtonLeftTrigger2},
	},
}

// TODO: better system for choosing controller
var gamepad = int32(1)

func (ui *UI) Init(romTitle string) {
	ui.windowTitle = "gbgo - " + romTitle

	rl.SetTraceLogLevel(rl.LogError)
	rl.SetConfigFlags(rl.FlagWindowResizable)
	rl.InitWindow(WIDTH*INITIAL_SCALE, HEIGHT*INITIAL_SCALE, ui.windowTitle)
	rl.SetTargetFPS(FPS)
	rl.HideCursor()

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
			if buttons[RIGHT].currentlyPressed {
				result &^= 0x1
			}

			if buttons[LEFT].currentlyPressed {
				result &^= 0x2
			}

			if buttons[UP].currentlyPressed {
				result &^= 0x4
			}

			if buttons[DOWN].currentlyPressed {
				result &^= 0x8
			}
		}

		if ui.joypad&0x20 == 0 {
			if buttons[A].currentlyPressed {
				result &^= 0x1
			}

			if buttons[B].currentlyPressed {
				result &^= 0x2
			}

			if buttons[SELECT].currentlyPressed {
				result &^= 0x4
			}

			if buttons[START].currentlyPressed {
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

func (ui *UI) Step(cycles int) {
	ui.cpuCycles += uint64(cycles)

	ui.handleEvents()

	ui.cycles++
}
func (ui *UI) DrawFrameBuffer(frameBuffer [WIDTH][HEIGHT]uint8) {
	if ui.img == nil {
		return
	}

	rl.BeginDrawing()

	for y := range HEIGHT {
		for x := range WIDTH {
			color := palette[frameBuffer[x][y]]
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

		return
	}

	ui.updateButtonsState()

	dpadPressed, buttonPressed := false, false

	if ui.joypad&0x10 == 0 {
		dpadPressed = buttons[UP].justPressed || buttons[DOWN].justPressed || buttons[LEFT].justPressed || buttons[RIGHT].justPressed
	}

	if ui.joypad&0x20 == 0 {
		buttonPressed = buttons[A].justPressed || buttons[B].justPressed || buttons[START].justPressed || buttons[SELECT].justPressed
	}

	// Trigger interrupt on rising edge
	if dpadPressed || buttonPressed {
		ui.CPU.RequestInterrupt(INTERRUPT_CODE)
	}

	fpsTarget := FPS

	if buttons[SLOWMO].currentlyPressed {
		fpsTarget = FPS * 0.5
	}

	if buttons[TURBO].currentlyPressed {
		fpsTarget = FPS * 4
	}

	rl.SetTargetFPS(int32(fpsTarget))

	if buttons[PAUSE].justPressed {
		ui.paused = !ui.paused
		ui.Console.Pause()

		if ui.paused {
			rl.SetWindowTitle(ui.windowTitle + " - PAUSED")
		} else {
			ui.updateTitleFPS()
		}
	}

	ui.currentFPS = rl.GetFPS()

	// Update FPS in title every second
	if !ui.paused && ui.cycles%uint64(fpsTarget) == 0 {
		ui.updateTitleFPS()
	}
}

func (ui *UI) Close() {
	rl.CloseWindow()
}

func (ui *UI) updateButtonsState() {
	for i, b := range buttons {
		previouslyPressed := b.currentlyPressed
		currentlyPressed := false

		for _, key := range b.keyboardKeys {
			currentlyPressed = currentlyPressed || rl.IsKeyDown(key)
		}

		if rl.IsGamepadAvailable(gamepad) {
			for _, in := range b.gamepadButtons {
				currentlyPressed = currentlyPressed || rl.IsGamepadButtonDown(gamepad, in)
			}

			for _, axis := range b.gamepadAxis {
				if b.gamepadAxisTrigger != 0 {
					if b.gamepadAxisTrigger > 0 {
						currentlyPressed = currentlyPressed || rl.GetGamepadAxisMovement(gamepad, axis) > b.gamepadAxisTrigger
					} else {
						currentlyPressed = currentlyPressed || rl.GetGamepadAxisMovement(gamepad, axis) < b.gamepadAxisTrigger
					}
				}
			}
		}

		buttons[i].currentlyPressed = currentlyPressed
		buttons[i].justPressed = !previouslyPressed && currentlyPressed
	}
}

func (ui *UI) updateTitleFPS() {
	fps := strconv.FormatInt(int64(ui.currentFPS), 10)

	// Don't show first FPS measurement as it's imprecise
	if ui.cycles == 0 {
		fps = "..."
	}

	rl.SetWindowTitle(ui.windowTitle + " - " + fps + " FPS")
}
