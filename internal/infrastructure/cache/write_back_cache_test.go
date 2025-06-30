package cache

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockStorer 模拟存储器，用于测试写回缓存
type MockStorer struct {
	data       map[string]any
	mu         sync.RWMutex
	failKeys   map[string]bool // 模拟失败的键
	storeDelay time.Duration   // 模拟存储延迟
	storeCalls []StoreCall     // 记录存储调用
}

// StoreCall 存储调用记录
type StoreCall struct {
	Key   string
	Value any
	Time  time.Time
}

// NewMockStorer 创建模拟存储器
func NewMockStorer() *MockStorer {
	return &MockStorer{
		data:       make(map[string]any),
		failKeys:   make(map[string]bool),
		storeCalls: make([]StoreCall, 0),
	}
}

// Store 模拟存储操作
func (m *MockStorer) Store(ctx context.Context, key string, val any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 记录调用
	m.storeCalls = append(m.storeCalls, StoreCall{
		Key:   key,
		Value: val,
		Time:  time.Now(),
	})

	// 模拟延迟
	if m.storeDelay > 0 {
		time.Sleep(m.storeDelay)
	}

	// 模拟失败
	if m.failKeys[key] {
		return errors.New("模拟存储失败")
	}

	m.data[key] = val
	return nil
}

// SetFailKey 设置失败的键
func (m *MockStorer) SetFailKey(key string, fail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failKeys[key] = fail
}

// SetStoreDelay 设置存储延迟
func (m *MockStorer) SetStoreDelay(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.storeDelay = delay
}

// GetStoreCalls 获取存储调用记录
func (m *MockStorer) GetStoreCalls() []StoreCall {
	m.mu.RLock()
	defer m.mu.RUnlock()
	calls := make([]StoreCall, len(m.storeCalls))
	copy(calls, m.storeCalls)
	return calls
}

// GetStoreCallCount 获取存储调用次数
func (m *MockStorer) GetStoreCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.storeCalls)
}

// ClearStoreCalls 清空存储调用记录
func (m *MockStorer) ClearStoreCalls() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.storeCalls = m.storeCalls[:0]
}

// TestWriteBackCache_SetDirty 测试设置脏数据
func TestWriteBackCache_SetDirty(t *testing.T) {
	tests := []struct {
		name        string
		setupCache  func() (*WriteBackCache, *MockStorer)
		key         string
		value       any
		expiration  time.Duration
		wantErr     bool
		wantInCache bool
		wantInStore bool
		wantDirty   bool
	}{
		{
			name: "设置脏数据_成功",
			setupCache: func() (*WriteBackCache, *MockStorer) {
				mockCache := &MockCache{store: make(map[string]any)}
				mockStorer := NewMockStorer()
				cache := NewWriteBackCache(mockCache, time.Minute, 10)
				return cache, mockStorer
			},
			key:         "key1",
			value:       "value1",
			expiration:  time.Minute,
			wantErr:     false,
			wantInCache: true,
			wantInStore: false, // 写回模式不立即写入存储
			wantDirty:   true,
		},
		{
			name: "设置脏数据_缓存失败",
			setupCache: func() (*WriteBackCache, *MockStorer) {
				mockCache := &MockCache{store: make(map[string]any), setShouldFail: true}
				mockStorer := NewMockStorer()
				cache := NewWriteBackCache(mockCache, time.Minute, 10)
				return cache, mockStorer
			},
			key:         "key1",
			value:       "value1",
			expiration:  time.Minute,
			wantErr:     true,
			wantInCache: false,
			wantInStore: false,
			wantDirty:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache, storer := tt.setupCache()

			err := cache.SetDirty(context.Background(), tt.key, tt.value, tt.expiration)

			// 检查错误
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// 检查缓存中是否存在
			if tt.wantInCache {
				val, err := cache.Get(context.Background(), tt.key)
				assert.NoError(t, err)
				assert.Equal(t, tt.value, val)
			}

			// 检查是否立即写入存储（写回模式不应该立即写入）
			if !tt.wantInStore {
				assert.Equal(t, 0, storer.GetStoreCallCount())
			}
		})
	}
}

