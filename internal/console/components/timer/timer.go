package timer

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"github.com/cterence/gbgo/internal/lib"
)

const (
	DIV  = 0xFF04
	TIMA = 0xFF05
	TMA  = 0xFF06
	TAC  = 0xFF07

	CPU_FREQ = 4194304

	INTERRUPT_CODE = 0x4
)

type CPU interface {
	RequestInterrupt(code uint8)
}

type Timer struct {
	cpu CPU
	state
}

type state struct {
	TIMACPUCycles int

	DIV  uint16 // 0xFF04
	TIMA uint8  // 0xFF05
	TMA  uint8  // 0xFF06
	TAC  uint8  // 0xFF07
}

var timaFreqs = []int{1024, 16, 64, 256}

func (t *Timer) Init(cpu CPU) {
	t.cpu = cpu
	t.TIMACPUCycles = 0
	t.DIV = 0
	t.TIMA = 0
	t.TMA = 0
	t.TAC = 0
}

func (t *Timer) Step(cycles int) {
	t.DIV += uint16(cycles)

	if t.TAC&0x4 != 0x4 {
		return
	}

	t.TIMACPUCycles += cycles
	timaFreq := t.TAC & 0x3

	if t.TIMACPUCycles >= timaFreqs[timaFreq] {
		t.TIMA++
		t.TIMACPUCycles -= timaFreqs[timaFreq]

		if t.TIMA == 0 {
			t.TIMA = t.TMA
			t.cpu.RequestInterrupt(INTERRUPT_CODE)
		}
	}
}

func (t *Timer) Read(addr uint16) uint8 {
	switch addr {
	case DIV:
		return uint8(t.DIV >> 8) // Upper nibble increments every 256, so we have the 16384Hz frequency desired
	case TIMA:
		return t.TIMA
	case TMA:
		return t.TMA
	case TAC:
		return t.TAC
	default:
		panic(fmt.Errorf("unsupported read on timer: %x", addr))
	}
}

func (t *Timer) Write(addr uint16, value uint8) {
	switch addr {
	case DIV:
		t.DIV = 0
	case TIMA:
		t.TIMA = value
	case TMA:
		t.TMA = value
	case TAC:
		t.TAC = value
	default:
		panic(fmt.Errorf("unsupported write on timer: %x", addr))
	}
}

func (t *Timer) Load(buf *bytes.Reader) {
	enc := gob.NewDecoder(buf)
	err := enc.Decode(&t.state)

	lib.Assert(err == nil, "failed to decode state: %v", err)
}

func (t *Timer) Save(buf *bytes.Buffer) {
	enc := gob.NewEncoder(buf)
	err := enc.Encode(t.state)

	lib.Assert(err == nil, "failed to encode state: %v", err)
}
