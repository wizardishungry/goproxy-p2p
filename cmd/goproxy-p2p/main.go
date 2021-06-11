package main

import (
	"context"
	"errors"
	"flag"
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

const (
	passwordLength = 20
)

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
	serve := flag.Bool("serve", true, "serve your local go cache to other users with your ssh keys")
	eval := flag.Bool("eval", false, "shell eval")
	flag.Parse()

	if *eval {
		args := os.Args

		proc, err := os.StartProcess(args[0], args[1:], &os.ProcAttr{
			Dir:   ".",
			Env:   os.Environ(),
			Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
			Sys:   &syscall.SysProcAttr{Setpgid: true, Pgid: 0, Noctty: true},
		})
		if err != nil {
			log.Fatal().Err(err).Msgf("Can't detach")
		}
		if err := proc.Release(); err != nil {
			log.Fatal().Err(err).Msgf("Can't release")
		}
		time.Sleep(time.Second) // TODO retry instead
	}

	a, c, err := agent.ConnectOrNew(ctx, *eval)
	if err != nil {
		log.Fatal().Msgf("agent.ConnectOrNew: %v", err)
	}
	if c != nil {
		if *eval {
			c.ShellEval()
		} else {
			log.Info().Str("GOPROXY", c.GOPROXY).Msgf("config received")
		}
		return
	}

	if a == nil {
		log.Fatal().Msgf("no running agent")
	}

	if *pass == "" {
		*pass = util.RandString(passwordLength)
	}

	i := service.Instance{
		LocalPort: *port,
		Password:  *pass,
		Serve:     *serve,
	}

	c = i.GetConfig()
	a.SetConfig(c)

	l := zerolog.Ctx(ctx)
	l.UpdateContext(func(c zerolog.Context) zerolog.Context {
		return c.Int("port", *port)
	})

	if err := i.Start(ctx); err != nil {
		log.Fatal().Msgf("Service.Start: %v", err)
	}

	if *eval {
		c.ShellEval()
	}

	<-ctx.Done()
	if err := ctx.Err(); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatal().Msgf("Service error: %v", err)
	}
}
