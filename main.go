package main

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"

	"github.com/cterence/gbgo/internal/console"
	"github.com/cterence/gbgo/internal/log"
	"github.com/urfave/cli/v3"
)

func main() {
	var opts []console.Option

	cmd := &cli.Command{
		Name:  "gbgo",
		Usage: "gameboy emulator",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "pprof",
				Aliases: []string{"p"},
				Usage:   "run pprof webserver on localhost:6060",
				Action: func(_ context.Context, _ *cli.Command, _ bool) error {
					go func() {
						fmt.Println(http.ListenAndServe("localhost:6060", nil))
					}()

					return nil
				},
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

			&cli.BoolFlag{
				Name:    "boot",
				Aliases: []string{"b"},
				Usage:   "use bootrom",
				Action: func(_ context.Context, _ *cli.Command, b bool) error {
					opts = append(opts, console.WithBootROM())

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

			return console.Run(ctx, romBytes, opts...)
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
