// Copyright (c) 2024-2026 MorCherlf. Licensed under the MIT License.

package redisstore

import (
	"reflect"
	"testing"
)

func TestConfigKey(t *testing.T) {
	got := configKey("user", "u-1", "plug-9")
	want := "user:u-1:plugin:plug-9:config"
	if got != want {
		t.Fatalf("configKey = %q, want %q", got, want)
	}
}

func TestNewWithHostPortDefaultsPrefix(t *testing.T) {
	s, err := New("plug", Config{Host: "redis.internal", Port: 6379})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if s == nil {
		t.Fatal("New returned nil store")
	}
	if s.keyPrefix != defaultKeyPrefix {
		t.Fatalf("keyPrefix = %q, want default %q", s.keyPrefix, defaultKeyPrefix)
	}
	if s.client == nil {
		t.Fatal("store has nil client")
	}
	t.Cleanup(func() { _ = s.Close() })
}

func TestNewWithURLHonoursPrefix(t *testing.T) {
	s, err := New("plug", Config{URL: "redis://localhost:6379/0", KeyPrefix: "tenant"})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if s.keyPrefix != "tenant" {
		t.Fatalf("keyPrefix = %q, want %q", s.keyPrefix, "tenant")
	}
	t.Cleanup(func() { _ = s.Close() })
}

func TestBuildOptionsURLOverrides(t *testing.T) {
	opts, err := buildOptions(Config{
		URL:      "redis://urluser:urlpass@localhost:6379/0",
		Username: "override",
		Password: "secret",
		DB:       3,
	})
	if err != nil {
		t.Fatalf("buildOptions returned error: %v", err)
	}
	if opts.Username != "override" {
		t.Errorf("Username = %q, want override", opts.Username)
	}
	if opts.Password != "secret" {
		t.Errorf("Password = %q, want secret", opts.Password)
	}
	if opts.DB != 3 {
		t.Errorf("DB = %d, want 3", opts.DB)
	}
}

func TestBuildOptionsURLNoOverride(t *testing.T) {
	opts, err := buildOptions(Config{URL: "redis://urluser:urlpass@localhost:6379/2"})
	if err != nil {
		t.Fatalf("buildOptions returned error: %v", err)
	}
	if opts.Username != "urluser" {
		t.Errorf("Username = %q, want urluser", opts.Username)
	}
	if opts.Password != "urlpass" {
		t.Errorf("Password = %q, want urlpass", opts.Password)
	}
	if opts.DB != 2 {
		t.Errorf("DB = %d, want 2", opts.DB)
	}
}

func TestBuildOptionsHostPort(t *testing.T) {
	opts, err := buildOptions(Config{Host: "example", Port: 6380, Username: "u", Password: "p", DB: 1})
	if err != nil {
		t.Fatalf("buildOptions returned error: %v", err)
	}
	if opts.Addr != "example:6380" {
		t.Errorf("Addr = %q, want example:6380", opts.Addr)
	}
	if opts.Username != "u" || opts.Password != "p" || opts.DB != 1 {
		t.Errorf("unexpected opts: %+v", opts)
	}
}

func TestBuildOptionsInvalidURL(t *testing.T) {
	if _, err := buildOptions(Config{URL: "://not-a-url"}); err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestDecodeValues(t *testing.T) {
	got, err := decodeValues(`{"theme":"dark","count":2}`)
	if err != nil {
		t.Fatalf("decodeValues returned error: %v", err)
	}
	want := map[string]any{"theme": "dark", "count": float64(2)}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("decodeValues = %#v, want %#v", got, want)
	}
}

func TestDecodeValuesEmpty(t *testing.T) {
	got, err := decodeValues("")
	if err != nil {
		t.Fatalf("decodeValues returned error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("decodeValues = %#v, want empty map", got)
	}
	if got == nil {
		t.Fatal("decodeValues returned nil map, want non-nil empty map")
	}
}

func TestDecodeValuesNull(t *testing.T) {
	got, err := decodeValues("null")
	if err != nil {
		t.Fatalf("decodeValues returned error: %v", err)
	}
	if got == nil || len(got) != 0 {
		t.Fatalf("decodeValues(null) = %#v, want empty map", got)
	}
}

func TestDecodeValuesNotObject(t *testing.T) {
	if _, err := decodeValues(`[1,2,3]`); err == nil {
		t.Fatal("expected error decoding a JSON array")
	}
}
