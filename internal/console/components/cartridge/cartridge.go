package cartridge

import (
	"errors"
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

	BANK_SIZE = 0x4000

	EXTERNAL_RAM_START = 0xA000
	EXTERNAL_RAM_END   = 0xBFFF
	EXTERNAL_RAM_SIZE  = EXTERNAL_RAM_END - EXTERNAL_RAM_START + 1

	EXTERNAL_RAM_FLUSH_PERIOD = 30 * time.Second
)

type Cartridge struct {
	romPath          string
	banks            [][BANK_SIZE]uint8
	currentBank      uint8
	externalRAM      [EXTERNAL_RAM_SIZE]uint8
	externalRAMDirty bool
	externalRAMMutex sync.Mutex
}

func (c *Cartridge) Init(romPath string, cartridgeType, romSize uint8) error {
	c.romPath = romPath
	c.currentBank = 1

	bankCount := 1 << (romSize + 1)

	if bankCount <= 0 || bankCount > 512 {
		return fmt.Errorf("unsupported bank count: %d", bankCount)
	}

	c.banks = make([][BANK_SIZE]uint8, bankCount)

	log.Debug("[cartridge] type: %d", cartridgeType)
	log.Debug("[cartridge] bank count: %d", bankCount)

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

	return nil
}

func (c *Cartridge) Read(addr uint16) uint8 {
	switch {
	case addr <= ROM_BANK_0_END:
		return c.banks[0][addr]

	case addr >= ROM_BANK_1_START && addr <= ROM_BANK_1_END:
		bankAddr := addr % BANK_SIZE

		return c.banks[c.currentBank][bankAddr]

	case addr >= EXTERNAL_RAM_START && addr <= EXTERNAL_RAM_END:
		return c.externalRAM[addr-EXTERNAL_RAM_START]

	default:
		panic(fmt.Errorf("out of bounds cartridge read: %x", addr))
	}
}

func (c *Cartridge) Write(addr uint16, value uint8) {
	switch {
	case addr >= EXTERNAL_RAM_START && addr <= EXTERNAL_RAM_END:
		c.externalRAMMutex.Lock()
		c.externalRAM[addr-EXTERNAL_RAM_START] = value
		c.externalRAMMutex.Unlock()

		if !c.externalRAMDirty {
			c.externalRAMDirty = true

			log.Debug("[cartridge] external ram is dirty")
		}

	case addr >= 0x2000 && addr <= 0x3FFF:
		if value == 0 {
			value = 1
		}

		c.currentBank = value
	}
}

func (c *Cartridge) Load(byteIdx uint32, value uint8) {
	bankIndex := byteIdx / BANK_SIZE
	bankAddr := byteIdx % BANK_SIZE
	c.banks[bankIndex][bankAddr] = value
}

func (c *Cartridge) Close() {
	if err := c.flushExternalRam(); err != nil {
		fmt.Println(err)
	}
}

func (c *Cartridge) loadExternalRam() error {
	savePath := strings.ReplaceAll(c.romPath, filepath.Ext(c.romPath), ".sav")

	externalRAM, err := os.ReadFile(savePath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("failed to read save file: %w", err)
		}

		return nil
	}

	if len(externalRAM) > EXTERNAL_RAM_SIZE {
		fmt.Printf("[cartridge] failed to load external RAM from %s: expected size: %d actual size: %d", savePath, EXTERNAL_RAM_SIZE, len(externalRAM))

		return nil
	}

	copy(c.externalRAM[:], externalRAM)

	log.Debug("[cartridge] loaded external RAM from %s", savePath)

	return nil
}

func (c *Cartridge) flushExternalRam() error {
	savePath := strings.ReplaceAll(c.romPath, filepath.Ext(c.romPath), ".sav")

	f, err := os.Create(savePath)
	if err != nil {
		return fmt.Errorf("failed to create save file: %w", err)
	}

	c.externalRAMMutex.Lock()

	_, err = f.Write(c.externalRAM[:])
	if err != nil {
		return fmt.Errorf("failed to write to save file: %w", err)
	}

	c.externalRAMMutex.Unlock()

	log.Debug("[cartridge] flushed external RAM to %s", savePath)

	return nil
}
