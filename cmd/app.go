package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"redCollar/internal/components"
	"redCollar/internal/config"
	"sync"
	"syscall"
)

func Run() error {
	logger := components.SetupLogger("local")
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Error("load config failed", "err", err)
		return err
	}
	if cfg.APIKey == "" {
		return fmt.Errorf("API_KEY is empty")
	}

	appCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	comps, err := components.InitComponents(appCtx, cfg, logger)
	if err != nil {
		logger.Error("could not init components", "err", err)
		return err
	}

	ctx, stop := context.WithCancel(context.Background())
	defer stop()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := comps.HttpServer.Run(ctx); err != nil {
			logger.Error("http server failed", "err", err)
		}
		logger.Info("http server stopped")
	}()

	quitChan := make(chan os.Signal, 1)
	signal.Notify(quitChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quitChan

	stop()
	logger.Info("captured signal, initiating shutdown", "signal", sig.String())

	wg.Wait()

	logger.Info("shutting down the services...")
	comps.ShutdownAll()
	logger.Info("gracefully shutting down the servers")

	return nil
}
