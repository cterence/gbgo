package joypad

import (
	"fmt"
)

const (
	JOYPAD         = 0xFF00
	INTERRUPT_CODE = 0x10
)

type cpu interface {
	RequestInterrupt(code uint8)
}

type Joypad struct {
	CPU    cpu
	joypad uint8

	a       bool
	b       bool
	right   bool
	left    bool
	up      bool
	down    bool
	start   bool
	selectB bool
}

func (j *Joypad) Init() {
	j.joypad = 0xCF
}

func (j *Joypad) Read(addr uint16) uint8 {
	switch addr {
	case JOYPAD:
		result := uint8(0xCF) | j.joypad

		if j.joypad&0x10 == 0 {
			if j.right {
				result &^= 0x1
			}

			if j.left {
				result &^= 0x2
			}

			if j.up {
				result &^= 0x4
			}

			if j.down {
				result &^= 0x8
			}
		}

		if j.joypad&0x20 == 0 {
			if j.a {
				result &^= 0x1
			}

			if j.b {
				result &^= 0x2
			}

			if j.selectB {
				result &^= 0x4
			}

			if j.start {
				result &^= 0x8
			}
		}

		return result
	default:
		panic(fmt.Errorf("unsupported read for joypad: %x", addr))
	}
}

func (j *Joypad) Write(addr uint16, value uint8) {
	switch addr {
	case JOYPAD:
		j.joypad = value & 0x30
	default:
		panic(fmt.Errorf("unsupported write for joypad: %x", addr))
	}
}

func (j *Joypad) UpdateButtons(a, b, right, left, up, down, selectB, start bool) {
	if (a && !j.a) || (b && !j.b) || (right && !j.right) || (up && !j.up) || (down && !j.down) || (left && !j.left) || (start && !j.start) || (selectB && !j.selectB) {
		j.CPU.RequestInterrupt(INTERRUPT_CODE)
	}

	j.a = a
	j.b = b
	j.right = right
	j.left = left
	j.up = up
	j.down = down
	j.start = start
	j.selectB = selectB
}
