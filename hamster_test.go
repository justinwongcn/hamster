package hamster

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/justinwongcn/hamster/cache"
	"github.com/justinwongcn/hamster/hash"
	"github.com/justinwongcn/hamster/lock"
)

func TestGetVersion(t *testing.T) {
	version := GetVersion()
	assert.Equal(t, Version, version)
	assert.Equal(t, "1.0.0", version)
}

func TestNewCache(t *testing.T) {
	tests := []struct {
		name    string
		options []cache.Option
		wantErr bool
	}{
		{
			name:    "default config",
			options: nil,
			wantErr: false,
		},
		{
			name: "with custom options",
			options: []cache.Option{
				cache.WithMaxMemory(2048),
				cache.WithEvictionPolicy("lru"),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewCache(tt.options...)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, service)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, service)
			}
		})
	}
}

func TestNewCacheWithConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *cache.Config
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name:    "valid config",
			config:  cache.DefaultConfig(),
			wantErr: false,
		},
		{
			name: "custom config",
			config: &cache.Config{
				MaxMemory:      2048,
				EvictionPolicy: "lru",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewCacheWithConfig(tt.config)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, service)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, service)
			}
		})
	}
}

func TestNewReadThroughCache(t *testing.T) {
	tests := []struct {
		name    string
		options []cache.Option
		wantErr bool
	}{
		{
			name:    "default config",
			options: nil,
			wantErr: false,
		},
		{
			name: "with custom options",
			options: []cache.Option{
				cache.WithMaxMemory(2048),
				cache.WithEvictionPolicy("lru"),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewReadThroughCache(tt.options...)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, service)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, service)
			}
		})
	}
}

func TestNewConsistentHash(t *testing.T) {
	tests := []struct {
		name    string
		options []hash.Option
		wantErr bool
	}{
		{
			name:    "default config",
			options: nil,
			wantErr: false,
		},
		{
			name: "with custom options",
			options: []hash.Option{
				hash.WithReplicas(100),
				hash.WithSingleflight(true),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewConsistentHash(tt.options...)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, service)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, service)
			}
		})
	}
}

func TestNewConsistentHashWithConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *hash.Config
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name:    "valid config",
			config:  hash.DefaultConfig(),
			wantErr: false,
		},
		{
			name: "custom config",
			config: &hash.Config{
				Replicas:           100,
				EnableSingleflight: false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewConsistentHashWithConfig(tt.config)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, service)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, service)
			}
		})
	}
}

func TestNewDistributedLock(t *testing.T) {
	tests := []struct {
		name    string
		options []lock.Option
		wantErr bool
	}{
		{
			name:    "default config",
			options: nil,
			wantErr: false,
		},
		{
			name: "with custom options",
			options: []lock.Option{
				lock.WithDefaultExpiration(60),
				lock.WithDefaultTimeout(10),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewDistributedLock(tt.options...)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, service)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, service)
			}
		})
	}
}

func TestNewDistributedLockWithConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *lock.Config
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name:    "valid config",
			config:  lock.DefaultConfig(),
			wantErr: false,
		},
		{
			name: "custom config",
			config: &lock.Config{
				DefaultExpiration: 60,
				DefaultTimeout:    10,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewDistributedLockWithConfig(tt.config)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, service)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, service)
			}
		})
	}
}

func TestIntegration_CacheBasicOperations(t *testing.T) {
	// Integration test for cache operations
	cacheService, err := NewCache(
		cache.WithMaxMemory(1024),
		cache.WithEvictionPolicy("lru"),
	)
	require.NoError(t, err)
	require.NotNil(t, cacheService)

	// This is a basic integration test to ensure the service is created properly
	// More detailed functionality tests are in the individual package tests
}

func TestIntegration_ConsistentHashBasicOperations(t *testing.T) {
	// Integration test for consistent hash operations
	hashService, err := NewConsistentHash(
		hash.WithReplicas(100),
		hash.WithSingleflight(true),
	)
	require.NoError(t, err)
	require.NotNil(t, hashService)

	// This is a basic integration test to ensure the service is created properly
	// More detailed functionality tests are in the individual package tests
}

func TestIntegration_DistributedLockBasicOperations(t *testing.T) {
	// Integration test for distributed lock operations
	lockService, err := NewDistributedLock(
		lock.WithDefaultExpiration(30),
		lock.WithDefaultTimeout(5),
	)
	require.NoError(t, err)
	require.NotNil(t, lockService)

	// This is a basic integration test to ensure the service is created properly
	// More detailed functionality tests are in the individual package tests
}

func TestVersion(t *testing.T) {
	// Test that version constant is properly defined
	assert.NotEmpty(t, Version)
	assert.Equal(t, "1.0.0", Version)
}

func TestAllServicesCanBeCreatedTogether(t *testing.T) {
	// Test that all services can be created simultaneously without conflicts
	cacheService, err := NewCache()
	require.NoError(t, err)
	require.NotNil(t, cacheService)

	hashService, err := NewConsistentHash()
	require.NoError(t, err)
	require.NotNil(t, hashService)

	lockService, err := NewDistributedLock()
	require.NoError(t, err)
	require.NotNil(t, lockService)

	// All services should be independent and not interfere with each other
	assert.NotNil(t, cacheService)
	assert.NotNil(t, hashService)
	assert.NotNil(t, lockService)
}
