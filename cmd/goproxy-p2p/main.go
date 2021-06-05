package main

import (
	"context"
	"errors"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"jonwillia.ms/goproxy-p2p/internal/service"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill, syscall.SIGTERM)
	defer cancel()
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	port := flag.Int("port", 8080, "listen on localhost port")
	flag.Parse()

	i := service.Instance{
		LocalPort: *port,
	}

	l := zerolog.Ctx(ctx)
	l.UpdateContext(func(c zerolog.Context) zerolog.Context {
		return c.Int("port", *port)
	})

	if err := i.Start(ctx); err != nil {
		log.Fatal().Msgf("Service.Start: %v", err)
	}

	<-ctx.Done()
	if err := ctx.Err(); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatal().Msgf("Service error: %v", err)
	}
}
