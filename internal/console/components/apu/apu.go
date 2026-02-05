package apu

import (
	"fmt"

	"github.com/cterence/gbgo/internal/lib"
)

type APU struct {
	state
}

// TODO: decompose channels according to https://gbdev.io/pandocs/Audio_Registers.html

type state struct {
	// Channel 1
	NR10 uint8 // FF10
	NR11 uint8 // FF11
	NR12 uint8 // FF12
	NR13 uint8 // FF13
	NR14 uint8 // FF14

	// Channel 2
	NR21 uint8 // FF16
	NR22 uint8 // FF17
	NR23 uint8 // FF18
	NR24 uint8 // FF19

	// Channel 3
	NR30 uint8 // FF1A
	NR31 uint8 // FF1B
	NR32 uint8 // FF1C
	NR33 uint8 // FF1D
	NR34 uint8 // FF1E

	// Channel 4
	NR41 uint8 // FF20
	NR42 uint8 // FF21
	NR43 uint8 // FF22
	NR44 uint8 // FF23

	NR50 uint8 // FF24
	NR51 uint8 // FF25

	// NR52 : Audio master control
	AudioEnabled bool
	CH4          bool
	CH3          bool
	CH2          bool
	CH1          bool

	WaveRAM [16]uint8 // FF30 -> FF3F
}

func (a *APU) Init() {}

func (a *APU) Step(cycles int) {}

func (a *APU) Read(addr uint16) uint8 {
	switch addr {
	case 0xFF10:
		return a.NR10
	case 0xFF11:
		return a.NR11
	case 0xFF12:
		return a.NR12
	case 0xFF13:
		return a.NR13
	case 0xFF14:
		return a.NR14
	case 0xFF16:
		return a.NR21
	case 0xFF17:
		return a.NR22
	case 0xFF18:
		return a.NR23
	case 0xFF19:
		return a.NR24
	case 0xFF1A:
		return a.NR30
	case 0xFF1B:
		return a.NR31
	case 0xFF1C:
		return a.NR32
	case 0xFF1D:
		return a.NR33
	case 0xFF1E:
		return a.NR34
	case 0xFF20:
		return a.NR41
	case 0xFF21:
		return a.NR42
	case 0xFF22:
		return a.NR43
	case 0xFF23:
		return a.NR44
	case 0xFF24:
		return a.NR50
	case 0xFF25:
		return a.NR51
	case 0xFF26:
		var value uint8

		value |= lib.BToU8(a.AudioEnabled) << 7
		value |= lib.BToU8(a.CH4) << 3
		value |= lib.BToU8(a.CH3) << 2
		value |= lib.BToU8(a.CH2) << 1
		value |= lib.BToU8(a.CH1)

		return value
	case 0xFF30, 0xFF31, 0xFF32, 0xFF33, 0xFF34, 0xFF35, 0xFF36, 0xFF37, 0xFF38, 0xFF39, 0xFF3A, 0xFF3B, 0xFF3C, 0xFF3D, 0xFF3E, 0xFF3F:
		return a.WaveRAM[addr-0xFF30]
	default:
		panic(fmt.Errorf("unsupported read on apu: %x", addr))
	}
}

func (a *APU) Write(addr uint16, value uint8) {
	switch addr {
	case 0xFF10:
		a.NR10 = value
	case 0xFF11:
		a.NR11 = value
	case 0xFF12:
		a.NR12 = value
	case 0xFF13:
		a.NR13 = value
	case 0xFF14:
		a.NR14 = value
	case 0xFF16:
		a.NR21 = value
	case 0xFF17:
		a.NR22 = value
	case 0xFF18:
		a.NR23 = value
	case 0xFF19:
		a.NR24 = value
	case 0xFF1A:
		a.NR30 = value
	case 0xFF1B:
		a.NR31 = value
	case 0xFF1C:
		a.NR32 = value
	case 0xFF1D:
		a.NR33 = value
	case 0xFF1E:
		a.NR34 = value
	case 0xFF20:
		a.NR41 = value
	case 0xFF21:
		a.NR42 = value
	case 0xFF22:
		a.NR43 = value
	case 0xFF23:
		a.NR44 = value
	case 0xFF24:
		a.NR50 = value
	case 0xFF25:
		a.NR51 = value
	case 0xFF26:
		a.AudioEnabled = value&0x80 != 0
		a.CH4 = value&0x08 != 0
		a.CH3 = value&0x04 != 0
		a.CH2 = value&0x02 != 0
		a.CH1 = value&0x01 != 0

	case 0xFF30, 0xFF31, 0xFF32, 0xFF33, 0xFF34, 0xFF35, 0xFF36, 0xFF37, 0xFF38, 0xFF39, 0xFF3A, 0xFF3B, 0xFF3C, 0xFF3D, 0xFF3E, 0xFF3F:
		a.WaveRAM[addr-0xFF30] = value
	default:
		panic(fmt.Errorf("unsupported write on apu: %x", addr))
	}
}
