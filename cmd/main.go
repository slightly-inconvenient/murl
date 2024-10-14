package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/slightly-inconvenient/murl"
	"github.com/slightly-inconvenient/murl/internal/config"
	"github.com/slightly-inconvenient/murl/internal/server"
)

func main() {
	os.Exit(run())
}

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()

	flags := createFlags()
	if err := flags.Parse(os.Args[1:]); err != nil {
		return handleError(err)
	}

	config, routes, err := config.ParseConfigFile(flags.Lookup("config").Value.String())
	if err != nil {
		return handleError(err)
	}

	if err := server.Run(ctx, config, murl.NewMux(routes)); err != nil {
		return handleError(err)
	}

	return 0
}

func createFlags() *flag.FlagSet {
	flags := flag.NewFlagSet("murl", flag.ExitOnError)
	flags.Usage = func() {
		flags.PrintDefaults()
	}
	flags.String("config", "config.json", "Path to the configuration file")
	return flags
}

func handleError(err error) int {
	fmt.Println(err)
	return 1
}
