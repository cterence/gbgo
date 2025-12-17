package serial

import "fmt"

const (
	SB = 0xFF01
	SC = 0xFF02

	// 4,194,304 / 8,192 = 512 CPU cycles per bit
	// 512 * 8 = 4,096 CPU cycles per byte transfer
	// It accounts for the hardware bit shifting mechanism
	SERIAL_CYCLES = 4096

	INTERRUPT_CODE = 0x8
)

type cpu interface {
	RequestInterrupt(code uint8)
}

type Serial struct {
	CPU cpu

	cycles int

	sb uint8
	sc uint8

	print bool
}

type Option func(*Serial)

func WithPrintSerial() Option {
	return func(s *Serial) {
		s.print = true
	}
}

func (s *Serial) Init(options ...Option) {
	s.sb = 0
	s.sc = 0

	for _, o := range options {
		o(s)
	}
}

func (s *Serial) Step(cycles int) {
	if s.sc&0x81 != 0x81 {
		return
	}

	s.cycles += cycles
	if s.cycles >= SERIAL_CYCLES {
		if s.print {
			fmt.Print(string(s.sb))
		}

		s.sb = 0xFF
		s.sc &= 0x7F
		s.cycles = 0
		s.CPU.RequestInterrupt(INTERRUPT_CODE)
	}
}

func (s *Serial) Read(addr uint16) uint8 {
	switch addr {
	case SB:
		return s.sb
	case SC:
		return s.sc
	default:
		panic(fmt.Errorf("unsupported read for serial: %x", addr))
	}
}

func (s *Serial) Write(addr uint16, value uint8) {
	switch addr {
	case SB:
		s.sb = value
	case SC:
		s.sc = value
	default:
		panic(fmt.Errorf("unsupported write for serial: %x", addr))
	}
}
