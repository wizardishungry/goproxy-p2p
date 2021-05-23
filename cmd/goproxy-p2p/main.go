package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"jonwillia.ms/goproxy-p2p/internal/service"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill, syscall.SIGTERM)
	defer cancel()

	i := service.Instance{
		LocalPort: 8080,
	}

	if err := i.Start(ctx); err != nil {
		log.Fatal().Msgf("Service.Start: %v", err)
	}

	<-ctx.Done()
	if err := ctx.Err(); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatal().Msgf("Service error: %v", err)
	}
}
