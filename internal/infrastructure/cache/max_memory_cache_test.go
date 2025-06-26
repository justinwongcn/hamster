// Package cache_test 包含MaxMemoryCache的测试用例
package cache

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/justinwongcn/hamster/internal/interfaces"
	"github.com/stretchr/testify/assert"
)

// errNotFound 表示键不存在的错误
var errNotFound = errors.New("not found")

// TestMaxMemoryCache_Set 测试Set方法
// 参数:
//   - t: 测试上下文
//
// 功能:
//   - 验证Set方法在不同场景下的行为
//
// 测试用例:
//   - 内存不足触发淘汰
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
		{
			name: "内存不足触发淘汰",
			cache: func() *MaxMemoryCache {
				mock := &mockCache{data: make(map[string]any)}
				cache := NewMaxMemoryCache(10, mock)
				cache.policy = NewLRUPolicy()
				// 预先添加一个将被淘汰的键
				_ = cache.Set(context.Background(), "oldKey", []byte("oldVal"), time.Minute)
				return cache
			},
			key:      "key2",
			val:      []byte("value2"), // 6个字节
			wantKeys: []string{"key2"},
			wantUsed: 6,
		},
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
// 参数:
//   - t: 测试上下文
//
// 功能:
//   - 验证Get方法在不同场景下的行为
//
// 测试用例:
//   - 正常获取
//   - 键不存在
func TestMaxMemoryCache_Get(t *testing.T) {
	testCases := []struct {
		name    string
		cache   func() *MaxMemoryCache
		key     string
		wantErr error
	}{
		{
			name: "正常获取",
			cache: func() *MaxMemoryCache {
				mock := &mockCache{data: make(map[string]any)}
				cache := NewMaxMemoryCache(100, mock)
				cache.policy = NewLRUPolicy()
				_ = cache.Set(context.Background(), "key1", []byte("value1"), time.Minute)
				return cache
			},
			key:     "key1",
			wantErr: nil,
		},
		{
			name: "键不存在",
			cache: func() *MaxMemoryCache {
				mock := &mockCache{data: make(map[string]any)}
				cache := NewMaxMemoryCache(100, mock)
				cache.policy = NewLRUPolicy()
				return cache
			},
			key:     "not_exist",
			wantErr: errNotFound,
		},
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
// 参数:
//   - t: 测试上下文
//
// 功能:
//   - 验证Delete方法在不同场景下的行为
//
// 测试用例:
//   - 删除存在的key
//   - 删除不存在的key
func TestMaxMemoryCache_Delete(t *testing.T) {
	testCases := []struct {
		name      string
		setup     func() *MaxMemoryCache
		key       string
		wantErr   error
		wantUsed  int64
		wantExist bool // 检查key是否存在
	}{{
		name: "删除存在的key",
		setup: func() *MaxMemoryCache {
			mock := &mockCache{data: make(map[string]any)}
			cache := NewMaxMemoryCache(100, mock)
			cache.policy = NewLRUPolicy()
			_ = cache.Set(context.Background(), "key1", []byte("value1"), time.Minute) // 6个字节
			_ = cache.Set(context.Background(), "key2", []byte("value2"), time.Minute) // 6个字节
			return cache
		},
		key:       "key1",
		wantErr:   nil,
		wantUsed:  6, // 只剩key2
		wantExist: false,
	}, {
		name: "删除不存在的key",
		setup: func() *MaxMemoryCache {
			mock := &mockCache{data: make(map[string]any)}
			cache := NewMaxMemoryCache(100, mock)
			cache.policy = NewLRUPolicy()
			_ = cache.Set(context.Background(), "key1", []byte("value1"), time.Minute)
			return cache
		},
		key:       "key3",
		wantErr:   nil,
		wantUsed:  6,     // 未删除任何键，保持原样
		wantExist: false, // key3不存在
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cache := tc.setup()
			err := cache.Delete(context.Background(), tc.key)

			// 验证错误
			if tc.wantErr != nil {
				assert.Equal(t, tc.wantErr, err)
			} else {
				assert.Nil(t, err)
			}

			// 验证内存使用量
			assert.Equal(t, tc.wantUsed, cache.used)

			// 验证key是否存在
			_, err = cache.Get(context.Background(), tc.key)
			if tc.wantExist {
				assert.NotEqual(t, errNotFound, err)
			} else {
				assert.Equal(t, errNotFound, err)
			}
		})
	}
}

// TestMaxMemoryCache_Set_Eviction 测试设置时的淘汰逻辑
func TestMaxMemoryCache_Set_Eviction(t *testing.T) {
	tests := []struct {
		name       string
		maxMemory  int64
		operations []struct {
			key   string
			value []byte
		}
		wantUsed      int64
		evictedKey    string
		remainingKeys []string
	}{
		{
			name:      "内存不足触发淘汰",
			maxMemory: 10,
			operations: []struct {
				key   string
				value []byte
			}{
				{"key1", []byte("val1")},
				{"key2", []byte("val2")},
				{"key3", []byte("val3")},
			},
			wantUsed:      8,
			evictedKey:    "key1",
			remainingKeys: []string{"key2", "key3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockCache{data: make(map[string]any)}
			cache := NewMaxMemoryCache(tt.maxMemory, mock)
			cache.policy = NewLRUPolicy()

			for _, op := range tt.operations {
				err := cache.Set(context.Background(), op.key, op.value, time.Minute)
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.wantUsed, cache.used)

			// 验证被淘汰的key
			if tt.evictedKey != "" {
				_, err := cache.Get(context.Background(), tt.evictedKey)
				assert.Equal(t, errNotFound, err)
			}

			// 验证剩余的key
			for _, key := range tt.remainingKeys {
				_, err := cache.Get(context.Background(), key)
				assert.NoError(t, err)
			}
		})
	}
}

// TestMaxMemoryCache_OnEvicted 测试淘汰回调函数是否正确触发
// 参数:
//   - t: 测试上下文
//
// 功能:
//   - 验证淘汰回调函数是否被正确调用
//   - 验证回调参数是否正确
//
// 测试场景:
//   - 单次淘汰回调
//   - 多次淘汰回调
func TestMaxMemoryCache_OnEvicted(t *testing.T) {
	testCases := []struct {
		name        string
		setup       func() *MaxMemoryCache
		keys        []string
		values      [][]byte
		wantEvicted string
		wantValue   []byte
		wantCount   int
		wantUsed    int64
	}{
		{
			name: "LRU策略触发淘汰",
			setup: func() *MaxMemoryCache {
				mock := &mockCache{data: make(map[string]any)}
				cache := NewMaxMemoryCache(10, mock)
				cache.policy = NewLRUPolicy()
				return cache
			},
			keys:        []string{"key1", "key2", "key3"},
			values:      [][]byte{[]byte("val1"), []byte("val2"), []byte("val3")},
			wantEvicted: "key1",
			wantValue:   []byte("val1"),
			wantCount:   1,
			wantUsed:    8,
		},
		{
			name: "多次触发淘汰",
			setup: func() *MaxMemoryCache {
				mock := &mockCache{data: make(map[string]any)}
				cache := NewMaxMemoryCache(5, mock)
				cache.policy = NewLRUPolicy()
				return cache
			},
			keys:        []string{"key1", "key2", "key3", "key4"},
			values:      [][]byte{[]byte("v1"), []byte("v2"), []byte("v3"), []byte("v4")},
			wantEvicted: "key2",
			wantValue:   []byte("v2"),
			wantCount:   2,
			wantUsed:    4,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cache := tc.setup()

			var evictedKey string
			var evictedVal any
			var evictCount int

			cache.OnEvicted(func(key string, val any) {
				evictedKey = key
				evictedVal = val
				evictCount++
			})

			for i, key := range tc.keys {
				_ = cache.Set(context.Background(), key, tc.values[i], time.Minute)
			}

			assert.Equal(t, tc.wantCount, evictCount)
			assert.Equal(t, tc.wantEvicted, evictedKey)
			assert.Equal(t, tc.wantValue, evictedVal)
			assert.Equal(t, tc.wantUsed, cache.used)
		})
	}
}