// TestWriteBackCache_FlushKey 测试刷新单个键
func TestWriteBackCache_FlushKey(t *testing.T) {
	tests := []struct {
		name           string
		setupCache     func() (*WriteBackCache, *MockStorer)
		setupData      func(*WriteBackCache)
		flushKey       string
		wantErr        bool
		wantStoreCount int
		wantDirtyAfter bool
	}{
		{
			name: "刷新单个键_成功",
			setupCache: func() (*WriteBackCache, *MockStorer) {
				mockCache := &MockCache{store: make(map[string]any)}
				mockStorer := NewMockStorer()
				cache := NewWriteBackCache(mockCache, time.Minute, 10)
				return cache, mockStorer
			},
			setupData: func(cache *WriteBackCache) {
				_ = cache.SetDirty(context.Background(), "key1", "value1", time.Minute)
				_ = cache.SetDirty(context.Background(), "key2", "value2", time.Minute)
			},
			flushKey:       "key1",
			wantErr:        false,
			wantStoreCount: 1,
			wantDirtyAfter: false,
		},
		{
			name: "刷新不存在的键",
			setupCache: func() (*WriteBackCache, *MockStorer) {
				mockCache := &MockCache{store: make(map[string]any)}
				mockStorer := NewMockStorer()
				cache := NewWriteBackCache(mockCache, time.Minute, 10)
				return cache, mockStorer
			},
			setupData: func(cache *WriteBackCache) {
				// 不设置任何数据
			},
			flushKey:       "nonexistent",
			wantErr:        true,
			wantStoreCount: 0,
			wantDirtyAfter: false,
		},
		{
			name: "刷新键_存储失败",
			setupCache: func() (*WriteBackCache, *MockStorer) {
				mockCache := &MockCache{store: make(map[string]any)}
				mockStorer := NewMockStorer()
				mockStorer.SetFailKey("key1", true)
				cache := NewWriteBackCache(mockCache, time.Minute, 10)
				return cache, mockStorer
			},
			setupData: func(cache *WriteBackCache) {
				_ = cache.SetDirty(context.Background(), "key1", "value1", time.Minute)
			},
			flushKey:       "key1",
			wantErr:        true,
			wantStoreCount: 1,
			wantDirtyAfter: true, // 存储失败，应该保持脏状态
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache, storer := tt.setupCache()
			tt.setupData(cache)

			err := cache.FlushKey(context.Background(), tt.flushKey, storer.Store)

			// 检查错误
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// 检查存储调用次数
			assert.Equal(t, tt.wantStoreCount, storer.GetStoreCallCount())
		})
	}
}

