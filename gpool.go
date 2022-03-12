package gpool

import (
	"errors"
	"sync"
	"sync/atomic"
)

var (
	gp = NewGoPool(10000)
)

// Go
func Go(fn func() error) error {
	return gp.Go(fn)
}

var GoPoolMaxGoroutineError = errors.New("reach max goroutine count.")

type GoPoolOptions struct {
	reject func(fn func() error) error
}

func (o *GoPoolOptions) apply() {
	if o.reject == nil {
		o.reject = func(func() error) error {
			return GoPoolMaxGoroutineError
		}
	}
}

type GoPoolOption func(*GoPoolOptions)

func WithReject(fn func(func() error) error) GoPoolOption {
	return func(options *GoPoolOptions) {
		options.reject = fn
	}
}

func NewGoPool(maxCount uint32, opts ...GoPoolOption) *GoPool {
	var opt GoPoolOptions
	for _, o := range opts {
		o(&opt)
	}

	opt.apply()
	return &GoPool{
		opts: opt,
		poolWorker: sync.Pool{
			New: func() interface{} {
				return newPoolWorker()
			},
		},
		maxCount: maxCount,
	}
}

// GoPool
type GoPool struct {
	poolWorker sync.Pool

	maxCount uint32
	curCount uint32
	opts     GoPoolOptions
}

// Go pool will try get a goroutine from pool to exec fn
// if goroutine count reach max count, GoPool will use reject strategy handle.
func (p *GoPool) Go(fn func() error) error {
	if atomic.AddUint32(&p.curCount, 1) > p.maxCount {
		atomic.AddUint32(&p.curCount, -1)
		return p.opts.reject(fn)
	}
	pw := p.poolWorker.Get().(*poolWorker)
	err := pw.work(fn)
	p.poolWorker.Put(pw)
	atomic.AddUint32(&p.curCount, -1)
	return err
}
