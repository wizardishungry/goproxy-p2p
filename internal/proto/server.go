package proto

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/rpc"
	"time"

	"github.com/goproxy/goproxy"
)

func NewServer(ctx context.Context, cacher goproxy.Cacher) (*rpc.Server, error) {
	s := rpc.NewServer()
	err := s.RegisterName(apiBase, &CacherRPC{
		ctx:    ctx,
		cacher: cacher,
	})
	return s, err
}

// CacherRPC binds a cache to an rpc service
type CacherRPC struct {
	cacher goproxy.Cacher
	ctx    context.Context
}

func (c *CacherRPC) Get(req RemoteRequest, reply *RemoteResponse) error {
	r, err := c.cacher.Get(c.ctx, req.Name)
	if err != nil {
		return err
	}
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return fmt.Errorf("ioutil.ReadAll: %w", err)
	}
	*reply = RemoteResponse{
		Body: b,
	}
	if mt, ok := r.(interface{ ModTime() time.Time }); ok {
		reply.ModTime = mt.ModTime()
	}
	return nil
}
