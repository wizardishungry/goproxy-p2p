package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"jonwillia.ms/goproxy-p2p/internal/agent"
	"jonwillia.ms/goproxy-p2p/internal/service"
	"jonwillia.ms/goproxy-p2p/internal/util"
)

const passwordLength = 20

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill, syscall.SIGTERM)
	defer cancel()
	{
		l := zerolog.ConsoleWriter{Out: os.Stderr}
		l.TimeFormat = time.ANSIC
		log.Logger = log.Output(l)
	}

	port := flag.Int("port", 8080, "listen on localhost port")
	pass := flag.String("password", "", "password for local service")
	flag.Parse()

	a, c, err := agent.ConnectOrNew(ctx)
	if err != nil {
		log.Fatal().Msgf("agent.ConnectOrNew: %v", err)
	}
	if c != nil {
		log.Info().Str("GOPROXY", c.GOPROXY)
		return
	}
	if a == nil {
		log.Fatal().Msgf("no agent")
	}

	if *pass == "" {
		*pass = util.RandString(passwordLength)
	}

	i := service.Instance{
		LocalPort: *port,
		Password:  *pass,
	}

	c = i.GetConfig()
	a.SetConfig(c)

	fmt.Printf("export GOPROXY=%s\n", c.GOPROXY)

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