// TestMaxMemoryCache_TypeAssertionFailure 测试类型断言失败
// 参数:
//   - t: 测试上下文
//
// 功能:
//   - 验证类型断言失败时的错误处理
//
// 测试场景:
//   - Get方法类型断言失败
//   - LoadAndDelete方法类型断言失败
func TestMaxMemoryCache_TypeAssertionFailure(t *testing.T) {
	testCases := []struct {
		name       string
		setup      func() *MaxMemoryCache
		key        string
		wantErrMsg string
	}{
		{
			name: "Get方法类型断言失败",
			setup: func() *MaxMemoryCache {
				mock := &mockCache{
					data: map[string]any{
						"key1": "not a byte slice",
					},
					fn: func(key string, val any) {},
				}
				cache := NewMaxMemoryCache(100, mock)
				cache.policy = NewLRUPolicy()
				_ = cache.policy.KeyAccessed(context.Background(), "key1")
				cache.used = 6
				return cache
			},
			key:        "key1",
			wantErrMsg: "value is not []byte",
		},
		{
			name: "LoadAndDelete方法类型断言失败",
			setup: func() *MaxMemoryCache {
				mock := &mockCache{
					data: map[string]any{
						"key1": 12345, // 测试不同类型的错误值
					},
					fn: func(key string, val any) {},
				}
				cache := NewMaxMemoryCache(100, mock)
				cache.policy = NewLRUPolicy()
				_ = cache.policy.KeyAccessed(context.Background(), "key1")
				cache.used = 6
				return cache
			},
			key:        "key1",
			wantErrMsg: "value is not []byte",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cache := tc.setup()

			val, err := cache.Get(context.Background(), tc.key)
			if assert.NotNil(t, err) {
				assert.Equal(t, tc.wantErrMsg, err.Error())
			}
			assert.Nil(t, val)

			val, err = cache.LoadAndDelete(context.Background(), tc.key)
			if assert.NotNil(t, err) {
				assert.Equal(t, tc.wantErrMsg, err.Error())
			}
			assert.Nil(t, val)
		})
	}
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

