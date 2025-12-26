package cartridge

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cterence/gbgo/internal/log"
)

const (
	ROM_BANK_0_START = 0
	ROM_BANK_0_END   = 0x3FFF
	ROM_BANK_0_SIZE  = ROM_BANK_0_END - ROM_BANK_0_START + 1

	ROM_BANK_1_START = 0x4000
	ROM_BANK_1_END   = 0x7FFF
	ROM_BANK_1_SIZE  = ROM_BANK_1_END - ROM_BANK_1_START + 1

	ROM_BANK_SIZE = 0x4000

	EXTERNAL_RAM_START = 0xA000
	EXTERNAL_RAM_END   = 0xBFFF
	EXTERNAL_RAM_SIZE  = EXTERNAL_RAM_END - EXTERNAL_RAM_START + 1

	EXTERNAL_RAM_FLUSH_PERIOD = 5 * time.Second
)

type mbc uint8

const (
	NONE mbc = iota
	MBC1
	MBC2
	MMM01
	MBC3
	MBC5
	MBC6
	MBC7
)

type Cartridge struct {
	romPath          string
	stateDir         string
	romBanks         [][ROM_BANK_SIZE]uint8
	externalRAM      [][EXTERNAL_RAM_SIZE]uint8
	externalRAMMutex sync.Mutex

	romBankCount     uint16
	ramBankCount     uint8
	currentROMBank   uint8
	currentRAMBank   uint8
	externalRAMDirty bool

	mbc     mbc
	ram     bool
	battery bool
	timer   bool
	rumble  bool
	sensor  bool

	ramEnabled bool
}

func (c *Cartridge) Init(romPath, stateDir string, cartridgeType, romSize, ramSize uint8) error {
	c.romPath = romPath
	c.currentROMBank = 1
	c.currentRAMBank = 0

	c.configure(cartridgeType)

	c.romBankCount = 1 << (romSize + 1)

	if c.romBankCount <= 0 || c.romBankCount > 512 {
		return fmt.Errorf("unsupported bank count: %d", c.romBankCount)
	}

	c.romBanks = make([][ROM_BANK_SIZE]uint8, c.romBankCount)

	switch ramSize {
	case 0x0:
		c.ramBankCount = 0
	case 0x2:
		c.ramBankCount = 1
	case 0x3:
		c.ramBankCount = 4
	case 0x4:
		c.ramBankCount = 16
	case 0x5:
		c.ramBankCount = 8
	}

	c.externalRAM = make([][EXTERNAL_RAM_SIZE]uint8, c.ramBankCount)

	if c.battery {
		if err := c.loadExternalRam(); err != nil {
			return fmt.Errorf("failed to load external RAM: %w", err)
		}

		go func() {
			t := time.NewTicker(EXTERNAL_RAM_FLUSH_PERIOD)
			defer t.Stop()

			for range t.C {
				if c.externalRAMDirty {
					if err := c.flushExternalRam(); err != nil {
						log.Debug("[cartridge] failed to flush external RAM: %v", err)
					}

					c.externalRAMDirty = false
				}
			}
		}()
	}

	log.Debug("[cartridge] type: %d", cartridgeType)
	log.Debug("[cartridge] rom bank count: %d", c.romBankCount)
	log.Debug("[cartridge] ram bank count: %d", c.ramBankCount)

	return nil
}

func (c *Cartridge) Read(addr uint16) uint8 {
	switch {
	case addr <= ROM_BANK_0_END:
		return c.romBanks[0][addr]

	case addr >= ROM_BANK_1_START && addr <= ROM_BANK_1_END:
		bankAddr := addr % ROM_BANK_SIZE

		return c.romBanks[c.currentROMBank][bankAddr]

	case addr >= EXTERNAL_RAM_START && addr <= EXTERNAL_RAM_END:
		if c.ram && c.ramEnabled {
			return c.externalRAM[c.currentRAMBank][addr-EXTERNAL_RAM_START]
		}

		return 0xFF

	default:
		panic(fmt.Errorf("out of bounds cartridge read: %x", addr))
	}
}

func (c *Cartridge) Write(addr uint16, value uint8) {
	switch {
	// RAM enable
	case addr <= 0x1FFF:
		if value&0xF == 0xA {
			c.ramEnabled = true
		} else {
			c.ramEnabled = false
		}

		// ROM bank switch
	case addr >= 0x2000 && addr <= 0x3FFF:
		if value == 0 {
			value = 1
		}

		c.currentROMBank = value

	// RAM bank switch
	case addr >= 0x4000 && addr <= 0x5FFF:
		c.currentRAMBank = value

	case addr >= EXTERNAL_RAM_START && addr <= EXTERNAL_RAM_END:
		if c.ram && c.ramEnabled {
			c.externalRAMMutex.Lock()
			c.externalRAM[c.currentRAMBank][addr-EXTERNAL_RAM_START] = value
			c.externalRAMMutex.Unlock()

			if !c.externalRAMDirty {
				c.externalRAMDirty = true

				log.Debug("[cartridge] external ram is dirty")
			}
		}
	}
}

