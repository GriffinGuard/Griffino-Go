// Copyright (c) 2024-2026 MorCherlf. Licensed under the MIT License.

// Package griffino is the Go SDK for building Griffino plugins.
//
// Griffino is a plugin-orchestration platform. Plugins run as Docker
// containers and communicate with the platform and with one another over
// RabbitMQ. This package lets a plugin author:
//
//   - register capability providers, event consumers, and user actions;
//   - invoke other plugins' capabilities and dispatch events;
//   - declare boot and user configuration via struct tags; and
//   - generate the platform manifest files (plugin.manifest.json,
//     config.boot.json, config.user.json, plugin.boot.yml).
//
// The public surface lives in this package. Wire-level helpers and optional
// backing stores live under internal/.
package griffino
