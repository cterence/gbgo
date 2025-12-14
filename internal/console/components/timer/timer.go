package timer

import "fmt"

const (
	DIV  = 0xFF04
	TIMA = 0xFF05
	TMA  = 0xFF06
	TAC  = 0xFF07

	CPU_FREQ = 4194304

	INTERRUPT_CODE = 0x4
)

type cpu interface {
	RequestInterrupt(code uint8)
}

type Timer struct {
	CPU cpu

	timaCPUCycles int

	div  uint16 // 0xFF04
	tima uint8  // 0xFF05
	tma  uint8  // 0xFF06
	tac  uint8  // 0xFF07
}

var timaFreqs = []int{1024, 16, 64, 256}

func (t *Timer) Init() {
	t.timaCPUCycles = 0
	t.div = 0
	t.tima = 0
	t.tma = 0
	t.tac = 0
}

func (t *Timer) Step(cycles int) {
	t.div += uint16(cycles)

	if t.tac&0x4 != 0x4 {
		return
	}

	t.timaCPUCycles += cycles
	timaFreq := t.tac & 0x3

	if t.timaCPUCycles >= timaFreqs[timaFreq] {
		t.tima++
		t.timaCPUCycles -= timaFreqs[timaFreq]

		if t.tima == 0 {
			t.tima = t.tma
			t.CPU.RequestInterrupt(INTERRUPT_CODE)
		}
	}
}

func (t *Timer) Read(addr uint16) uint8 {
	if addr == DIV {
		return uint8(t.div >> 8) // Upper nibble increments every 256, so we have the 16384Hz frequency desired
	}

	if addr == TIMA {
		return t.tima
	}

	if addr == TMA {
		return t.tma
	}

	if addr == TAC {
		return t.tac
	}

	panic(fmt.Errorf("unsupported read on timer: %x", addr))
}

func (t *Timer) Write(addr uint16, value uint8) {
	if addr == DIV {
		t.div = 0
		return
	}

	if addr == TIMA {
		t.tima = value
		return
	}

	if addr == TMA {
		t.tma = value
		return
	}

	if addr == TAC {
		t.tac = value
		return
	}

	panic(fmt.Errorf("unsupported write on timer: %x", addr))
}