func (c *Cartridge) Load(byteIdx uint32, value uint8) {
	bankIndex := byteIdx / ROM_BANK_SIZE
	bankAddr := byteIdx % ROM_BANK_SIZE
	c.romBanks[bankIndex][bankAddr] = value
}

func (c *Cartridge) Close() {
	if c.battery {
		if err := c.flushExternalRam(); err != nil {
			fmt.Println(err)
		}
	}
}

func (c *Cartridge) loadExternalRam() error {
	savePath := strings.ReplaceAll(filepath.Base(c.romPath), filepath.Ext(c.romPath), ".sav")

	ramBytes, err := os.ReadFile(filepath.Join(c.stateDir, savePath))
	if err != nil {
		if !strings.Contains(err.Error(), "no such file or directory") {
			return fmt.Errorf("failed to read save file: %w", err)
		}

		return nil
	}

	for i := range c.ramBankCount {
		copy(c.externalRAM[i][:], ramBytes[int(i)*EXTERNAL_RAM_SIZE:int(i+1)*EXTERNAL_RAM_SIZE])
	}

	log.Debug("[cartridge] loaded external RAM from %s", savePath)

	return nil
}

func (c *Cartridge) flushExternalRam() error {
	savePath := strings.ReplaceAll(filepath.Base(c.romPath), filepath.Ext(c.romPath), ".sav")

	f, err := os.Create(filepath.Join(c.stateDir, savePath))
	if err != nil {
		return fmt.Errorf("failed to create save file: %w", err)
	}

	c.externalRAMMutex.Lock()

	ramBytes := make([]uint8, int(c.ramBankCount)*EXTERNAL_RAM_SIZE)

	for i := range c.ramBankCount {
		copy(ramBytes[int(i)*EXTERNAL_RAM_SIZE:int(i+1)*EXTERNAL_RAM_SIZE], c.externalRAM[i][:])
	}

	_, err = f.Write(ramBytes[:])
	if err != nil {
		return fmt.Errorf("failed to write to save file: %w", err)
	}

	c.externalRAMMutex.Unlock()

	log.Debug("[cartridge] flushed external RAM to %s", savePath)

	return nil
}

func (c *Cartridge) configure(cartridgeType uint8) {
	switch cartridgeType {
	case 0x0:
		c.mbc = NONE
	case 0x1:
		c.mbc = MBC1
	case 0x2:
		c.mbc = MBC1
		c.ram = true
	case 0x3:
		c.mbc = MBC1
		c.ram = true
		c.battery = true
	case 0x5:
		c.mbc = MBC2
	case 0x6:
		c.mbc = MBC2
		c.ram = true
	case 0x8:
		c.mbc = NONE
		c.ram = true
	case 0x9:
		c.mbc = NONE
		c.ram = true
		c.battery = true
	case 0xB:
		c.mbc = MMM01
	case 0xC:
		c.mbc = MMM01
		c.ram = true
	case 0xD:
		c.mbc = MMM01
		c.ram = true
		c.battery = true
	case 0xF:
		c.mbc = MBC3
		c.timer = true
		c.battery = true
	case 0x10:
		c.mbc = MBC3
		c.timer = true
		c.ram = true
		c.battery = true
	case 0x11:
		c.mbc = MBC3
	case 0x12:
		c.mbc = MBC3
		c.ram = true
	case 0x13:
		c.mbc = MBC3
		c.ram = true
		c.battery = true
	case 0x19:
		c.mbc = MBC5
	case 0x1A:
		c.mbc = MBC5
		c.ram = true
	case 0x1B:
		c.mbc = MBC5
		c.ram = true
		c.battery = true
	case 0x1C:
		c.mbc = MBC5
		c.rumble = true
	case 0x1D:
		c.mbc = MBC5
		c.ram = true
		c.rumble = true
	case 0x1E:
		c.mbc = MBC5
		c.ram = true
		c.rumble = true
		c.battery = true
	case 0x20:
		c.mbc = MBC6
	case 0x22:
		c.mbc = MBC7
		c.rumble = true
		c.ram = true
		c.battery = true
		c.sensor = true

	default:
		log.Debug("[cartridge] unsupported cartridge type: %x", cartridgeType)
	}
}
