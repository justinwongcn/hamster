package cache

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var implErrKeyNotFound = ErrCacheKeyNotFound

func TestBuildInMapCache(t *testing.T) {
	tests := []struct {
		name        string
		operation   func(*BuildInMapCache) error
		wantErr     bool
		wantErrType error
		checkResult func(*testing.T, *BuildInMapCache)
	}{
		{
			name: "基本设置和获取",
			operation: func(c *BuildInMapCache) error {
				return c.Set(context.Background(), "key1", "value1", time.Minute)
			},
			checkResult: func(t *testing.T, c *BuildInMapCache) {
				val, err := c.Get(context.Background(), "key1")
				assert.Nil(t, err)
				assert.Equal(t, "value1", val)
			},
		},
		{
			name: "获取不存在的key",
			operation: func(c *BuildInMapCache) error {
				_, err := c.Get(context.Background(), "not_exist")
				return err
			},
			wantErr:     true,
			wantErrType: ErrCacheKeyNotFound,
		},
		// 可以继续添加更多测试用例
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewBuildInMapCache(time.Minute)
			err := tt.operation(c)

			if tt.wantErr {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantErrType))
			} else {
				assert.NoError(t, err)
			}

			if tt.checkResult != nil {
				tt.checkResult(t, c)
			}
		})
	}
}

// TestBuildInMapCache_Expiration 测试缓存的过期功能
func TestBuildInMapCache_Expiration(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		value      string
		expiration time.Duration
		sleepTime  time.Duration
		wantErr    bool
	}{
		{
			name:       "缓存项过期后无法获取",
			key:        "key1",
			value:      "value1",
			expiration: time.Millisecond * 100,
			sleepTime:  time.Millisecond * 150,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewBuildInMapCache(time.Minute)

			err := c.Set(context.Background(), tt.key, tt.value, tt.expiration)
			assert.NoError(t, err)

			time.Sleep(tt.sleepTime)
			_, err = c.Get(context.Background(), tt.key)

			if tt.wantErr {
				assert.Error(t, err)
				assert.True(t, strings.Contains(err.Error(), implErrKeyNotFound.Error()))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestBuildInMapCache_Delete 测试缓存的删除功能
func TestBuildInMapCache_Delete(t *testing.T) {
	tests := []struct {
		name      string
		setupKey  string
		setupVal  string
		deleteKey string
		checkKey  string
		wantErr   bool
	}{
		{
			name:      "成功删除缓存项",
			setupKey:  "key1",
			setupVal:  "value1",
			deleteKey: "key1",
			checkKey:  "key1",
			wantErr:   true,
		},
		{
			name:      "删除不存在的key",
			setupKey:  "key1",
			setupVal:  "value1",
			deleteKey: "key2",
			checkKey:  "key1",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewBuildInMapCache(time.Minute)

			err := c.Set(context.Background(), tt.setupKey, tt.setupVal, time.Minute)
			assert.NoError(t, err)

			err = c.Delete(context.Background(), tt.deleteKey)
			assert.NoError(t, err)

			_, err = c.Get(context.Background(), tt.checkKey)
			if tt.wantErr {
				assert.Error(t, err)
				assert.True(t, strings.Contains(err.Error(), implErrKeyNotFound.Error()))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestBuildInMapCache_LoadAndDelete 测试获取并删除缓存项功能
func TestBuildInMapCache_LoadAndDelete(t *testing.T) {
	tests := []struct {
		name         string
		setupKey     string
		setupVal     string
		loadKey      string
		wantVal      any
		wantLoadErr  bool
		wantCheckErr bool
	}{
		{
			name:         "成功获取并删除缓存项",
			setupKey:     "key1",
			setupVal:     "value1",
			loadKey:      "key1",
			wantVal:      "value1",
			wantLoadErr:  false,
			wantCheckErr: true,
		},
		{
			name:         "获取并删除不存在的key",
			setupKey:     "key1",
			setupVal:     "value1",
			loadKey:      "key2",
			wantVal:      nil,
			wantLoadErr:  true,
			wantCheckErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewBuildInMapCache(time.Minute)

			err := c.Set(context.Background(), tt.setupKey, tt.setupVal, time.Minute)
			assert.NoError(t, err)

			val, err := c.LoadAndDelete(context.Background(), tt.loadKey)
			if tt.wantLoadErr {
				assert.Error(t, err)
				assert.Equal(t, tt.wantVal, val)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantVal, val)

				// 验证删除后无法获取
				_, err = c.Get(context.Background(), tt.loadKey)
				if tt.wantCheckErr {
					assert.Error(t, err)
					assert.True(t, strings.Contains(err.Error(), implErrKeyNotFound.Error()))
				}
			}
		})
	}
}

// TestBuildInMapCache_Concurrency 测试缓存的并发安全性
// 验证点:
// 1. 并发写入不会导致数据竞争
// 2. 并发读取能获取正确的值
// 3. 读写操作在并发情况下能正常工作
// TestBuildInMapCache_Concurrency 测试缓存的并发安全性
// 验证点:
// 1. 并发写入不会导致数据竞争
// 2. 并发读取能获取正确的值
// 3. 读写操作在并发情况下能正常工作
func TestBuildInMapCache_Concurrency(t *testing.T) {
	// 创建一个新的内置映射缓存实例，设置默认过期时间为1分钟
	c := NewBuildInMapCache(time.Minute)
	// 定义一个同步等待组，用于等待所有的并发goroutine完成
	var wg sync.WaitGroup

	// 并发写测试
	// 启动100个goroutine并发地向缓存中写入数据
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			// 生成一个唯一的键，格式为 "key%d"
			key := fmt.Sprintf("键：%d", i)
			// 向缓存中设置键值对，值的格式为 "value%d"，过期时间为1分钟
			err := c.Set(context.Background(), key, fmt.Sprintf("값：%d", i), time.Minute)
			assert.Nil(t, err)
		}(i)
	}

	// 并发读测试
	// 启动100个goroutine并发地从缓存中读取数据
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			// 生成一个唯一的键，格式为 "key%d"
			key := fmt.Sprintf("键：%d", i)
			// 从缓存中获取指定键的值
			val, err := c.Get(context.Background(), key)
			if err == nil {
				assert.Equal(t, fmt.Sprintf("값：%d", i), val)
			}
		}(i)
	}

	// 等待所有的goroutine执行完毕
	wg.Wait()
}

// TestBuildInMapCache_OnEvicted 测试内置映射缓存的淘汰回调功能
func TestBuildInMapCache_OnEvicted(t *testing.T) {
	tests := []struct {
		name           string
		key            string
		value          string
		operation      string
		wantEvictedKey string
		wantEvictedVal string
	}{
		{
			name:           "删除操作触发回调",
			key:            "key1",
			value:          "value1",
			operation:      "delete",
			wantEvictedKey: "key1",
			wantEvictedVal: "value1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var evictedKey string
			var evictedValue any

			c := NewBuildInMapCache(time.Minute, BuildInMapCacheWithEvictedCallback(func(key string, val any) {
				evictedKey = key
				evictedValue = val
			}))

			err := c.Set(context.Background(), tt.key, tt.value, time.Minute)
			assert.NoError(t, err)

			switch tt.operation {
			case "delete":
				err = c.Delete(context.Background(), tt.key)
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.wantEvictedKey, evictedKey)
			assert.Equal(t, tt.wantEvictedVal, evictedValue)
		})
	}
}

