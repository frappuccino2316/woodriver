package woodriver

import (
	"context"
	"fmt"
	"sync"
)

// SessionPool manages a fixed pool of browser sessions for concurrent use.
//
// Each goroutine acquires a session, uses it, then releases it back to the
// pool.  The pool blocks on Acquire until a session is available or the
// context is cancelled.
//
// Usage:
//
//	pool, err := woodriver.NewSessionPool(ctx, driver, 4, caps)
//	if err != nil { ... }
//	defer pool.Close()
//
//	var wg sync.WaitGroup
//	for _, url := range urls {
//	    wg.Add(1)
//	    go func(u string) {
//	        defer wg.Done()
//	        sess, err := pool.Acquire(ctx)
//	        if err != nil { return }
//	        defer pool.Release(sess)
//	        sess.Navigate(u)
//	    }(url)
//	}
//	wg.Wait()
type SessionPool struct {
	ch       chan WindowOps
	mu       sync.Mutex
	sessions []WindowOps // kept for Close
	closed   bool
}

// NewSessionPool creates a pool of size sessions using the given Driver and
// Capabilities.  All sessions are started eagerly; if any session fails to
// open the already-opened ones are closed and the error is returned.
func NewSessionPool(ctx context.Context, d *Driver, size int, caps Capabilities) (*SessionPool, error) {
	if size <= 0 {
		return nil, fmt.Errorf("SessionPool: size must be > 0")
	}

	pool := &SessionPool{
		ch:       make(chan WindowOps, size),
		sessions: make([]WindowOps, 0, size),
	}

	for i := 0; i < size; i++ {
		sess, err := d.NewSession(caps)
		if err != nil {
			// Close already-created sessions before returning.
			for _, s := range pool.sessions {
				_ = s.Quit()
			}
			return nil, fmt.Errorf("SessionPool: open session %d: %w", i, err)
		}
		pool.sessions = append(pool.sessions, sess)
		pool.ch <- sess
	}

	return pool, nil
}

// Acquire waits for a free session and returns it.
// Returns ctx.Err() if the context is cancelled before a session is available.
// Returns ErrSessionPoolClosed if the pool has been closed.
func (p *SessionPool) Acquire(ctx context.Context) (WindowOps, error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, ErrSessionPoolClosed
	}
	p.mu.Unlock()

	select {
	case sess := <-p.ch:
		return sess, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Release returns a session to the pool so another goroutine can use it.
// Panics if the pool is already closed to surface incorrect usage early.
func (p *SessionPool) Release(sess WindowOps) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		panic("woodriver: Release called on closed SessionPool")
	}
	p.ch <- sess
}

// Len returns the number of sessions currently available in the pool
// (i.e. not held by any goroutine).
func (p *SessionPool) Len() int { return len(p.ch) }

// Cap returns the total capacity of the pool.
func (p *SessionPool) Cap() int { return cap(p.ch) }

// Close quits all sessions and renders the pool unusable.
// It blocks until every session has been returned via Release.
func (p *SessionPool) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	p.mu.Unlock()

	// Drain the channel; wait until all borrowed sessions are returned.
	var errs []error
	for range p.sessions {
		sess := <-p.ch
		if err := sess.Quit(); err != nil {
			errs = append(errs, err)
		}
	}
	close(p.ch)

	if len(errs) > 0 {
		return fmt.Errorf("SessionPool.Close: %v", errs)
	}
	return nil
}

// ErrSessionPoolClosed is returned by Acquire when the pool is closed.
var ErrSessionPoolClosed = &WebDriverError{Code: "session pool closed", Message: "SessionPool has been closed"}
