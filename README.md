# gbgo

A Game Boy emulator written in Golang.

## TODO

- [x] Pause / Resume
- [x] Turbo / Slowmo modes
- [x] PPU window
- [x] Support MBC
- [x] Pass dmg-acid2 test
- [x] Frame FIFO
- [x] External RAM save
- [x] Pixel FIFO
- [ ] Only save external RAM for cartridge with batteries
- [ ] Serializable interface for state save

  ```go
  type Serializable interface {
    Save(*bytes.Buffer)
    Load(*bytes.Reader)
  }
  ```

- [ ] Debug overlay
- [ ] Runtime assertions
- [ ] Trace ring buffer
- [ ] CPU debug to file with goroutines
- [ ] Save states
- [ ] APU
- [ ] Release cross-platform binaries (goreleaser)
- [ ] Pass mooneye tests
