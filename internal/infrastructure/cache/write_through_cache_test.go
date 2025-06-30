package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWriteThroughCache_Set 测试WriteThroughCache的Set方法
// 验证写透缓存的核心功能：先写存储，再写缓存
func TestWriteThroughCache_Set(t *testing.T) {
	tests := []struct {
		name           string
		setupCache     func() *MockCache
		setupStoreFunc func() (func(ctx context.Context, key string, val any) error, *[]string)
		key            string
		value          any
		expiration     time.Duration
		wantErr        error
		wantStored     bool
		wantCached     bool
	}{
		{
			name: "成功写入存储和缓存",
			setupCache: func() *MockCache {
				return &MockCache{store: make(map[string]any)}
			},
			setupStoreFunc: func() (func(ctx context.Context, key string, val any) error, *[]string) {
				stored := make([]string, 0)
				storeFunc := func(ctx context.Context, key string, val any) error {
					stored = append(stored, key)
					return nil
				}
				return storeFunc, &stored
			},
			key:        "key1",
			value:      "value1",
			expiration: time.Minute,
			wantErr:    nil,
			wantStored: true,
			wantCached: true,
		},
		{
			name: "存储失败_不写入缓存",
			setupCache: func() *MockCache {
				return &MockCache{store: make(map[string]any)}
			},
			setupStoreFunc: func() (func(ctx context.Context, key string, val any) error, *[]string) {
				stored := make([]string, 0)
				storeFunc := func(ctx context.Context, key string, val any) error {
					return errors.New("store failed")
				}
				return storeFunc, &stored
			},
			key:        "key1",
			value:      "value1",
			expiration: time.Minute,
			wantErr:    errors.New("store failed"),
			wantStored: false,
			wantCached: false,
		},
		{
			name: "存储成功_缓存失败",
			setupCache: func() *MockCache {
				return &MockCache{store: make(map[string]any), setShouldFail: true}
			},
			setupStoreFunc: func() (func(ctx context.Context, key string, val any) error, *[]string) {
				stored := make([]string, 0)
				storeFunc := func(ctx context.Context, key string, val any) error {
					stored = append(stored, key)
					return nil
				}
				return storeFunc, &stored
			},
			key:        "key1",
			value:      "value1",
			expiration: time.Minute,
			wantErr:    errors.New("mock set error"),
			wantStored: true,
			wantCached: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCache := tt.setupCache()
			storeFunc, stored := tt.setupStoreFunc()

			wtCache := &WriteThroughCache{
				Repository: mockCache,
				StoreFunc:  storeFunc,
			}

			err := wtCache.Set(context.Background(), tt.key, tt.value, tt.expiration)

			// 检查错误
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.Equal(t, tt.wantErr.Error(), err.Error())
			} else {
				require.NoError(t, err)
			}

			// 检查是否写入存储
			if tt.wantStored {
				assert.Contains(t, *stored, tt.key)
			} else {
				assert.NotContains(t, *stored, tt.key)
			}

			// 检查是否写入缓存
			if tt.wantCached {
				cachedVal, err := mockCache.Get(context.Background(), tt.key)
				require.NoError(t, err)
				assert.Equal(t, tt.value, cachedVal)
			} else {
				_, err := mockCache.Get(context.Background(), tt.key)
				assert.Error(t, err)
			}
		})
	}
}

// TestWriteThroughCache_OtherMethods 测试WriteThroughCache的其他方法
func TestWriteThroughCache_OtherMethods(t *testing.T) {
	tests := []struct {
		name      string
		setupData map[string]any
		operation string
		key       string
		value     any
		wantValue any
		wantErr   bool
	}{
		{
			name:      "Get方法委托给底层缓存",
			setupData: map[string]any{"key1": "value1"},
			operation: "get",
			key:       "key1",
			wantValue: "value1",
			wantErr:   false,
		},
		{
			name:      "Delete方法委托给底层缓存",
			setupData: map[string]any{"key1": "value1"},
			operation: "delete",
			key:       "key1",
			wantErr:   false,
		},
		{
			name:      "LoadAndDelete方法委托给底层缓存",
			setupData: map[string]any{"key2": "value2"},
			operation: "loadAndDelete",
			key:       "key2",
			wantValue: "value2",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCache := &MockCache{store: tt.setupData}
			wtCache := &WriteThroughCache{Repository: mockCache}

			switch tt.operation {
			case "get":
				val, err := wtCache.Get(context.Background(), tt.key)
				if tt.wantErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					assert.Equal(t, tt.wantValue, val)
				}
			case "delete":
				err := wtCache.Delete(context.Background(), tt.key)
				if tt.wantErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			case "loadAndDelete":
				val, err := wtCache.LoadAndDelete(context.Background(), tt.key)
				if tt.wantErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					assert.Equal(t, tt.wantValue, val)
				}
			}
		})
	}

	// 测试OnEvicted方法
	t.Run("OnEvicted方法委托给底层缓存", func(t *testing.T) {
		mockCache := &MockCache{store: make(map[string]any)}
		wtCache := &WriteThroughCache{Repository: mockCache}

		wtCache.OnEvicted(func(key string, val any) {
			// 测试回调函数
		})
	})
}

