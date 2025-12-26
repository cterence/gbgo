package main

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/pprof"

	"github.com/cterence/gbgo/internal/console"
	"github.com/cterence/gbgo/internal/log"
	"github.com/urfave/cli/v3"
)

const (
	PPROF_FILE = "./profile.tar.gz"
)

func main() {
	var opts []console.Option

	var (
		runPProf bool
	)

	cmd := &cli.Command{
		Name:  "gbgo",
		Usage: "gameboy emulator",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "pprof",
				Aliases:     []string{"p"},
				Usage:       "create pprof file on exit at " + PPROF_FILE,
				Destination: &runPProf,
			},

			&cli.BoolFlag{
				Name:    "debug",
				Aliases: []string{"d"},
				Usage:   "print emulator debug logs",
				Action: func(_ context.Context, _ *cli.Command, b bool) error {
					log.DebugEnabled = b

					return nil
				},
			},

			&cli.StringFlag{
				Name:      "boot",
				Aliases:   []string{"b"},
				Usage:     "path to boot rom file",
				TakesFile: true,
				Action: func(_ context.Context, _ *cli.Command, bootRomPath string) error {
					bootRom, err := os.ReadFile(bootRomPath)
					if err != nil {
						return fmt.Errorf("failed to read boot rom file: %w", err)
					}

					if len(bootRom) == 0 {
						return errors.New("boot rom is empty")
					}

					opts = append(opts, console.WithBootROM(bootRom))

					return nil
				},
			},

			&cli.BoolFlag{
				Name:    "print-serial",
				Aliases: []string{"ps"},
				Usage:   "print serial output to console",
				Action: func(_ context.Context, _ *cli.Command, b bool) error {
					opts = append(opts, console.WithPrintSerial())

					return nil
				},
			},

			&cli.BoolFlag{
				Name:    "no-state",
				Aliases: []string{"ns"},
				Usage:   "do not load state file",
				Action: func(_ context.Context, _ *cli.Command, b bool) error {
					opts = append(opts, console.WithNoState())

					return nil
				},
			},

			&cli.BoolFlag{
				Name:    "headless",
				Aliases: []string{"hl"},
				Usage:   "run without UI",
				Action: func(_ context.Context, _ *cli.Command, b bool) error {
					opts = append(opts, console.WithHeadless())

					return nil
				},
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if runPProf {
				f, err := os.Create(PPROF_FILE)
				if err != nil {
					fmt.Printf("failed to create pprof file: %v\n", err)
				}

				defer func() {
					if err := f.Close(); err != nil {
						fmt.Printf("failed to close pprof file: %v\n", err)
					}
				}()

				if err := pprof.StartCPUProfile(f); err != nil {
					return fmt.Errorf("failed to start CPU profile: %w", err)
				}

				defer pprof.StopCPUProfile()
			}

			romPath := cmd.Args().First()

			if romPath == "" {
				fmt.Printf("error: no rom path given\n\n")
				return cli.ShowSubcommandHelp(cmd)
			}

			romBytes, err := os.ReadFile(romPath)
			if err != nil {
				return err
			}

			if filepath.Ext(romPath) == ".zip" {
				bytesReader := bytes.NewReader(romBytes)

				r, err := zip.NewReader(bytesReader, int64(len(romBytes)))
				if err != nil {
					return fmt.Errorf("failed to create zip reader: %w", err)
				}

				for _, f := range r.File {
					if filepath.Ext(f.Name) == ".gb" {
						rc, err := f.Open()
						if err != nil {
							return fmt.Errorf("failed to open file %s in zip archive: %w", f.Name, err)
						}

						romBytes, err = io.ReadAll(rc)
						if err != nil {
							return fmt.Errorf("failed to read file %s bytes: %w", f.Name, err)
						}

						log.Debug("[main] read file %s in archive", f.Name)

						break // Only read one .gb file
					}
				}
			}

			homeDir, err := os.UserHomeDir()
			if err != nil {
				return err
			}

			stateDir := filepath.Join(homeDir, ".local", "share", "gbgo")

			err = os.MkdirAll(stateDir, 0755)
			if err != nil {
				return err
			}

			if err := console.Run(ctx, romBytes, romPath, stateDir, opts...); err != nil {
				return err
			}

			return nil
		},
		Commands: []*cli.Command{
			{
				Name:    "disassemble",
				Aliases: []string{"d"},
				Usage:   "disassemble a rom",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					romPath := cmd.Args().First()

					if romPath == "" {
						fmt.Printf("error: no rom path given\n\n")
						return cli.ShowSubcommandHelp(cmd)
					}

					romBytes, err := os.ReadFile(romPath)
					if err != nil {
						return err
					}

					return console.Disassemble(romBytes)
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Printf("runtime error: %v\n", err)
		os.Exit(1)
	}
}
