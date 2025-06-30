package cache

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/singleflight"
)

// MockCache 实现完整的 Cache 接口
type MockCache struct {
	store         map[string]any
	getShouldFail bool
	setShouldFail bool
	evictedFn     func(string, any)
	mu            sync.RWMutex
}

func (m *MockCache) Get(_ context.Context, key string) (any, error) {
	if m.getShouldFail {
		return nil, errors.New("mock get error")
	}
	m.mu.RLock()
	val, ok := m.store[key]
	m.mu.RUnlock()
	if !ok {
		return nil, ErrKeyNotFound
	}
	return val, nil
}

func (m *MockCache) Set(_ context.Context, key string, value any, _ time.Duration) error {
	if m.setShouldFail {
		return errors.New("mock set error")
	}
	m.mu.Lock()
	m.store[key] = value
	m.mu.Unlock()
	return nil
}

func (m *MockCache) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	delete(m.store, key)
	m.mu.Unlock()
	return nil
}

func (m *MockCache) LoadAndDelete(ctx context.Context, key string) (any, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	val, ok := m.store[key]
	if !ok {
		return nil, ErrKeyNotFound
	}
	delete(m.store, key)
	return val, nil
}

func (m *MockCache) OnEvicted(fn func(string, any)) {
	m.mu.Lock()
	m.evictedFn = fn
	m.mu.Unlock()
}

// TestReadThroughCache_Get 测试ReadThroughCache的Get方法
// 参数:
//   - t: 测试上下文
//
// 测试场景:
//   - 缓存命中
//   - 缓存未命中_加载成功
//   - 缓存未命中_加载失败
//   - 缓存未命中_设置缓存失败
func TestReadThroughCache_Get(t *testing.T) {
	tests := []struct {
		name          string
		setupCache    func() *MockCache
		setupLoadFunc func() func(ctx context.Context, key string) (any, error)
		setupLogFunc  func() (func(format string, args ...any), *bool)
		key           string
		wantValue     any
		wantErr       error
		wantLogCalled *bool
		checkCacheSet bool
	}{
		{
			name: "缓存命中",
			setupCache: func() *MockCache {
				return &MockCache{store: map[string]any{"key1": "value1"}}
			},
			setupLoadFunc: func() func(ctx context.Context, key string) (any, error) {
				return func(ctx context.Context, key string) (any, error) { return nil, nil }
			},
			setupLogFunc: func() (func(format string, args ...any), *bool) {
				return nil, nil
			},
			key:           "key1",
			wantValue:     "value1",
			wantErr:       nil,
			checkCacheSet: false,
		},
		{
			name: "缓存未命中_加载成功",
			setupCache: func() *MockCache {
				return &MockCache{store: make(map[string]any)}
			},
			setupLoadFunc: func() func(ctx context.Context, key string) (any, error) {
				return func(ctx context.Context, key string) (any, error) {
					return "loaded_value", nil
				}
			},
			setupLogFunc: func() (func(format string, args ...any), *bool) {
				return nil, nil
			},
			key:           "key1",
			wantValue:     "loaded_value",
			wantErr:       nil,
			checkCacheSet: true,
		},
		{
			name: "缓存未命中_加载失败",
			setupCache: func() *MockCache {
				return &MockCache{store: make(map[string]any)}
			},
			setupLoadFunc: func() func(ctx context.Context, key string) (any, error) {
				return func(ctx context.Context, key string) (any, error) {
					return nil, errors.New("load failed")
				}
			},
			setupLogFunc: func() (func(format string, args ...any), *bool) {
				return nil, nil
			},
			key:           "key1",
			wantValue:     nil,
			wantErr:       errors.New("load failed"),
			checkCacheSet: false,
		},
		{
			name: "缓存未命中_设置缓存失败",
			setupCache: func() *MockCache {
				return &MockCache{store: make(map[string]any), setShouldFail: true}
			},
			setupLoadFunc: func() func(ctx context.Context, key string) (any, error) {
				return func(ctx context.Context, key string) (any, error) {
					return "loaded_value", nil
				}
			},
			setupLogFunc: func() (func(format string, args ...any), *bool) {
				logCalled := false
				logFunc := func(format string, args ...any) {
					logCalled = true
				}
				return logFunc, &logCalled
			},
			key:           "key1",
			wantValue:     "loaded_value",
			wantErr:       ErrFailedToRefreshCache,
			wantLogCalled: func() *bool { b := true; return &b }(),
			checkCacheSet: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCache := tt.setupCache()
			loadFunc := tt.setupLoadFunc()
			logFunc, logCalled := tt.setupLogFunc()

			rtCache := &ReadThroughCache{
				Repository: mockCache,
				LoadFunc:   loadFunc,
				Expiration: time.Minute,
				logFunc:    logFunc,
			}

			val, err := rtCache.Get(context.Background(), tt.key)

			// 检查错误
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("Expected error %v, got nil", tt.wantErr)
				} else if !errors.Is(err, tt.wantErr) && err.Error() != tt.wantErr.Error() {
					t.Errorf("Expected error %v, got %v", tt.wantErr, err)
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// 检查返回值
			if val != tt.wantValue {
				t.Errorf("Expected value %v, got %v", tt.wantValue, val)
			}

			// 检查日志是否被调用
			if tt.wantLogCalled != nil && logCalled != nil {
				if *logCalled != *tt.wantLogCalled {
					t.Errorf("Expected log called %v, got %v", *tt.wantLogCalled, *logCalled)
				}
			}

			// 检查缓存是否已设置
			if tt.checkCacheSet && err == nil {
				cachedVal, _ := mockCache.Get(context.Background(), tt.key)
				if cachedVal != tt.wantValue {
					t.Errorf("Expected cached value %v, got %v", tt.wantValue, cachedVal)
				}
			}
		})
	}
}

