package rpcserver

import (
	"context"
	"sync"
	"time"

	"github.com/sourcegraph/jsonrpc2"
)

// cancellations holds the cancel function of every request still running, by id, and
// counts them so the server can wait for the last one.
type cancellations struct {
	mu      sync.Mutex
	running map[jsonrpc2.ID]context.CancelFunc
	active  sync.WaitGroup
}

func newCancellations() *cancellations {
	return &cancellations{running: map[jsonrpc2.ID]context.CancelFunc{}}
}

// start derives the request's context and remembers how to cancel it; done forgets it.
func (c *cancellations) start(ctx context.Context, id jsonrpc2.ID) (context.Context, func()) {
	ctx, cancel := context.WithCancel(ctx)
	c.active.Add(1)
	c.mu.Lock()
	c.running[id] = cancel
	c.mu.Unlock()

	return ctx, func() {
		c.mu.Lock()
		delete(c.running, id)
		c.mu.Unlock()
		cancel()
		c.active.Done()
	}
}

// stopAll cancels every running request and waits until each has returned, or until
// patience runs out.
func (c *cancellations) stopAll(patience time.Duration) bool {
	c.mu.Lock()
	for _, cancel := range c.running {
		cancel()
	}
	c.mu.Unlock()
	finished := make(chan struct{})
	go func() {
		c.active.Wait()
		close(finished)
	}()
	select {
	case <-finished:
		return true
	case <-time.After(patience):
		return false
	}
}

// cancel stops a running request; an id that already finished is ignored.
func (c *cancellations) cancel(id jsonrpc2.ID) {
	c.mu.Lock()
	cancel := c.running[id]
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}
