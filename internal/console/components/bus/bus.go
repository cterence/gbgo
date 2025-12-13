package bus

const (
	ROM_BANK_0_END = 0x3FFF
	ROM_BANK_1_END = 0x7FFF

	DIV  = 0xFF04
	TIMA = 0xFF05
	TMA  = 0xFF06
	TAC  = 0xFF07
	IFF  = 0xFF0F
	IE   = 0xFFFF
)

type cpu interface {
	Read(addr uint16) uint8
	Write(addr uint16, value uint8)
}

type timer interface {
	Read(addr uint16) uint8
	Write(addr uint16, value uint8)
}

type memory interface {
	Read(addr uint16) uint8
	Write(addr uint16, value uint8)
}

type cartridge interface {
	Read(addr uint16) uint8
	Write(addr uint16, value uint8)
}

type Bus struct {
	Memory    memory
	Cartridge cartridge
	CPU       cpu
	Timer     timer

	gbDoctor bool
}

type Option func(*Bus)

func WithGBDoctor(gbDoctor bool) Option {
	return func(b *Bus) {
		b.gbDoctor = gbDoctor
	}
}

func (b *Bus) Init(options ...Option) {
	for _, o := range options {
		o(b)
	}
}

func (b *Bus) Read(addr uint16) uint8 {
	if addr <= ROM_BANK_1_END {
		return b.Cartridge.Read(addr)
	}

	// FIXME: for gameboy doctor
	if addr == 0xFF44 && b.gbDoctor {
		return 0x90
	}

	if addr == DIV || addr == TIMA || addr == TMA || addr == TAC {
		return b.Timer.Read(addr)
	}

	if addr == IFF || addr == IE {
		return b.CPU.Read(addr)
	}

	return b.Memory.Read(addr)
}

func (b *Bus) Write(addr uint16, value uint8) {
	if addr <= ROM_BANK_1_END {
		b.Cartridge.Write(addr, value)

		return
	}

	if addr == DIV || addr == TIMA || addr == TMA || addr == TAC {
		b.Timer.Write(addr, value)

		return
	}

	if addr == IFF || addr == IE {
		b.CPU.Write(addr, value)

		return
	}

	b.Memory.Write(addr, value)
}
