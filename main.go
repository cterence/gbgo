package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"

	"github.com/cterence/gbgo/internal/console"
	"github.com/urfave/cli/v3"
)

func main() {
	var (
		stopCPUAfter int
	)

	cmd := &cli.Command{
		Name:  "gbgo",
		Usage: "gameboy emulator",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:        "stop-cpu-after",
				Aliases:     []string{"s"},
				Usage:       "stop CPU execution after N cycles",
				Destination: &stopCPUAfter,
			},

			&cli.BoolFlag{
				Name:    "pprof",
				Aliases: []string{"p"},
				Usage:   "run pprof webserver on localhost:6060",
				Action: func(_ context.Context, _ *cli.Command, _ bool) error {
					go func() {
						log.Println(http.ListenAndServe("localhost:6060", nil))
					}()

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

			return console.Run(ctx, romBytes, console.WithStopCPUAfter(stopCPUAfter))
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
