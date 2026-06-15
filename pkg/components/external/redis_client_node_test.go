package external

import (
	"strings"
	"testing"
	"time"

	"github.com/neohetj/matrix/pkg/types"
	"github.com/redis/go-redis/v9"
)

func TestRedisClientNodeInitIsLazy(t *testing.T) {
	node := RedisClientNodePrototype.New()

	err := node.Init(types.ConfigMap{
		"uri":      "redis://127.0.0.1:1/0",
		"poolSize": 1,
	})
	if err != nil {
		t.Fatalf("Init should not dial Redis: %v", err)
	}
}

func TestRedisClientNodeGetInstanceDoesNotPing(t *testing.T) {
	node := RedisClientNodePrototype.New()
	err := node.Init(types.ConfigMap{
		"uri":      "redis://127.0.0.1:1/0",
		"poolSize": 1,
	})
	if err != nil {
		t.Fatalf("Init should not dial Redis: %v", err)
	}

	instance, err := node.(types.SharedNode).GetInstance()
	if err != nil {
		t.Fatalf("GetInstance should return a lazy Redis client: %v", err)
	}
	client, ok := instance.(*redis.Client)
	if !ok || client == nil {
		t.Fatalf("instance = %T, want *redis.Client", instance)
	}
}

func TestRedisClientNodeAppliesTimeoutAndRetryOptions(t *testing.T) {
	node := RedisClientNodePrototype.New()
	err := node.Init(types.ConfigMap{
		"uri":             "redis://127.0.0.1:1/0",
		"poolSize":        3,
		"dialTimeout":     "15s",
		"readTimeout":     "12s",
		"writeTimeout":    "13s",
		"maxRetries":      4,
		"minRetryBackoff": "200ms",
		"maxRetryBackoff": "2s",
	})
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	instance, err := node.(types.SharedNode).GetInstance()
	if err != nil {
		t.Fatalf("GetInstance failed: %v", err)
	}
	client := instance.(*redis.Client)
	opts := client.Options()

	if opts.PoolSize != 3 {
		t.Fatalf("PoolSize=%d, want 3", opts.PoolSize)
	}
	if opts.DialTimeout != 15*time.Second {
		t.Fatalf("DialTimeout=%s, want 15s", opts.DialTimeout)
	}
	if opts.ReadTimeout != 12*time.Second {
		t.Fatalf("ReadTimeout=%s, want 12s", opts.ReadTimeout)
	}
	if opts.WriteTimeout != 13*time.Second {
		t.Fatalf("WriteTimeout=%s, want 13s", opts.WriteTimeout)
	}
	if opts.MaxRetries != 4 {
		t.Fatalf("MaxRetries=%d, want 4", opts.MaxRetries)
	}
	if opts.MinRetryBackoff != 200*time.Millisecond {
		t.Fatalf("MinRetryBackoff=%s, want 200ms", opts.MinRetryBackoff)
	}
	if opts.MaxRetryBackoff != 2*time.Second {
		t.Fatalf("MaxRetryBackoff=%s, want 2s", opts.MaxRetryBackoff)
	}
}

func TestRedisClientNodeDoesNotEnableTLSForRedisURIWhenTLSInsecureIsTrue(t *testing.T) {
	node := RedisClientNodePrototype.New()
	err := node.Init(types.ConfigMap{
		"uri":          "redis://127.0.0.1:1/0",
		"tls_insecure": true,
	})
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	instance, err := node.(types.SharedNode).GetInstance()
	if err != nil {
		t.Fatalf("GetInstance failed: %v", err)
	}
	client := instance.(*redis.Client)

	if client.Options().TLSConfig != nil {
		t.Fatal("TLSConfig should stay nil for redis:// even when tls_insecure=true")
	}
}

func TestRedisClientNodeEnablesTLSInsecureForRedissURI(t *testing.T) {
	node := RedisClientNodePrototype.New()
	err := node.Init(types.ConfigMap{
		"uri":          "rediss://127.0.0.1:1/0",
		"tls_insecure": true,
	})
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	instance, err := node.(types.SharedNode).GetInstance()
	if err != nil {
		t.Fatalf("GetInstance failed: %v", err)
	}
	client := instance.(*redis.Client)
	tlsConfig := client.Options().TLSConfig

	if tlsConfig == nil {
		t.Fatal("TLSConfig should be set for rediss://")
	}
	if !tlsConfig.InsecureSkipVerify {
		t.Fatal("InsecureSkipVerify should be true for rediss:// with tls_insecure=true")
	}
}

func TestRedisClientNodeRejectsInvalidDuration(t *testing.T) {
	node := RedisClientNodePrototype.New()
	err := node.Init(types.ConfigMap{
		"uri":         "redis://127.0.0.1:1/0",
		"dialTimeout": "forever",
	})
	if err == nil {
		t.Fatal("Init succeeded, want invalid duration error")
	}
	if !strings.Contains(err.Error(), "dialTimeout") {
		t.Fatalf("error=%q, want dialTimeout context", err.Error())
	}
}
