package util

import (
	"bytes"
	"os/exec"
	"strings"

	"github.com/goproxy/goproxy"
	"github.com/rs/zerolog/log"
)

func Gomodcacher() goproxy.Cacher {
	gomodcache, err := runCmd("go", "env", "GOMODCACHE")
	if err != nil {
		log.Fatal().Err(err).Msg("GOMODCACHE")
	}
	dir := gomodcache + "/cache/download"
	log.Debug().Str("path", dir).Msg("path to cache")

	return goproxy.DirCacher(dir)
}

func runCmd(name string, arg ...string) (string, error) {
	cmd := exec.Command(name, arg...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), err
}
