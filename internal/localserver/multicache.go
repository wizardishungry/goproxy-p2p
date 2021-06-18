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
		resultIdx int = -1
	)
	for iI, cI := range m.cachers {
		i, c := iI, cI // capture
		g.Go(func() error {
			log.Trace().Int("number", i).Msg("multicacher req")
			rc, err := c.Get(ctx, name)
			if err != nil || rc == nil {
				if err != nil && errors.Is(err, context.Canceled) {
					log.Debug().Int("number", i).Err(err).Msg("multicacher miss")
					mutex.Lock()
					defer mutex.Unlock()
					resultErr = err
				}
				return nil
			}

			select {
			case <-ctx.Done():
				return nil
			default:
			}
			mutex.Lock()
			defer mutex.Unlock()
			var cacherName string
			if s, ok := m.cachers[i].(fmt.Stringer); ok {
				cacherName = s.String()
			} else {
				cacherName = fmt.Sprintf("%T", m.cachers[i])
			}
			log.Trace().Str("name", cacherName).Int("number", i).Msg("multicacher hit")
			result = rc
			resultIdx = i
			cancel()
			return nil
		})
	}
	err := g.Wait()

	if err != nil && !errors.Is(err, context.Canceled) {
		return nil, err
	}

	if result == nil {
		resultErr = fmt.Errorf("not found %s: %w", name, resultErr)
	} else {
		resultErr = nil
		m.reshuffle(resultIdx)
	}
	return result, resultErr
}

func (m *multicacher) Set(ctx context.Context, name string, content io.ReadSeeker) error {
	if len(m.cachers) < 1 {
		return fmt.Errorf("no cachers")
	}
	return m.cachers[0].Set(ctx, name, content)
}

// reshuffle moves the succeeding index to the second slot
func (m *multicacher) reshuffle(idx int) {
	if idx <= 0 || idx >= len(m.cachers) {
		return
	}

	// naive
	m.cachers[1], m.cachers[idx] = m.cachers[idx], m.cachers[1]
}