// TestMaxMemoryCache_ConstructorMethods 测试不同的构造函数方法
func TestMaxMemoryCache_ConstructorMethods(t *testing.T) {
	tests := []struct {
		name        string
		constructor func(int64, interfaces.Cache) *MaxMemoryCache
		maxMemory   int64
	}{
		{
			name:        "NewMaxMemoryCacheWithLRU",
			constructor: NewMaxMemoryCacheWithLRU,
			maxMemory:   1024,
		},
		{
			name:        "NewMaxMemoryCacheWithFIFO",
			constructor: NewMaxMemoryCacheWithFIFO,
			maxMemory:   1024,
		},
		{
			name:        "NewMaxMemoryCacheWithRandom",
			constructor: NewMaxMemoryCacheWithRandom,
			maxMemory:   1024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewBuildInMapCache(time.Minute)
			maxCache := tt.constructor(tt.maxMemory, cache)

			assert.NotNil(t, maxCache)
			assert.Equal(t, tt.maxMemory, maxCache.max)
		})
	}
}

// TestMaxMemoryCache_LoadAndDelete_Error 测试LoadAndDelete的错误情况
func TestMaxMemoryCache_LoadAndDelete_Error(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
		wantVal any
	}{
		{
			name:    "不存在的key",
			key:     "not_exist",
			wantErr: true,
			wantVal: []byte(nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewBuildInMapCache(time.Minute)
			maxCache := NewMaxMemoryCache(1024, cache, NewLRUPolicy())

			val, err := maxCache.LoadAndDelete(context.Background(), tt.key)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantVal, val)
		})
	}
}

// TestMaxMemoryCache_NilCache 测试nil缓存的情况
func TestMaxMemoryCache_NilCache(t *testing.T) {
	tests := []struct {
		name      string
		maxMemory int64
		cache     interfaces.Cache
		policy    EvictionPolicy
	}{
		{
			name:      "nil缓存",
			maxMemory: 1024,
			cache:     nil,
			policy:    NewLRUPolicy(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			maxCache := NewMaxMemoryCache(tt.maxMemory, tt.cache, tt.policy)
			assert.NotNil(t, maxCache)
			assert.Equal(t, tt.cache, maxCache.Cache)
		})
	}
}

// TestMaxMemoryCache_Set_PolicyError 测试策略操作失败的情况
func TestMaxMemoryCache_Set_PolicyError(t *testing.T) {
	tests := []struct {
		name      string
		maxMemory int64
		values    [][]byte
		wantUsed  func(int64, int64) bool
	}{
		{
			name:      "设置大值触发淘汰",
			maxMemory: 10,
			values:    [][]byte{[]byte("12345"), []byte("67890")},
			wantUsed:  func(used, max int64) bool { return used <= max },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewBuildInMapCache(time.Minute)
			maxCache := NewMaxMemoryCache(tt.maxMemory, cache, NewLRUPolicy())

			for i, value := range tt.values {
				err := maxCache.Set(context.Background(), fmt.Sprintf("key%d", i+1), value, time.Minute)
				assert.NoError(t, err)
			}

			assert.True(t, tt.wantUsed(maxCache.used, maxCache.max))
		})
	}
}