// TestRateLimitWriteThroughCache_Set 测试RateLimitWriteThroughCache的Set方法
// 验证带限流功能的写透缓存：限流时跳过存储，非限流时正常写入存储和缓存
func TestRateLimitWriteThroughCache_Set(t *testing.T) {
	tests := []struct {
		name           string
		setupCache     func() *MockCache
		setupStoreFunc func() (func(ctx context.Context, key string, val any) error, *[]string)
		setupContext   func() context.Context
		key            string
		value          any
		expiration     time.Duration
		wantErr        error
		wantStored     bool
		wantCached     bool
	}{
		{
			name: "未限流_成功写入存储和缓存",
			setupCache: func() *MockCache {
				return &MockCache{store: make(map[string]any)}
			},
			setupStoreFunc: func() (func(ctx context.Context, key string, val any) error, *[]string) {
				stored := make([]string, 0)
				storeFunc := func(ctx context.Context, key string, val any) error {
					stored = append(stored, key)
					return nil
				}
				return storeFunc, &stored
			},
			setupContext: func() context.Context {
				return context.Background()
			},
			key:        "key1",
			value:      "value1",
			expiration: time.Minute,
			wantErr:    nil,
			wantStored: true,
			wantCached: true,
		},
		{
			name: "被限流_跳过存储_仅写入缓存",
			setupCache: func() *MockCache {
				return &MockCache{store: make(map[string]any)}
			},
			setupStoreFunc: func() (func(ctx context.Context, key string, val any) error, *[]string) {
				stored := make([]string, 0)
				storeFunc := func(ctx context.Context, key string, val any) error {
					stored = append(stored, key)
					return nil
				}
				return storeFunc, &stored
			},
			setupContext: func() context.Context {
				return context.WithValue(context.Background(), "limited", true)
			},
			key:        "key1",
			value:      "value1",
			expiration: time.Minute,
			wantErr:    nil,
			wantStored: false,
			wantCached: true,
		},
		{
			name: "未限流_存储失败_不写入缓存",
			setupCache: func() *MockCache {
				return &MockCache{store: make(map[string]any)}
			},
			setupStoreFunc: func() (func(ctx context.Context, key string, val any) error, *[]string) {
				stored := make([]string, 0)
				storeFunc := func(ctx context.Context, key string, val any) error {
					return errors.New("store failed")
				}
				return storeFunc, &stored
			},
			setupContext: func() context.Context {
				return context.Background()
			},
			key:        "key1",
			value:      "value1",
			expiration: time.Minute,
			wantErr:    errors.New("store failed"),
			wantStored: false,
			wantCached: false,
		},
		{
			name: "被限流_缓存失败",
			setupCache: func() *MockCache {
				return &MockCache{store: make(map[string]any), setShouldFail: true}
			},
			setupStoreFunc: func() (func(ctx context.Context, key string, val any) error, *[]string) {
				stored := make([]string, 0)
				storeFunc := func(ctx context.Context, key string, val any) error {
					stored = append(stored, key)
					return nil
				}
				return storeFunc, &stored
			},
			setupContext: func() context.Context {
				return context.WithValue(context.Background(), "limited", true)
			},
			key:        "key1",
			value:      "value1",
			expiration: time.Minute,
			wantErr:    errors.New("mock set error"),
			wantStored: false,
			wantCached: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCache := tt.setupCache()
			storeFunc, stored := tt.setupStoreFunc()
			ctx := tt.setupContext()

			rlwtCache := &RateLimitWriteThroughCache{
				Repository: mockCache,
				StoreFunc:  storeFunc,
			}

			err := rlwtCache.Set(ctx, tt.key, tt.value, tt.expiration)

			// 检查错误
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.Equal(t, tt.wantErr.Error(), err.Error())
			} else {
				require.NoError(t, err)
			}

			// 检查是否写入存储
			if tt.wantStored {
				assert.Contains(t, *stored, tt.key)
			} else {
				assert.NotContains(t, *stored, tt.key)
			}

			// 检查是否写入缓存
			if tt.wantCached {
				cachedVal, err := mockCache.Get(context.Background(), tt.key)
				require.NoError(t, err)
				assert.Equal(t, tt.value, cachedVal)
			} else {
				_, err := mockCache.Get(context.Background(), tt.key)
				assert.Error(t, err)
			}
		})
	}
}
