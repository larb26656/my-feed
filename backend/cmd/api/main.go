package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/larb26656/my-feed/backend/internal/bootstrap"
)

// @title My Feed API
// @version 1.0
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := bootstrap.Run(ctx); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
