package serial

import "fmt"

const (
	SB = 0xFF01
	SC = 0xFF02

	SERIAL_CYCLES = 8192

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

func WithPrintSerial(printSerial bool) Option {
	return func(s *Serial) {
		s.print = printSerial
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
	s.cycles += cycles

	if s.cycles >= SERIAL_CYCLES {
		if s.sc&0x80 != 0 {
			if s.print {
				fmt.Print(string(s.sb))
			}

			s.sc &= 0x7F
			s.CPU.RequestInterrupt(INTERRUPT_CODE)
		}

		s.cycles -= SERIAL_CYCLES
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
