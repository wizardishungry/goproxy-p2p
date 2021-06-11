package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
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
	// TODO add -kill & -restart & -daemon
	flag.Parse()

	var (
		myAgent  *agent.Agent
		myConfig *agent.Config
		err      error
	)

	if *eval {
		myAgent, myConfig, err = agent.ConnectOrNew(ctx, true)
		if err == nil {
			goto EXISTING_DAEMON
		}
		args := make([]string, 0, len(os.Args[1:]))
		oldArgs := os.Args[1:]
		for i := 0; i < len(oldArgs); i++ {
			if oldArgs[i] == "-eval" {
				continue
			}
			args = append(args, oldArgs[i])
		}
		// TODO: add daemon flag
		fmt.Println(args)
		// args = []string{"sleep", "60"}
		fmt.Println(args)

		cmd := exec.Command(os.Args[0], args...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stdout // TODO attach to log in child
		cmd.SysProcAttr = &syscall.SysProcAttr{
			// Setpgid: true,
			// Pgid:    0,
			Setsid: true,
			// Noctty: true,
		}
		// TODO: pass log fds?

		err := cmd.Start()
		if err != nil {
			log.Fatal().Err(err).Msgf("Can't start detached process")
		}
		if err := cmd.Process.Release(); err != nil {
			log.Fatal().Err(err).Msgf("Can't release")
		}
		time.Sleep(time.Second) // TODO retry instead
	}

	myAgent, myConfig, err = agent.ConnectOrNew(ctx, *eval) // TODO: eval should only connect
	if err != nil {
		fmt.Println(err)
		log.Fatal().Msgf("agent.ConnectOrNew: %v", err)
	}
EXISTING_DAEMON:
	if myConfig != nil {
		if *eval {
			myConfig.ShellEval()
		} else {
			log.Info().Str("GOPROXY", myConfig.GOPROXY).Msgf("config received")
		}
		return
	}

	if myAgent == nil {
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

	myConfig = i.GetConfig()
	myAgent.SetConfig(myConfig)
	fmt.Println(os.Args)
	log.Info().Str("pass", *pass).Msgf("deee")

	l := zerolog.Ctx(ctx)
	l.UpdateContext(func(c zerolog.Context) zerolog.Context {
		return c.Int("port", *port)
	})

	if err := i.Start(ctx); err != nil {
		log.Fatal().Msgf("Service.Start: %v", err)
	}

	if *eval {
		myConfig.ShellEval()
	}

	<-ctx.Done()
	if err := ctx.Err(); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatal().Msgf("Service error: %v", err)
	}
}