// TestSingleFlight 测试singleflight防止缓存击穿功能
// 参数:
//   - t: 测试上下文
//
// 测试场景:
//   - ReadThroughCache_SingleFlight防止缓存击穿
//   - RateLimitReadThroughCache_SingleFlight防止缓存击穿
func TestSingleFlight(t *testing.T) {
	tests := []struct {
		name              string
		cacheType         string
		concurrentCalls   int
		expectedLoadCalls int
		expectedValue     any
		simulateDelay     time.Duration
	}{
		{
			name:              "ReadThroughCache_SingleFlight防止缓存击穿",
			cacheType:         "ReadThroughCache",
			concurrentCalls:   5,
			expectedLoadCalls: 1,
			expectedValue:     "loaded_value",
			simulateDelay:     100 * time.Millisecond,
		},
		{
			name:              "RateLimitReadThroughCache_SingleFlight防止缓存击穿",
			cacheType:         "RateLimitReadThroughCache",
			concurrentCalls:   5,
			expectedLoadCalls: 1,
			expectedValue:     "loaded_value",
			simulateDelay:     100 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCache := &MockCache{store: make(map[string]any)}
			loadCount := 0

			loadFunc := func(ctx context.Context, key string) (any, error) {
				loadCount++
				time.Sleep(tt.simulateDelay) // 模拟耗时操作
				return tt.expectedValue, nil
			}

			var cache interface {
				Get(ctx context.Context, key string) (any, error)
			}

			switch tt.cacheType {
			case "ReadThroughCache":
				cache = &ReadThroughCache{
					Repository: mockCache,
					LoadFunc:   loadFunc,
					Expiration: time.Minute,
					g:          singleflight.Group{},
				}
			case "RateLimitReadThroughCache":
				cache = &RateLimitReadThroughCache{
					Repository: mockCache,
					LoadFunc:   loadFunc,
					Expiration: time.Minute,
					g:          singleflight.Group{},
				}
			}

			// 启动并发请求
			results := make(chan any, tt.concurrentCalls)
			for i := 0; i < tt.concurrentCalls; i++ {
				go func() {
					val, err := cache.Get(context.Background(), "key1")
					if err != nil {
						results <- err
					} else {
						results <- val
					}
				}()
			}

			// 收集结果
			for i := 0; i < tt.concurrentCalls; i++ {
				res := <-results
				if res != tt.expectedValue {
					t.Errorf("Expected %v, got %v", tt.expectedValue, res)
				}
			}

			// 验证加载函数只被调用了预期的次数
			if loadCount != tt.expectedLoadCalls {
				t.Errorf("Expected load function to be called %d times, got %d times", tt.expectedLoadCalls, loadCount)
			}
		})
	}
}

