package runner_test

import (
	"context"
	"fmt"

	"eagain.net/go/securityblanket/internal/runner"
	"golang.org/x/sync/errgroup"
)

func Example() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// this is just here to make the example reproducible
	ch := make(chan struct{})
	fn := func() error {
		fmt.Println("run")
		ch <- struct{}{}
		return nil
	}
	r := runner.New(ctx, fn, nil)
	var g errgroup.Group
	fmt.Println("before")
	g.Go(r.Loop)
	<-ch
	r.Wakeup()
	<-ch
	cancel()
	if err := g.Wait(); err != nil && err != context.Canceled {
		panic(err)
	}
	fmt.Println("after")

	// Output:
	// before
	// run
	// run
	// after
}