// TestMaxMemoryCache_LoadAndDelete_TypeAssertion 测试LoadAndDelete的类型断言
func TestMaxMemoryCache_LoadAndDelete_TypeAssertion(t *testing.T) {
	tests := []struct {
		name       string
		setupValue any
		wantErr    string
		wantVal    any
	}{
		{
			name:       "非[]byte类型返回错误",
			setupValue: "string_value",
			wantErr:    "value is not []byte",
			wantVal:    []byte(nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewBuildInMapCache(time.Minute)
			maxCache := NewMaxMemoryCache(1024, cache, NewLRUPolicy())

			err := cache.Set(context.Background(), "key1", tt.setupValue, time.Minute)
			assert.NoError(t, err)

			val, err := maxCache.LoadAndDelete(context.Background(), "key1")
			assert.Error(t, err)
			assert.Equal(t, tt.wantErr, err.Error())
			assert.Equal(t, tt.wantVal, val)
		})
	}
}

// TestMaxMemoryCache_Evicted_TypeAssertion 测试evicted方法的类型断言
func TestMaxMemoryCache_Evicted_TypeAssertion(t *testing.T) {
	tests := []struct {
		name       string
		setupValue any
		evictValue any
		wantUsed   int64
	}{
		{
			name:       "非[]byte类型不减少内存使用",
			setupValue: "string_value",
			evictValue: "string_value",
			wantUsed:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewBuildInMapCache(time.Minute)
			maxCache := NewMaxMemoryCache(1024, cache, NewLRUPolicy())

			err := cache.Set(context.Background(), "key1", tt.setupValue, time.Minute)
			assert.NoError(t, err)

			maxCache.evicted("key1", tt.evictValue)

			assert.Equal(t, tt.wantUsed, maxCache.used)
		})
	}
}

// TestMaxMemoryCache_Set_EvictionError 测试淘汰过程中的错误处理
func TestMaxMemoryCache_Set_EvictionError(t *testing.T) {
	cache := NewBuildInMapCache(time.Minute)
	maxCache := NewMaxMemoryCache(5, cache, NewLRUPolicy())

	// 设置一个值
	err := maxCache.Set(context.Background(), "key1", []byte("12345"), time.Minute)
	assert.Nil(t, err)

	// 设置另一个值，触发淘汰
	err = maxCache.Set(context.Background(), "key2", []byte("67890"), time.Minute)
	assert.Nil(t, err)

	// 验证内存使用在限制内
	assert.LessOrEqual(t, maxCache.used, maxCache.max)
}

// TestMaxMemoryCache_Set_CacheSetError 测试底层缓存设置失败
func TestMaxMemoryCache_Set_CacheSetError(t *testing.T) {
	cache := &MockCache{store: make(map[string]any), setShouldFail: true}
	maxCache := NewMaxMemoryCache(1024, cache, NewLRUPolicy())

	// 设置值应该失败
	err := maxCache.Set(context.Background(), "key1", []byte("value1"), time.Minute)
	assert.Error(t, err)
	assert.Equal(t, "mock set error", err.Error())

	// 内存使用应该为0
	assert.Equal(t, int64(0), maxCache.used)
}

// TestMaxMemoryCache_LoadAndDelete_EvictedUpdate 测试LoadAndDelete时的内存更新
func TestMaxMemoryCache_LoadAndDelete_EvictedUpdate(t *testing.T) {
	cache := NewBuildInMapCache(time.Minute)
	maxCache := NewMaxMemoryCache(1024, cache, NewLRUPolicy())

	// 设置一个值
	err := maxCache.Set(context.Background(), "key1", []byte("value1"), time.Minute)
	assert.Nil(t, err)

	initialUsed := maxCache.used

	// LoadAndDelete应该减少内存使用
	val, err := maxCache.LoadAndDelete(context.Background(), "key1")
	assert.Nil(t, err)
	assert.Equal(t, []byte("value1"), val)

	// 验证内存使用减少
	assert.Less(t, maxCache.used, initialUsed)
}
