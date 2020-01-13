package runner

import (
	"context"

	"go.uber.org/zap"
)

// Runner is a helper that makes it easy to write a loop that only
// wakes up when it's signaled, and terminates on context
// cancellation.
type Runner struct {
	ctx    context.Context
	run    func() error
	log    *zap.Logger
	wakeup chan struct{}
}

func New(ctx context.Context, fn func() error, log *zap.Logger) *Runner {
	if log == nil {
		log = zap.NewNop()
	}
	r := &Runner{
		ctx:    ctx,
		run:    fn,
		log:    log,
		wakeup: make(chan struct{}, 1),
	}
	// process any leftovers
	r.wakeup <- struct{}{}
	return r
}

func (r *Runner) Loop() error {
	// Loop does not take ctx as arg to fit better to the errgroup.Do
	// calling convention.
	for {
		select {
		case <-r.ctx.Done():
			r.log.Debug("exit")
			err := r.ctx.Err()
			return err
		case <-r.wakeup:
		}

		if err := r.run(); err != nil {
			return err
		}
	}
}

func (r *Runner) Wakeup() {
	select {
	case r.wakeup <- struct{}{}:
		r.log.Debug("wakeup")
	default:
		r.log.Debug("wakeup.slow")
	}
}
