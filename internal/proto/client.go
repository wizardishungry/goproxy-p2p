package proto

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/rpc"
	"time"

	"github.com/goproxy/goproxy"
	"github.com/rs/zerolog/log"
)

const apiBase = "Cacher"

var APIBase = apiBase

func New(conn io.ReadWriteCloser, stringer fmt.Stringer) interface {
	goproxy.Cacher
	Close() error
} {
	rpcClient := rpc.NewClient(conn)
	return &remoter{
		Client: rpcClient,
	}
}

// Get gets the matched cache for the name. It returns the
// `os.ErrNotExist` if not found.
//
// It is the caller's responsibility to close the returned
// `io.ReadCloser`.
//
// Note that the returned `io.ReadCloser` can optionally implement the
// following interfaces:
//   * `io.Seeker`
//       For the Range request header.
//   * `interface{ ModTime() time.Time }`
//       For the Last-Modified response header.
//   * `interface{ Checksum() []byte }`
//       For the ETag response header.

type remoter struct {
	*rpc.Client
	stringer fmt.Stringer
}

var (
	_ goproxy.Cacher = &remoter{}
	_ fmt.Stringer   = &remoter{}
)

func (r *remoter) String() string {
	return r.stringer.String()
}

func (r *remoter) Get(ctx context.Context, name string) (io.ReadCloser, error) {
	const svcMethod = "Get"
	resp := &RemoteResponse{}

	log.Debug().Msg("Remoter go")
	call := r.Client.Go(apiBase+"."+svcMethod, RemoteRequest{name}, resp, nil)

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-call.Done:
	}

	if err := call.Error; err != nil {
		return nil, err
	}

	return &remoterReturn{
		ReadCloser: io.NopCloser(bytes.NewBuffer(resp.Body)), // TODO on found defer pushing bytes by binding rpc to response object
	}, nil
}

// Set sets the content as a cache with the name.
func (r *remoter) Set(ctx context.Context, name string, content io.ReadSeeker) error {
	return fmt.Errorf("unimplemented")
}

type RemoteRequest struct {
	Name string
}
type RemoteResponse struct {
	ModTime time.Time
	Body    []byte
}
type remoterReturn struct {
	io.ReadCloser
}

var _ interface {
	io.ReadCloser
	// ModTime() time.Time
	// Checksum() []byte
} = &remoterReturn{}
