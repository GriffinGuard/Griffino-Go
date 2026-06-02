// Copyright (c) 2024-2026 MorCherlf. Licensed under the MIT License.

// Command deepseek is a runnable Griffino plugin that exposes an
// "ai.chat.model" capability. It mirrors the reference Python example: the chat
// handler here is an illustrative stub, but the wiring (metadata, provider
// registration, config declaration, and the Run loop) is what a real plugin
// uses.
//
// Generate the platform artifacts with: go run ./gen
package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/GriffinGuard/Griffino-Go/examples/deepseek/plugin"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	client, err := plugin.NewClient()
	if err != nil {
		log.Fatalf("deepseek: failed to build client: %v", err)
	}

	if err := client.Run(ctx); err != nil {
		log.Fatalf("deepseek: client exited with error: %v", err)
	}
}
