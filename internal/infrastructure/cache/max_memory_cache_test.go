// Package cache_test 包含MaxMemoryCache的测试用例
package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var errNotFound = errors.New("not found")

// TestMaxMemoryCache_Set 测试Set方法
func TestMaxMemoryCache_Set(t *testing.T) {
	testCases := []struct {
		name     string
		cache    func() *MaxMemoryCache
		key      string
		val      []byte
		wantKeys []string
		wantErr  error
		wantUsed int64
	}{
		// 测试用例保持不变
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cache := tc.cache()
			err := cache.Set(context.Background(), tc.key, tc.val, time.Minute)
			assert.Equal(t, tc.wantUsed, cache.used)
			if tc.wantErr != nil {
				assert.Equal(t, tc.wantErr, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

// TestMaxMemoryCache_Get 测试Get方法
func TestMaxMemoryCache_Get(t *testing.T) {
	testCases := []struct {
		name    string
		cache   func() *MaxMemoryCache
		key     string
		wantErr error
	}{
		// 测试用例保持不变
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cache := tc.cache()
			_, err := cache.Get(context.Background(), tc.key)
			if tc.wantErr != nil {
				assert.Equal(t, tc.wantErr, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

// TestMaxMemoryCache_TypeAssertionFailure 测试类型断言失败
func TestMaxMemoryCache_TypeAssertionFailure(t *testing.T) {
	mock := &mockCache{
		data: map[string]any{
			"key1": "not a byte slice",
		},
		fn: func(key string, val any) {}, // 初始化回调函数
	}
	cache := NewMaxMemoryCache(100, mock)
	// 初始化缓存策略
	cache.policy = NewLRUPolicy()
	cache.policy.KeyAccessed("key1")
	cache.used = 6

	// 测试Get方法类型断言失败
	val, err := cache.Get(context.Background(), "key1")
	assert.Equal(t, "value is not []byte", err.Error())
	assert.Nil(t, val)

	// 测试LoadAndDelete方法类型断言失败
	val, err = cache.LoadAndDelete(context.Background(), "key1")
	assert.Equal(t, "value is not []byte", err.Error())
	assert.Nil(t, val)
}

// mockCache 模拟缓存实现
type mockCache struct {
	fn   func(key string, val any)
	data map[string]any
}

func (m *mockCache) Set(ctx context.Context, key string, val any, expiration time.Duration) error {
	m.data[key] = val
	return nil
}

func (m *mockCache) Get(ctx context.Context, key string) (any, error) {
	val, ok := m.data[key]
	if ok {
		return val, nil
	}
	return nil, errNotFound
}

func (m *mockCache) Delete(ctx context.Context, key string) error {
	val, ok := m.data[key]
	if ok {
		m.fn(key, val)
	}
	return nil
}

func (m *mockCache) LoadAndDelete(ctx context.Context, key string) (any, error) {
	val, ok := m.data[key]
	if ok {
		m.fn(key, val)
		return val, nil
	}
	return nil, errNotFound
}

func (m *mockCache) OnEvicted(fn func(key string, val any)) {
	m.fn = fn
}