// TestWriteBackCache_Flush 测试批量刷新
func TestWriteBackCache_Flush(t *testing.T) {
	tests := []struct {
		name           string
		setupCache     func() (*WriteBackCache, *MockStorer)
		setupData      func(*WriteBackCache)
		wantErr        bool
		wantStoreCount int
	}{
		{
			name: "批量刷新_成功",
			setupCache: func() (*WriteBackCache, *MockStorer) {
				mockCache := &MockCache{store: make(map[string]any)}
				mockStorer := NewMockStorer()
				cache := NewWriteBackCache(mockCache, time.Minute, 10)
				return cache, mockStorer
			},
			setupData: func(cache *WriteBackCache) {
				_ = cache.SetDirty(context.Background(), "key1", "value1", time.Minute)
				_ = cache.SetDirty(context.Background(), "key2", "value2", time.Minute)
				_ = cache.SetDirty(context.Background(), "key3", "value3", time.Minute)
			},
			wantErr:        false,
			wantStoreCount: 3,
		},
		{
			name: "批量刷新_无脏数据",
			setupCache: func() (*WriteBackCache, *MockStorer) {
				mockCache := &MockCache{store: make(map[string]any)}
				mockStorer := NewMockStorer()
				cache := NewWriteBackCache(mockCache, time.Minute, 10)
				return cache, mockStorer
			},
			setupData: func(cache *WriteBackCache) {
				// 不设置脏数据
			},
			wantErr:        false,
			wantStoreCount: 0,
		},
		{
			name: "批量刷新_部分失败",
			setupCache: func() (*WriteBackCache, *MockStorer) {
				mockCache := &MockCache{store: make(map[string]any)}
				mockStorer := NewMockStorer()
				mockStorer.SetFailKey("key2", true) // key2存储失败
				cache := NewWriteBackCache(mockCache, time.Minute, 10)
				return cache, mockStorer
			},
			setupData: func(cache *WriteBackCache) {
				_ = cache.SetDirty(context.Background(), "key1", "value1", time.Minute)
				_ = cache.SetDirty(context.Background(), "key2", "value2", time.Minute)
				_ = cache.SetDirty(context.Background(), "key3", "value3", time.Minute)
			},
			wantErr:        true, // 部分失败应该返回错误
			wantStoreCount: 3,    // 但所有键都会尝试存储
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache, storer := tt.setupCache()
			tt.setupData(cache)

			err := cache.Flush(context.Background(), storer.Store)

			// 检查错误
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// 检查存储调用次数
			assert.Equal(t, tt.wantStoreCount, storer.GetStoreCallCount())
		})
	}
}

// TestWriteBackCache_AutoFlush 测试自动刷新
func TestWriteBackCache_AutoFlush(t *testing.T) {
	t.Run("定时自动刷新", func(t *testing.T) {
		mockCache := &MockCache{store: make(map[string]any)}
		mockStorer := NewMockStorer()

		// 设置较短的刷新间隔用于测试
		cache := NewWriteBackCache(mockCache, 100*time.Millisecond, 10)

		// 设置脏数据
		err := cache.SetDirty(context.Background(), "key1", "value1", time.Minute)
		require.NoError(t, err)

		// 启动自动刷新
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go cache.StartAutoFlush(ctx, mockStorer.Store)

		// 等待自动刷新触发
		time.Sleep(200 * time.Millisecond)

		// 检查是否已刷新
		assert.Greater(t, mockStorer.GetStoreCallCount(), 0)
	})

	t.Run("批量大小触发刷新", func(t *testing.T) {
		mockCache := &MockCache{store: make(map[string]any)}
		mockStorer := NewMockStorer()

		// 设置较小的批量大小
		cache := NewWriteBackCache(mockCache, time.Hour, 2)

		// 启动自动刷新
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go cache.StartAutoFlush(ctx, mockStorer.Store)

		// 设置脏数据，达到批量大小
		_ = cache.SetDirty(context.Background(), "key1", "value1", time.Minute)
		_ = cache.SetDirty(context.Background(), "key2", "value2", time.Minute)

		// 等待自动刷新触发
		time.Sleep(100 * time.Millisecond)

		// 检查是否已刷新
		assert.Equal(t, 2, mockStorer.GetStoreCallCount())
	})
}

// TestWriteBackCache_ConcurrentOperations 测试并发操作
func TestWriteBackCache_ConcurrentOperations(t *testing.T) {
	mockCache := &MockCache{store: make(map[string]any)}
	mockStorer := NewMockStorer()
	cache := NewWriteBackCache(mockCache, time.Second, 100)

	const numGoroutines = 10
	const numOperations = 100

	var wg sync.WaitGroup

	// 并发写入
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key_%d_%d", id, j)
				value := fmt.Sprintf("value_%d_%d", id, j)
				_ = cache.SetDirty(context.Background(), key, value, time.Minute)
			}
		}(i)
	}

	wg.Wait()

	// 检查脏数据数量
	expectedCount := numGoroutines * numOperations
	assert.Equal(t, expectedCount, cache.GetDirtyCount())

	// 刷新所有数据
	err := cache.Flush(context.Background(), mockStorer.Store)
	assert.NoError(t, err)

	// 检查存储调用次数
	assert.Equal(t, expectedCount, mockStorer.GetStoreCallCount())

	// 检查脏数据已清空
	assert.Equal(t, 0, cache.GetDirtyCount())
}

