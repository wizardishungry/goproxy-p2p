package locator

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"runtime/debug"
	"sync"

	"github.com/grandcat/zeroconf"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

var service = func() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		log.Fatal().Msg("debug.ReadBuildInfo")
	}
	main := bi.Main
	return main.Path
}()

func Lookup(ctx context.Context) (<-chan map[string]string, error) {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize resolver: %w", err)
	}

	results := make(chan *zeroconf.ServiceEntry)
	output := make(chan map[string]string)
	err = resolver.Lookup(ctx, "", service, "", results)
	if err != nil {
		return nil, fmt.Errorf("Failed to Lookup: %w", err)
	}
	rss := serviceEntryState{
		state:      make(map[string]string),
		updateChan: make(chan struct{}, 1),
	}

	go func() {
		defer close(output)
		for {
			select {
			case <-rss.updateChan:
				output <- rss.Get(ctx)
			case result := <-results:
				go rss.Put(ctx, result)
			case <-ctx.Done():
				return
			}
		}
	}()
	return output, nil
}

type serviceEntryState struct {
	state      map[string]string
	mutex      sync.Mutex
	updateChan chan struct{}
}

func (ses *serviceEntryState) Get(ctx context.Context) map[string]string {
	ses.mutex.Lock()
	defer ses.mutex.Unlock()
	return ses.state
}

func (ses *serviceEntryState) Put(ctx context.Context, se *zeroconf.ServiceEntry) {
	ips := append([]net.IP{}, se.AddrIPv4...)
	ips = append(ips, se.AddrIPv6...)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	g, ctx := errgroup.WithContext(ctx)
	for _, ipI := range ips {
		ip := ipI // capture
		g.Go(func() error {
			url := fmt.Sprintf("http://%s:%d/", ip.String(), se.Port)
			log.Info().Msgf("going to try %v", url)
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			if err != nil {
				log.Error().Err(err).Msg("http.NewRequestWithContext")
				return nil
			}
			_, err = http.DefaultClient.Do(req)
			if err != nil && !errors.Is(err, context.Canceled) {
				log.Error().Err(err).Msg("http.DefaultClient.Do")
				return nil
			}
			ses.mutex.Lock()
			defer ses.mutex.Unlock()
			if ctx.Err() != nil {
				return nil
			}
			log.Info().Msgf("url ok %s", url)
			ses.state[se.Instance] = url
			cancel()
			ses.updateChan <- struct{}{}
			return nil
		})
	}
	g.Wait()
}
