// Copyright (c) 2024-2026 MorCherlf. Licensed under the MIT License.

// Package redisstore implements a Redis-backed store for per-user plugin
// configuration. It mirrors the behavior of the Python SDK's
// griffino.redis_store.RedisUserConfigStore.
//
// The store reads a single key per user/plugin pair, whose value is a JSON
// object of user config values:
//
//	{keyPrefix}:{userId}:plugin:{pluginId}:config
//
// A missing key yields an empty map rather than an error.
//
// This package is intentionally standalone: it does not import the root
// griffino package, to avoid an import cycle.
package redisstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"
)

const defaultKeyPrefix = "user"

// Config describes how to reach the Redis instance that backs per-user plugin
// configuration. It is a local mirror of the root package's RedisConfig so that
// this package stays free of an import cycle.
type Config struct {
	URL       string
	Host      string
	Port      int
	DB        int
	Username  string
	Password  string
	KeyPrefix string
}

// Store reads per-user plugin configuration from Redis. It wraps a redis client
// together with the plugin identity and the key prefix.
type Store struct {
	client    *redis.Client
	pluginID  string
	keyPrefix string
}

// New builds a Store from cfg for the given plugin id.
//
// When cfg.URL is set, the client is built from it via redis.ParseURL; any
// non-zero Username, Password or DB then override the values parsed from the
// URL. Otherwise a client is built from Host:Port with the supplied
// credentials and DB. An empty KeyPrefix defaults to "user".
//
// go-redis dials lazily, so a well-formed config never produces a connection
// error here.
func New(pluginID string, cfg Config) (*Store, error) {
	opts, err := buildOptions(cfg)
	if err != nil {
		return nil, err
	}

	prefix := cfg.KeyPrefix
	if prefix == "" {
		prefix = defaultKeyPrefix
	}

	return &Store{
		client:    redis.NewClient(opts),
		pluginID:  pluginID,
		keyPrefix: prefix,
	}, nil
}

// buildOptions translates a Config into redis.Options, applying the URL-vs-host
// precedence described on New.
func buildOptions(cfg Config) (*redis.Options, error) {
	if cfg.URL != "" {
		opts, err := redis.ParseURL(cfg.URL)
		if err != nil {
			return nil, fmt.Errorf("redisstore: invalid redis URL: %w", err)
		}
		// Explicitly configured fields take precedence over the URL.
		if cfg.Username != "" {
			opts.Username = cfg.Username
		}
		if cfg.Password != "" {
			opts.Password = cfg.Password
		}
		if cfg.DB != 0 {
			opts.DB = cfg.DB
		}
		return opts, nil
	}

	return &redis.Options{
		Addr:     cfg.Host + ":" + strconv.Itoa(cfg.Port),
		Username: cfg.Username,
		Password: cfg.Password,
		DB:       cfg.DB,
	}, nil
}

// GetValues returns the stored user config values for userID. A missing key
// yields an empty map and a nil error. The stored value must be a JSON object.
func (s *Store) GetValues(ctx context.Context, userID string) (map[string]any, error) {
	key := configKey(s.keyPrefix, userID, s.pluginID)
	payload, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return map[string]any{}, nil
		}
		return nil, fmt.Errorf("redisstore: get %q: %w", key, err)
	}
	return decodeValues(payload)
}

// Close releases the underlying redis client.
func (s *Store) Close() error {
	return s.client.Close()
}

// configKey builds the Redis key for a user/plugin config blob. It mirrors the
// Python SDK's RedisUserConfigStore._build_key.
func configKey(prefix, userID, pluginID string) string {
	return fmt.Sprintf("%s:%s:plugin:%s:config", prefix, userID, pluginID)
}

// decodeValues parses a stored payload into a config map. An empty payload
// yields an empty map; a non-empty payload must decode to a JSON object.
func decodeValues(payload string) (map[string]any, error) {
	if payload == "" {
		return map[string]any{}, nil
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
		return nil, fmt.Errorf("redisstore: user config payload must decode to a JSON object: %w", err)
	}
	if decoded == nil {
		return map[string]any{}, nil
	}
	return decoded, nil
}
