// themis-serve is the Themis AI runtime service. It exposes evidence-
// based vulnerability-analysis operations over HTTP, routing each
// request to the model that scores best on the matching benchmark
// category.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tofchaliss/themis/internal/llm"
	"github.com/tofchaliss/themis/internal/service"
)

func main() {

	addr := flag.String("addr", ":8080", "listen address")
	benchmarksRoot := flag.String(
		"benchmarks-root",
		"benchmarks",
		"benchmark suite root (definitions and validation results drive model routing)",
	)
	configRoot := flag.String(
		"config-root",
		".",
		"directory containing models.json (model registry)",
	)
	defaultModel := flag.String(
		"default-model",
		"",
		"model to use when routing has no benchmark data for a category",
	)
	timeout := flag.Duration(
		"timeout",
		llm.DefaultTimeout,
		"per-request model invocation timeout",
	)
	flag.Parse()

	if err := run(*addr, *benchmarksRoot, *configRoot, *defaultModel, *timeout); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func run(addr, benchmarksRoot, configRoot, defaultModel string, timeout time.Duration) error {

	registry, err := llm.LoadRegistry(configRoot)
	if err != nil {
		return err
	}

	router, err := service.NewRouter(benchmarksRoot)
	if err != nil {
		return fmt.Errorf("build routing table: %w", err)
	}

	srv := &service.Server{
		Registry:     registry,
		Router:       router,
		DefaultModel: defaultModel,
		Timeout:      timeout,
	}

	for category, models := range router.Table() {
		best, _ := router.Route(category)
		log.Printf("route %s -> %s (candidates: %v)", category, best, models)
	}

	httpServer := &http.Server{
		Addr:    addr,
		Handler: srv.Handler(),

		ReadTimeout: 30 * time.Second,
		// Write timeout must cover a full model generation.
		WriteTimeout: timeout + time.Minute,
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	errCh := make(chan error, 1)

	go func() {
		log.Printf("themis-serve listening on %s", addr)
		errCh <- httpServer.ListenAndServe()
	}()

	select {

	case err := <-errCh:
		return err

	case <-ctx.Done():
		log.Print("shutting down")

		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cancel()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return err
		}

		if err := <-errCh; !errors.Is(err, http.ErrServerClosed) {
			return err
		}

		return nil
	}
}
