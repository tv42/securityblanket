package rtl433receive

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strconv"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// Store describes methods for receiving radio messages.
//
// All methods are expected to return quickly.
//
// If a method returns an error, Run will exit with that error.
type Store interface {
	Store(ctx context.Context, data []byte) error
}

func Receive(ctx context.Context, log *zap.Logger, device string, frequency uint64, store Store) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	cmd := exec.CommandContext(ctx, "rtl_433",
		"-M", "newmodel",
		"-F", "json",
		"-f", strconv.FormatUint(frequency, 10),
	)
	if device != "" {
		cmd.Args = append(cmd.Args,
			"-d", device,
		)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("running rtl_433: cannot set up stdout: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("running rtl_433: cannot set up stderr: %v", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting rtl_433: %v", err)
	}

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		s := bufio.NewScanner(stderr)
		for s.Scan() {
			log.Named("stderr").Info(s.Text())
		}
		if err := s.Err(); err != nil {
			return fmt.Errorf("reading rtl_433 stderr: %v", err)
		}
		return nil
	})
	g.Go(func() error {
		s := bufio.NewScanner(stdout)
		for s.Scan() {
			if err := store.Store(ctx, s.Bytes()); err != nil {
				return fmt.Errorf("rtl433 store error: %w", err)
			}
		}
		if err := s.Err(); err != nil {
			return fmt.Errorf("reading from rtl_433: %v", err)
		}
		return nil
	})
	if err := g.Wait(); err != nil {
		return err
	}
	return cmd.Wait()
}