// TestWriteBackCache_EvictionHandling 测试淘汰处理
func TestWriteBackCache_EvictionHandling(t *testing.T) {
	mockCache := &MockCache{store: make(map[string]any)}
	cache := NewWriteBackCache(mockCache, time.Hour, 10)

	evictedKeys := make([]string, 0)
	evictedValues := make([]any, 0)

	// 设置淘汰回调
	cache.OnEvicted(func(key string, val any) {
		evictedKeys = append(evictedKeys, key)
		evictedValues = append(evictedValues, val)
	})

	// 设置脏数据
	_ = cache.SetDirty(context.Background(), "key1", "value1", time.Minute)

	// 验证脏数据存在
	assert.Equal(t, 1, cache.GetDirtyCount())

	// 手动触发淘汰回调来测试
	mockCache.mu.RLock()
	evictedFn := mockCache.evictedFn
	mockCache.mu.RUnlock()

	if evictedFn != nil {
		evictedFn("key1", "value1")
	}

	// 检查脏数据是否被清理
	assert.Equal(t, 0, cache.GetDirtyCount())

	// 检查回调是否被调用
	assert.Equal(t, 1, len(evictedKeys))
	assert.Equal(t, "key1", evictedKeys[0])
	assert.Equal(t, "value1", evictedValues[0])
}

// TestWriteBackCache_ErrorHandling 测试错误处理
func TestWriteBackCache_ErrorHandling(t *testing.T) {
	tests := []struct {
		name       string
		setupCache func() (*WriteBackCache, *MockStorer)
		operation  func(*WriteBackCache, *MockStorer) error
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "刷新不存在的键",
			setupCache: func() (*WriteBackCache, *MockStorer) {
				mockCache := &MockCache{store: make(map[string]any)}
				mockStorer := NewMockStorer()
				cache := NewWriteBackCache(mockCache, time.Hour, 10)
				return cache, mockStorer
			},
			operation: func(cache *WriteBackCache, storer *MockStorer) error {
				return cache.FlushKey(context.Background(), "nonexistent", storer.Store)
			},
			wantErr:    true,
			wantErrMsg: "不存在或不是脏数据",
		},
		{
			name: "存储函数为nil",
			setupCache: func() (*WriteBackCache, *MockStorer) {
				mockCache := &MockCache{store: make(map[string]any)}
				mockStorer := NewMockStorer()
				cache := NewWriteBackCache(mockCache, time.Hour, 10)
				_ = cache.SetDirty(context.Background(), "key1", "value1", time.Minute)
				return cache, mockStorer
			},
			operation: func(cache *WriteBackCache, storer *MockStorer) error {
				return cache.FlushKey(context.Background(), "key1", nil)
			},
			wantErr:    true,
			wantErrMsg: "runtime error", // nil函数调用会导致panic
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache, storer := tt.setupCache()

			var err error
			if tt.wantErrMsg == "runtime error" {
				// 捕获panic
				defer func() {
					if r := recover(); r != nil {
						err = fmt.Errorf("runtime error: %v", r)
					}
				}()
			}

			if err == nil {
				err = tt.operation(cache, storer)
			}

			if tt.wantErr {
				assert.Error(t, err)
				if tt.wantErrMsg != "" && tt.wantErrMsg != "runtime error" {
					assert.Contains(t, err.Error(), tt.wantErrMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
