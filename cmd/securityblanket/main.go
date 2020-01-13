package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"crawshaw.io/sqlite"
	"eagain.net/go/securityblanket/internal/database"
	"eagain.net/go/securityblanket/internal/honeywell5800/hw58receive"
	"eagain.net/go/securityblanket/internal/honeywell5800/hw58trip"
	"eagain.net/go/securityblanket/internal/rtl433receive"
	"eagain.net/go/securityblanket/internal/rtl433sql"
	"eagain.net/go/securityblanket/internal/runner"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/sync/errgroup"
)

func newZap() (*zap.Logger, error) {
	zapCfg := zap.Config{
		Level:             zap.NewAtomicLevelAt(defaultZapLevel),
		Development:       isDev,
		DisableCaller:     true,
		DisableStacktrace: true,
		Encoding:          defaultZapEncoding,
		EncoderConfig:     zap.NewProductionEncoderConfig(),
		OutputPaths:       []string{"stderr"},
		ErrorOutputPaths:  []string{"stderr"},
	}
	zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	logger, err := zapCfg.Build()
	if err != nil {
		return nil, fmt.Errorf("zap: %w", err)
	}
	return logger, nil
}

type config struct {
	DBPath    string
	SDRDevice string
}

func run(conf *config) error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	log, err := newZap()
	if err != nil {
		return fmt.Errorf("error configuring logging: %w", err)
	}
	defer log.Sync()
	_ = zap.ReplaceGlobals(log)
	_ = zap.RedirectStdLog(log)

	sqlite.Logger = func(code sqlite.ErrorCode, msg []byte) {
		switch code {
		default:
			log.Debug("sqlite",
				zap.Stringer("code", code),
				zap.ByteString("msg", msg),
			)
		case sqlite.SQLITE_MISUSE:
			// https://www.sqlite.org/rescode.html#misuse
			log.DPanic("sqlite.misuse",
				zap.Stringer("code", code),
				zap.ByteString("msg", msg),
			)
		case sqlite.SQLITE_LOCKED_SHAREDCACHE:
			// silence: this is something that happens often and is
			// handled by the sqlite wrapper
			//
			// https://www.sqlite.org/rescode.html#locked_sharedcache
			// https://www.sqlite.org/unlock_notify.html
		case sqlite.SQLITE_NOTICE_RECOVER_WAL:
			// silence: we don't need to see this on every startup
		}
	}

	db, err := database.Open(conf.DBPath)
	if err != nil {
		return err
	}
	defer db.Close()

	g, ctx := errgroup.WithContext(ctx)

	hw58TripLog := log.Named("honeywell5800.trip")
	hw58Trip := hw58trip.New(ctx, db, hw58TripLog)
	hw58TripRunnerLog := log.Named("honeywell5800.trip.runner")
	hw58TripRunner := runner.New(ctx, hw58Trip.Run, hw58TripRunnerLog)
	g.Go(hw58TripRunner.Loop)

	hw58RecvLog := log.Named("honeywell5800.receive")
	hw58Recv := hw58receive.New(ctx, db, hw58RecvLog, hw58TripRunner.Wakeup)
	hw58RecvRunnerLog := log.Named("honeywell5800.receive.runner")
	hw58RecvRunner := runner.New(ctx, hw58Recv.Run, hw58RecvRunnerLog)
	g.Go(hw58RecvRunner.Loop)

	rtl433store := rtl433sql.New(db, 345,
		rtl433sql.Wakeup(hw58RecvRunner.Wakeup),
	)
	g.Go(func() error {
		return rtl433receive.Receive(
			ctx,
			log.Named("rtl433.receive"),
			conf.SDRDevice,
			344975000,
			rtl433store,
		)
	})

	return g.Wait()
}

const prog = "securityblanket"

func usage() {
	fmt.Fprintf(flag.CommandLine.Output(), "Usage:\n")
	fmt.Fprintf(flag.CommandLine.Output(), "  %s DATABASE\n", prog)
	flag.PrintDefaults()
	if isDev {
		fmt.Fprintf(flag.CommandLine.Output(), "\nrunning in development mode\n")
	}
}

func main() {
	conf := &config{}
	flag.StringVar(&conf.SDRDevice, "sdr-device", "",
		"SDR device to listen to. USB device index or colon and serial number.",
	)
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() != 1 {
		usage()
		os.Exit(2)
	}
	conf.DBPath = flag.Arg(0)

	if err := run(conf); err != nil {
		log.Fatal("aborting", zap.Error(err))
	}
}
