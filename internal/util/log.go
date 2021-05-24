package util

import (
	"log"

	"github.com/rs/zerolog"
)

func Logger(logger zerolog.Logger, prefix string) *log.Logger {
	return log.New(logger, prefix+": ", 0)
}
