package localserver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/goproxy/goproxy"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

type multicacher struct {
	cachers []goproxy.Cacher
}

var _ goproxy.Cacher = &multicacher{}

func newMulticacher() *multicacher {
	return &multicacher{}
}

func (m *multicacher) Add(cachers ...goproxy.Cacher) {
	// TODO only add new?
	m.cachers = append(m.cachers, cachers...)
}

func (m *multicacher) Get(ctx context.Context, name string) (io.ReadCloser, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	g, ctx := errgroup.WithContext(ctx)
	var (
		mutex     sync.Mutex
		result    io.ReadCloser
		resultErr error
	)
	for iI, cI := range m.cachers {
		i, c := iI, cI // capture
		g.Go(func() error {
			rc, err := c.Get(ctx, name)
			if err != nil || rc == nil {
				log.Debug().Int("number", i).Err(err).Msg("multicacher miss")
				return nil
			}

			select {
			case <-ctx.Done():
				return nil
			default:
			}
			mutex.Lock()
			defer mutex.Unlock()
			log.Info().Int("number", i).Msg("multicacher hit")
			result = rc
			cancel()
			return nil
		})
	}
	err := g.Wait()
	log.Info().AnErr("errgroup", err).AnErr("resultErr", resultErr).Bool("result", result != nil).Msg("multicacher done")

	if err != nil && !errors.Is(err, context.Canceled) {
		log.Trace().Err(err).Msg("multicacher bailout")
		return nil, err
	}

	if result == nil {
		resultErr = fmt.Errorf("not found %s", name)
		log.Trace().Err(resultErr).Msg("multicacher bailout2")
	}
	return result, resultErr
}

func (m *multicacher) Set(ctx context.Context, name string, content io.ReadSeeker) error {
	// NOOP
	return nil
}