// TestReadThroughCache_SetLogFunc 测试设置日志函数的功能
// 参数:
//
//	t *testing.T - 测试上下文
//
// 测试场景:
//  1. 验证日志函数能被正确设置和调用
func TestReadThroughCache_SetLogFunc(t *testing.T) {
	tests := []struct {
		name        string
		setupCache  func() *ReadThroughCache
		testLogFunc func(format string, args ...any)
		wantCalled  bool
	}{
		{
			name: "设置日志函数_正常调用",
			setupCache: func() *ReadThroughCache {
				return &ReadThroughCache{}
			},
			testLogFunc: func(format string, args ...any) {
				// 测试日志函数被正确调用
			},
			wantCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rtCache := tt.setupCache()
			logCalled := false

			// 设置日志函数
			rtCache.SetLogFunc(func(format string, args ...any) {
				logCalled = true
				if tt.testLogFunc != nil {
					tt.testLogFunc(format, args...)
				}
			})

			// 调用日志函数
			if rtCache.logFunc != nil {
				rtCache.logFunc("test log")
			}

			// 验证日志函数是否被调用
			if logCalled != tt.wantCalled {
				t.Errorf("Expected log called %v, got %v", tt.wantCalled, logCalled)
			}
		})
	}
}

// TestRateLimitReadThroughCache_Get 测试带限流功能的读穿透缓存的Get方法
// 参数:
//
//	t *testing.T - 测试上下文
//
// 测试场景:
//  1. 缓存命中
//  2. 缓存未命中且未限流时加载成功
//  3. 缓存未命中且限流时不加载
//  4. 缓存未命中且加载失败
//  5. 缓存未命中但设置缓存失败
func TestRateLimitReadThroughCache_Get(t *testing.T) {
	tests := []struct {
		name              string
		setupCache        func() *MockCache
		setupLoadFunc     func() (func(ctx context.Context, key string) (any, error), *int)
		setupContext      func() context.Context
		key               string
		wantValue         any
		wantErr           error
		wantLoadCallCount int
		checkCacheSet     bool
	}{
		{
			name: "缓存命中",
			setupCache: func() *MockCache {
				return &MockCache{store: map[string]any{"key1": "value1"}}
			},
			setupLoadFunc: func() (func(ctx context.Context, key string) (any, error), *int) {
				return func(ctx context.Context, key string) (any, error) {
					return nil, errors.New("should not be called")
				}, nil
			},
			setupContext: func() context.Context {
				return context.Background()
			},
			key:               "key1",
			wantValue:         "value1",
			wantErr:           nil,
			wantLoadCallCount: 0,
			checkCacheSet:     false,
		},
		{
			name: "缓存未命中_未限流_加载成功",
			setupCache: func() *MockCache {
				return &MockCache{store: make(map[string]any)}
			},
			setupLoadFunc: func() (func(ctx context.Context, key string) (any, error), *int) {
				loadCount := 0
				loadFunc := func(ctx context.Context, key string) (any, error) {
					loadCount++
					return "loaded_value", nil
				}
				return loadFunc, &loadCount
			},
			setupContext: func() context.Context {
				return context.Background()
			},
			key:               "key1",
			wantValue:         "loaded_value",
			wantErr:           nil,
			wantLoadCallCount: 1,
			checkCacheSet:     true,
		},
		{
			name: "缓存未命中_限流_不加载",
			setupCache: func() *MockCache {
				return &MockCache{store: make(map[string]any)}
			},
			setupLoadFunc: func() (func(ctx context.Context, key string) (any, error), *int) {
				loadCount := 0
				loadFunc := func(ctx context.Context, key string) (any, error) {
					loadCount++
					return "loaded_value", nil
				}
				return loadFunc, &loadCount
			},
			setupContext: func() context.Context {
				return context.WithValue(context.Background(), "limited", true)
			},
			key:               "key1",
			wantValue:         nil,
			wantErr:           ErrKeyNotFound,
			wantLoadCallCount: 0,
			checkCacheSet:     false,
		},
		{
			name: "缓存未命中_加载失败",
			setupCache: func() *MockCache {
				return &MockCache{store: make(map[string]any)}
			},
			setupLoadFunc: func() (func(ctx context.Context, key string) (any, error), *int) {
				return func(ctx context.Context, key string) (any, error) {
					return nil, errors.New("load failed")
				}, nil
			},
			setupContext: func() context.Context {
				return context.Background()
			},
			key:               "key1",
			wantValue:         nil,
			wantErr:           errors.New("load failed"),
			wantLoadCallCount: 0, // 不检查计数，因为没有计数器
			checkCacheSet:     false,
		},
		{
			name: "缓存未命中_设置缓存失败",
			setupCache: func() *MockCache {
				return &MockCache{store: make(map[string]any), setShouldFail: true}
			},
			setupLoadFunc: func() (func(ctx context.Context, key string) (any, error), *int) {
				return func(ctx context.Context, key string) (any, error) {
					return "loaded_value", nil
				}, nil
			},
			setupContext: func() context.Context {
				return context.Background()
			},
			key:               "key1",
			wantValue:         "loaded_value",
			wantErr:           ErrFailedToRefreshCache,
			wantLoadCallCount: 0, // 不检查计数，因为没有计数器
			checkCacheSet:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCache := tt.setupCache()
			loadFunc, loadCount := tt.setupLoadFunc()
			ctx := tt.setupContext()

			rlCache := &RateLimitReadThroughCache{
				Repository: mockCache,
				LoadFunc:   loadFunc,
				Expiration: time.Minute,
			}

			val, err := rlCache.Get(ctx, tt.key)

			// 检查错误
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("Expected error %v, got nil", tt.wantErr)
				} else if !errors.Is(err, tt.wantErr) && err.Error() != tt.wantErr.Error() {
					t.Errorf("Expected error %v, got %v", tt.wantErr, err)
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// 检查返回值
			if val != tt.wantValue {
				t.Errorf("Expected value %v, got %v", tt.wantValue, val)
			}

			// 检查加载函数调用次数
			if loadCount != nil && *loadCount != tt.wantLoadCallCount {
				t.Errorf("Expected LoadFunc to be called %d times, got %d times", tt.wantLoadCallCount, *loadCount)
			}

			// 检查缓存是否已设置
			if tt.checkCacheSet && err == nil {
				cachedVal, _ := mockCache.Get(ctx, tt.key)
				if cachedVal != tt.wantValue {
					t.Errorf("Expected cached value %v, got %v", tt.wantValue, cachedVal)
				}
			}
		})
	}
}

// TestReadThroughCache_Get_CacheError 测试缓存获取时的错误情况
func TestReadThroughCache_Get_CacheError(t *testing.T) {
	mockCache := &MockCache{store: make(map[string]any), getShouldFail: true}
	rtCache := &ReadThroughCache{
		Repository: mockCache,
		LoadFunc: func(ctx context.Context, key string) (any, error) {
			return "loaded_value", nil
		},
		Expiration: time.Minute,
	}

	val, err := rtCache.Get(context.Background(), "key1")
	assert.Error(t, err)
	assert.Nil(t, val)
	assert.Equal(t, "mock get error", err.Error())
}