// TestBuildInMapCache_BackgroundCleanup 测试后台清理过期缓存项功能
func TestBuildInMapCache_BackgroundCleanup(t *testing.T) {
	tests := []struct {
		name            string
		cleanupInterval time.Duration
		key             string
		value           string
		expiration      time.Duration
		sleepTime       time.Duration
		wantErr         bool
	}{
		{
			name:            "后台清理过期缓存项",
			cleanupInterval: time.Millisecond * 100,
			key:             "expireKey",
			value:           "value",
			expiration:      time.Millisecond * 50,
			sleepTime:       time.Millisecond * 200,
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewBuildInMapCache(tt.cleanupInterval)
			ctx := context.Background()

			err := c.Set(ctx, tt.key, tt.value, tt.expiration)
			assert.NoError(t, err)

			time.Sleep(tt.sleepTime)

			_, err = c.Get(ctx, tt.key)
			if tt.wantErr {
				assert.Error(t, err)
				assert.True(t, strings.Contains(err.Error(), implErrKeyNotFound.Error()))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestBuildInMapCache_Close 测试缓存的关闭功能
func TestBuildInMapCache_Close(t *testing.T) {
	tests := []struct {
		name       string
		operations []string
		wantErr    error
	}{
		{
			name:       "第一次关闭成功",
			operations: []string{"close"},
			wantErr:    nil,
		},
		{
			name:       "重复关闭返回错误",
			operations: []string{"close", "close"},
			wantErr:    ErrDuplicateClose,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewBuildInMapCache(time.Minute)
			var err error

			for _, op := range tt.operations {
				if op == "close" {
					err = c.Close()
				}
			}

			if tt.wantErr != nil {
				assert.Equal(t, tt.wantErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestBuildInMapCache_OnEvicted 测试OnEvicted方法
func TestBuildInMapCache_OnEvicted_Method(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		operation    string
		wantContains string
	}{
		{
			name:         "删除操作触发OnEvicted回调",
			key:          "key1",
			value:        "value1",
			operation:    "delete",
			wantContains: "key1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewBuildInMapCache(time.Minute)

			evictedKeys := make([]string, 0)
			c.OnEvicted(func(key string, val any) {
				evictedKeys = append(evictedKeys, key)
			})

			err := c.Set(context.Background(), tt.key, tt.value, time.Minute)
			assert.NoError(t, err)

			switch tt.operation {
			case "delete":
				err = c.Delete(context.Background(), tt.key)
				assert.NoError(t, err)
			}

			assert.Contains(t, evictedKeys, tt.wantContains)
		})
	}
}

// TestBuildInMapCache_LoadAndDelete_NotFound 测试LoadAndDelete方法处理不存在的key
func TestBuildInMapCache_LoadAndDelete_NotFound(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr error
		wantVal any
	}{
		{
			name:    "LoadAndDelete不存在的key",
			key:     "not_exist",
			wantErr: ErrCacheKeyNotFound,
			wantVal: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewBuildInMapCache(time.Minute)

			val, err := c.LoadAndDelete(context.Background(), tt.key)
			assert.Equal(t, tt.wantErr, err)
			assert.Equal(t, tt.wantVal, val)
		})
	}
}

// TestBuildInMapCache_Get_EdgeCases 测试Get方法的边界情况
func TestBuildInMapCache_Get_EdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		value      string
		expiration time.Duration
		sleepTime  time.Duration
		wantErr    bool
		wantVal    any
	}{
		{
			name:       "获取过期的值返回错误",
			key:        "key1",
			value:      "value1",
			expiration: time.Nanosecond,
			sleepTime:  time.Millisecond,
			wantErr:    true,
			wantVal:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewBuildInMapCache(time.Minute)

			err := c.Set(context.Background(), tt.key, tt.value, tt.expiration)
			assert.NoError(t, err)

			time.Sleep(tt.sleepTime)

			val, err := c.Get(context.Background(), tt.key)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), ErrCacheKeyNotFound.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantVal, val)
		})
	}
}

