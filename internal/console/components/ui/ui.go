package ui

import (
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cterence/gbgo/internal/log"
	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	WIDTH         = 160
	HEIGHT        = 144
	INITIAL_SCALE = 4
	FPS           = 60
	AXIS_TRIGGER  = 0.5
)

type console interface {
	Shutdown()
	Pause()
	Reset()
}

type joypad interface {
	UpdateButtons(a, b, right, left, up, down, selectB, start bool)
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
	// Emulator buttons
	TURBO
	PAUSE
	SLOWMO
	RESET
)

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
	// RESET
	{
		keyboardKeys:   []int32{rl.KeyTab},
		gamepadButtons: []int32{},
	},
}

type UI struct {
	Console console
	Joypad  joypad

	windowTitle string
	pixels      []rl.Color

	frames  uint64
	texture rl.Texture2D

	currentFPS int32

	paused bool
}

var palette = [4]rl.Color{
	{A: 0xFF, R: 0xFF, G: 0xFF, B: 0xFF},
	{A: 0xFF, R: 0xAA, G: 0xAA, B: 0xAA},
	{A: 0xFF, R: 0x55, G: 0x55, B: 0x55},
	{A: 0xFF, R: 0x00, G: 0x00, B: 0x00},
}

// TODO: better system for choosing controller
var gamepad = int32(1)

func (ui *UI) Init(romPath string) {
	romFile := filepath.Base(romPath)
	romTitle := strings.ReplaceAll(romFile, filepath.Ext(romFile), "")
	ui.windowTitle = "gbgo - " + romTitle

	if ui.frames == 0 {
		rl.SetTraceLogLevel(rl.LogError)
		rl.SetConfigFlags(rl.FlagWindowResizable | rl.FlagWindowHighdpi)
		rl.InitWindow(WIDTH*INITIAL_SCALE, HEIGHT*INITIAL_SCALE, ui.windowTitle)
		rl.SetTargetFPS(FPS)
		rl.HideCursor()

		ui.texture = rl.LoadTextureFromImage(rl.GenImageColor(WIDTH, HEIGHT, rl.Black))
		rl.SetTextureFilter(ui.texture, rl.FilterPoint)
	}

	// Must use make to properly initialize array for CGo calls
	ui.pixels = make([]rl.Color, WIDTH*HEIGHT)
}

func (ui *UI) Step(cycles int) {
	ui.handleEvents()

	ui.frames++
}

func (ui *UI) DrawFrameBuffer(frameBuffer [WIDTH][HEIGHT]uint8) {
	// Return if ui was not initialized
	if len(ui.pixels) == 0 {
		return
	}

	for y := range HEIGHT {
		for x := range WIDTH {
			color := palette[frameBuffer[x][y]]
			ui.pixels[y*WIDTH+x] = color
		}
	}

	rl.UpdateTexture(ui.texture, ui.pixels[:])

	screenW := float32(rl.GetScreenWidth())
	screenH := float32(rl.GetScreenHeight())
	currentScale := min(screenW/WIDTH, screenH/HEIGHT)

	src := rl.Rectangle{
		X:      0,
		Y:      0,
		Width:  WIDTH,
		Height: HEIGHT,
	}

	dst := rl.Rectangle{
		X:      (screenW - WIDTH*currentScale) / 2,
		Y:      (screenH - HEIGHT*currentScale) / 2,
		Width:  WIDTH * currentScale,
		Height: HEIGHT * currentScale,
	}

	rl.BeginDrawing()
	rl.DrawTexturePro(ui.texture, src, dst, rl.Vector2{}, 0, rl.White)
	rl.EndDrawing()
}

func (ui *UI) handleEvents() {
	if rl.WindowShouldClose() {
		ui.Console.Shutdown()

		return
	}

	ui.updateButtonsState()

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

	if buttons[RESET].justPressed {
		log.Debug("[ui] reset")
		ui.Console.Reset()
	}

	ui.currentFPS = rl.GetFPS()

	// Update FPS in title every second
	if !ui.paused && ui.frames%uint64(fpsTarget) == 0 {
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

	right := buttons[RIGHT].currentlyPressed
	a := buttons[A].currentlyPressed
	left := buttons[LEFT].currentlyPressed
	b := buttons[B].currentlyPressed
	selectB := buttons[SELECT].currentlyPressed
	up := buttons[UP].currentlyPressed
	start := buttons[START].currentlyPressed
	down := buttons[DOWN].currentlyPressed

	ui.Joypad.UpdateButtons(a, b, right, left, up, down, selectB, start)
}

func (ui *UI) updateTitleFPS() {
	fps := strconv.FormatInt(int64(ui.currentFPS), 10)

	// Don't show first FPS measurement as it's imprecise
	if ui.frames == 0 {
		fps = "..."
	}

	rl.SetWindowTitle(ui.windowTitle + " - " + fps + " FPS")
}
