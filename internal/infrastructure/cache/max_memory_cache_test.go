// Package cache_test 包含MaxMemoryCache的测试用例
package cache

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var errNotFound = errors.New("not found")

// TestMaxMemoryCache_Set 测试Set方法
func TestMaxMemoryCache_Set(t *testing.T) {
	var testCases []struct {
		name     string
		cache    func() *MaxMemoryCache
		key      string
		val      []byte
		wantKeys []string
		wantErr  error
		wantUsed int64
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
	var testCases []struct {
		name    string
		cache   func() *MaxMemoryCache
		key     string
		wantErr error
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

// TestMaxMemoryCache_Delete 测试Delete方法
func TestMaxMemoryCache_Delete(t *testing.T) {
	mock := &mockCache{
		data: make(map[string]any),
	}
	cache := NewMaxMemoryCache(100, mock)
	cache.policy = NewLRUPolicy()

	// 添加测试数据
	_ = cache.Set(context.Background(), "key1", []byte("value1"), time.Minute)
	_ = cache.Set(context.Background(), "key2", []byte("value2"), time.Minute)

	// 测试删除存在的key
	err := cache.Delete(context.Background(), "key1")
	assert.Nil(t, err)
	_, err = cache.Get(context.Background(), "key1")
	assert.Equal(t, errNotFound, err)

	// 测试删除不存在的key
	err = cache.Delete(context.Background(), "key3")
	assert.Nil(t, err)

	// 验证内存使用量减少
	assert.Equal(t, int64(6), cache.used) // "value2"长度为6
}

// TestMaxMemoryCache_Set_Eviction 测试设置时的淘汰逻辑
func TestMaxMemoryCache_Set_Eviction(t *testing.T) {
	mock := &mockCache{
		data: make(map[string]any),
	}
	cache := NewMaxMemoryCache(10, mock) // 设置小内存限制
	cache.policy = NewLRUPolicy()

	// 添加第一个key
	err := cache.Set(context.Background(), "key1", []byte("val1"), time.Minute)
	assert.Nil(t, err)
	assert.Equal(t, int64(4), cache.used) // "val1"长度为4

	// 添加第二个key，不会触发淘汰
	err = cache.Set(context.Background(), "key2", []byte("val2"), time.Minute)
	assert.Nil(t, err)
	assert.Equal(t, int64(8), cache.used) // 4+4=8

	// 添加第三个key，触发淘汰
	err = cache.Set(context.Background(), "key3", []byte("val3"), time.Minute)
	assert.Nil(t, err)
	assert.Equal(t, int64(8), cache.used) // 淘汰key1后，8-4+4=8

	// 验证key1被淘汰
	_, err = cache.Get(context.Background(), "key1")
	assert.Equal(t, errNotFound, err)

	// 验证key2和key3存在
	val, err := cache.Get(context.Background(), "key2")
	assert.Nil(t, err)
	assert.Equal(t, []byte("val2"), val)

	val, err = cache.Get(context.Background(), "key3")
	assert.Nil(t, err)
	assert.Equal(t, []byte("val3"), val)
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
	if assert.NotNil(t, err) {
		assert.Equal(t, "value is not []byte", err.Error())
	}
	assert.Nil(t, val)

	// 测试LoadAndDelete方法类型断言失败
	val, err = cache.LoadAndDelete(context.Background(), "key1")
	if assert.NotNil(t, err) {
		assert.Equal(t, "value is not []byte", err.Error())
	}
	assert.Nil(t, val)
}

// mockCache 模拟缓存实现
type mockCache struct {
	fn   func(key string, val any)
	data map[string]any
	mu   sync.Mutex
}

func (m *mockCache) Set(_ context.Context, key string, val any, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = val
	return nil
}

func (m *mockCache) Get(_ context.Context, key string) (any, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	val, ok := m.data[key]
	if ok {
		return val, nil
	}
	return nil, errNotFound
}

func (m *mockCache) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	val, ok := m.data[key]
	if ok {
		delete(m.data, key) // 实际删除键
		if m.fn != nil {
			m.fn(key, val)
		}
	}
	return nil
}

func (m *mockCache) LoadAndDelete(_ context.Context, key string) (any, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	val, ok := m.data[key]
	if ok {
		delete(m.data, key) // 实际删除键
		if m.fn != nil {
			m.fn(key, val)
		}
		return val, nil
	}
	return nil, errNotFound
}

func (m *mockCache) OnEvicted(fn func(key string, val any)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.fn = fn
}