// TestBuildInMapCache_Delete_NonExistent 测试删除不存在的key
func TestBuildInMapCache_Delete_NonExistent(t *testing.T) {
	tests := []struct {
		name             string
		deleteKey        string
		wantEvictedEmpty bool
	}{
		{
			name:             "删除不存在的key不触发回调",
			deleteKey:        "not_exist",
			wantEvictedEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewBuildInMapCache(time.Minute)

			evictedKeys := make([]string, 0)
			c.OnEvicted(func(key string, val any) {
				evictedKeys = append(evictedKeys, key)
			})

			err := c.Delete(context.Background(), tt.deleteKey)
			assert.NoError(t, err)

			if tt.wantEvictedEmpty {
				assert.Empty(t, evictedKeys)
			}
		})
	}
}

// TestBuildInMapCache_ZeroInterval 测试零间隔时间的情况
func TestBuildInMapCache_ZeroInterval(t *testing.T) {
	tests := []struct {
		name     string
		interval time.Duration
		key      string
		value    string
		wantVal  string
	}{
		{
			name:     "零间隔不启动清理goroutine",
			interval: 0,
			key:      "key1",
			value:    "value1",
			wantVal:  "value1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewBuildInMapCache(tt.interval)

			err := c.Set(context.Background(), tt.key, tt.value, time.Minute)
			assert.NoError(t, err)

			val, err := c.Get(context.Background(), tt.key)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantVal, val)

			err = c.Close()
			assert.NoError(t, err)
		})
	}
}

// TestBuildInMapCache_Get_ConcurrentExpiration 测试并发过期检查
func TestBuildInMapCache_Get_ConcurrentExpiration(t *testing.T) {
	c := NewBuildInMapCache(time.Minute)

	// 设置一个即将过期的值
	err := c.Set(context.Background(), "key1", "value1", time.Nanosecond)
	assert.Nil(t, err)

	// 等待过期
	time.Sleep(time.Millisecond)

	// 并发获取，测试双重检查逻辑
	go func() {
		_, _ = c.Get(context.Background(), "key1")
	}()

	val, err := c.Get(context.Background(), "key1")
	assert.Error(t, err)
	assert.Nil(t, val)
}
