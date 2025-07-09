package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, int64(1024*1024), config.MaxMemory)
	assert.Equal(t, time.Hour, config.DefaultExpiration)
	assert.Equal(t, 10*time.Minute, config.CleanupInterval)
	assert.Equal(t, "lru", config.EvictionPolicy)
	assert.False(t, config.EnableBloomFilter)
	assert.Equal(t, 0.01, config.BloomFilterFalsePositiveRate)
}

func TestWithMaxMemory(t *testing.T) {
	config := DefaultConfig()
	option := WithMaxMemory(2048)
	option(config)

	assert.Equal(t, int64(2048), config.MaxMemory)
}

func TestWithDefaultExpiration(t *testing.T) {
	config := DefaultConfig()
	option := WithDefaultExpiration(30 * time.Minute)
	option(config)

	assert.Equal(t, 30*time.Minute, config.DefaultExpiration)
}

func TestWithCleanupInterval(t *testing.T) {
	config := DefaultConfig()
	option := WithCleanupInterval(5 * time.Minute)
	option(config)

	assert.Equal(t, 5*time.Minute, config.CleanupInterval)
}

func TestWithEvictionPolicy(t *testing.T) {
	config := DefaultConfig()
	option := WithEvictionPolicy("fifo")
	option(config)

	assert.Equal(t, "fifo", config.EvictionPolicy)
}

func TestWithBloomFilter(t *testing.T) {
	config := DefaultConfig()
	option := WithBloomFilter(true, 0.05)
	option(config)

	assert.True(t, config.EnableBloomFilter)
	assert.Equal(t, 0.05, config.BloomFilterFalsePositiveRate)
}

func TestNewService(t *testing.T) {
	tests := []struct {
		name    string
		options []Option
		wantErr bool
	}{
		{
			name:    "default config",
			options: nil,
			wantErr: false,
		},
		{
			name: "with custom options",
			options: []Option{
				WithMaxMemory(2048),
				WithEvictionPolicy("lru"),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewService(tt.options...)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, service)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, service)
				assert.NotNil(t, service.appService)
			}
		})
	}
}

func TestNewServiceWithConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name:    "valid config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "custom config",
			config: &Config{
				MaxMemory:         2048,
				DefaultExpiration: 30 * time.Minute,
				CleanupInterval:   5 * time.Minute,
				EvictionPolicy:    "fifo",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewServiceWithConfig(tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, service)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, service)
				assert.NotNil(t, service.appService)
			}
		})
	}
}

func TestService_SetAndGet(t *testing.T) {
	service, err := NewService()
	require.NoError(t, err)

	ctx := context.Background()
	key := "test_key"
	value := "test_value"
	expiration := time.Hour

	// Test Set
	err = service.Set(ctx, key, value, expiration)
	assert.NoError(t, err)

	// Test Get
	retrievedValue, err := service.Get(ctx, key)
	assert.NoError(t, err)
	assert.Equal(t, value, retrievedValue)
}

func TestService_GetNonExistentKey(t *testing.T) {
	service, err := NewService()
	require.NoError(t, err)

	ctx := context.Background()

	_, err = service.Get(ctx, "non_existent_key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "键不存在")
}

func TestService_Delete(t *testing.T) {
	service, err := NewService()
	require.NoError(t, err)

	ctx := context.Background()
	key := "test_key"
	value := "test_value"

	// Set a value first
	err = service.Set(ctx, key, value, time.Hour)
	require.NoError(t, err)

	// Delete the value
	err = service.Delete(ctx, key)
	assert.NoError(t, err)

	// Verify it's deleted
	_, err = service.Get(ctx, key)
	assert.Error(t, err)
}

func TestService_LoadAndDelete(t *testing.T) {
	service, err := NewService()
	require.NoError(t, err)

	ctx := context.Background()
	key := "test_key"
	value := "test_value"

	// Set a value first
	err = service.Set(ctx, key, value, time.Hour)
	require.NoError(t, err)

	// Load and delete
	retrievedValue, err := service.LoadAndDelete(ctx, key)
	assert.NoError(t, err)
	assert.Equal(t, value, retrievedValue)

	// Verify it's deleted
	_, err = service.Get(ctx, key)
	assert.Error(t, err)
}

func TestService_LoadAndDeleteNonExistentKey(t *testing.T) {
	service, err := NewService()
	require.NoError(t, err)

	ctx := context.Background()

	_, err = service.LoadAndDelete(ctx, "non_existent_key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "键不存在")
}

func TestService_Stats(t *testing.T) {
	service, err := NewService()
	require.NoError(t, err)

	ctx := context.Background()

	stats, err := service.Stats(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.GreaterOrEqual(t, stats.HitRate, 0.0)
	assert.LessOrEqual(t, stats.HitRate, 1.0)
}

func TestService_Clear(t *testing.T) {
	service, err := NewService()
	require.NoError(t, err)

	ctx := context.Background()

	err = service.Clear(ctx)
	assert.Error(t, err) // 暂时不支持清空缓存
	assert.Contains(t, err.Error(), "暂未实现")
}

func TestService_OnEvicted(t *testing.T) {
	service, err := NewService()
	require.NoError(t, err)

	// 测试设置回调函数不会出错
	service.OnEvicted(func(key string, val any) {
		// 回调函数
	})
}

func TestNewReadThroughService(t *testing.T) {
	tests := []struct {
		name    string
		options []Option
		wantErr bool
	}{
		{
			name:    "default config",
			options: nil,
			wantErr: false,
		},
		{
			name: "with custom options",
			options: []Option{
				WithMaxMemory(2048),
				WithEvictionPolicy("lru"),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewReadThroughService(tt.options...)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, service)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, service)
				assert.NotNil(t, service.service)
			}
		})
	}
}

func TestReadThroughService_GetWithLoader(t *testing.T) {
	service, err := NewReadThroughService()
	require.NoError(t, err)

	ctx := context.Background()
	key := "test_key"
	expectedValue := "loaded_value"

	loader := func(ctx context.Context, key string) (any, error) {
		return expectedValue, nil
	}

	value, err := service.GetWithLoader(ctx, key, loader, time.Hour)
	assert.NoError(t, err)
	assert.Equal(t, expectedValue, value)
}

func TestReadThroughService_GetWithLoaderError(t *testing.T) {
	service, err := NewReadThroughService()
	require.NoError(t, err)

	ctx := context.Background()
	key := "test_key"

	loader := func(ctx context.Context, key string) (any, error) {
		return nil, assert.AnError
	}

	_, err = service.GetWithLoader(ctx, key, loader, time.Hour)
	assert.Error(t, err)
}
