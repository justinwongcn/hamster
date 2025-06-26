package cache

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var implErrKeyNotFound = ErrCacheKeyNotFound

// TestBuildInMapCache_SetAndGet 测试缓存的基本设置和获取功能
// 验证点:
// 1. 可以成功设置缓存值
// 2. 可以正确获取已设置的缓存值
// 3. 获取不存在的key返回正确的错误信息
func TestBuildInMapCache_SetAndGet(t *testing.T) {
	c := NewBuildInMapCache(time.Minute)

	// 测试正常设置和获取
	err := c.Set(context.Background(), "key1", "value1", time.Minute)
	assert.Nil(t, err)

	val, err := c.Get(context.Background(), "key1")
	assert.Nil(t, err)
	assert.Equal(t, "value1", val)

	// 测试获取不存在的key
	_, err = c.Get(context.Background(), "not_exist")
	assert.True(t, strings.Contains(err.Error(), implErrKeyNotFound.Error()), "错误消息应包含：%s", implErrKeyNotFound)
}

// TestBuildInMapCache_Expiration 测试缓存的过期功能
// 验证点:
// 1. 缓存项在过期后无法获取
// 2. 过期后获取返回正确的错误信息
func TestBuildInMapCache_Expiration(t *testing.T) {
	c := NewBuildInMapCache(time.Minute)

	// 测试设置过期时间
	err := c.Set(context.Background(), "key1", "value1", time.Millisecond*100)
	assert.Nil(t, err)

	time.Sleep(time.Millisecond * 150)
	_, err = c.Get(context.Background(), "key1")
	assert.True(t, strings.Contains(err.Error(), implErrKeyNotFound.Error()), "错误消息应包含：%s", implErrKeyNotFound)
}

// TestBuildInMapCache_Delete 测试缓存的删除功能
// 验证点:
// 1. 可以成功删除缓存项
// 2. 删除后无法获取该缓存项
// 3. 获取已删除的key返回正确的错误信息
func TestBuildInMapCache_Delete(t *testing.T) {
	c := NewBuildInMapCache(time.Minute)

	// 测试删除
	err := c.Set(context.Background(), "key1", "value1", time.Minute)
	assert.Nil(t, err)

	err = c.Delete(context.Background(), "key1")
	assert.Nil(t, err)

	_, err = c.Get(context.Background(), "key1")
	assert.True(t, strings.Contains(err.Error(), implErrKeyNotFound.Error()), "错误消息应包含：%s", implErrKeyNotFound)
}

// TestBuildInMapCache_LoadAndDelete 测试获取并删除缓存项功能
// 验证点:
// 1. 可以成功获取并删除缓存项
// 2. 操作后无法再次获取该缓存项
// 3. 返回被删除的缓存值
func TestBuildInMapCache_LoadAndDelete(t *testing.T) {
	c := NewBuildInMapCache(time.Minute)

	// 测试获取并删除
	err := c.Set(context.Background(), "key1", "value1", time.Minute)
	assert.Nil(t, err)

	val, err := c.LoadAndDelete(context.Background(), "key1")
	assert.Nil(t, err)
	assert.Equal(t, "value1", val)

	_, err = c.Get(context.Background(), "key1")
	assert.True(t, strings.Contains(err.Error(), implErrKeyNotFound.Error()), "错误消息应包含：%s", implErrKeyNotFound)
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
	for i := range 100 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			// 生成一个唯一的键，格式为 "key%d"
			key := fmt.Sprintf("键：%d", i)
			// 向缓存中设置键值对，值的格式为 "value%d"，过期时间为1分钟
			err := c.Set(context.Background(), key, fmt.Sprintf("值：%d", i), time.Minute)
			assert.Nil(t, err)
		}(i)
	}

	// 并发读测试
	// 启动100个goroutine并发地从缓存中读取数据
	for i := range 100 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			// 生成一个唯一的键，格式为 "key%d"
			key := fmt.Sprintf("键：%d", i)
			// 从缓存中获取指定键的值
			val, err := c.Get(context.Background(), key)
			if err == nil {
				assert.Equal(t, fmt.Sprintf("值：%d", i), val)
			}
		}(i)
	}

	// 等待所有的goroutine执行完毕
	wg.Wait()
}

// TestBuildInMapCache_OnEvicted 测试内置映射缓存的淘汰回调功能
// 验证点:
// 1. 当缓存项被删除时，淘汰回调函数是否被正确触发
// 2. 淘汰回调函数接收到的参数是否正确
func TestBuildInMapCache_OnEvicted(t *testing.T) {
	// 定义变量用于存储淘汰回调函数接收到的键和值
	var evictedKey string
	var evictedValue any

	c := NewBuildInMapCache(time.Minute, BuildInMapCacheWithEvictedCallback(func(key string, val any) {
		// 在回调函数中更新存储淘汰键和值的变量
		evictedKey = key
		evictedValue = val
	}))

	err := c.Set(context.Background(), "key1", "value1", time.Millisecond*10)
	assert.Nil(t, err)

	// 直接删除键值以触发回调
	err = c.Delete(context.Background(), "key1")
	assert.Nil(t, err)

	assert.Equal(t, "key1", evictedKey)
	assert.Equal(t, "value1", evictedValue)
}

// TestBuildInMapCache_BackgroundCleanup 测试后台清理过期缓存项功能
func TestBuildInMapCache_BackgroundCleanup(t *testing.T) {
	c := NewBuildInMapCache(time.Millisecond * 100)
	ctx := context.Background()

	err := c.Set(ctx, "expireKey", "value", time.Millisecond*50)
	assert.Nil(t, err)

	time.Sleep(time.Millisecond * 200)

	_, err = c.Get(ctx, "expireKey")
	assert.True(t, strings.Contains(err.Error(), implErrKeyNotFound.Error()), "过期缓存项应被后台清理")
}

// TestBuildInMapCache_Close 测试缓存的关闭功能
// 验证点:
// 1. 成功关闭缓存
// 2. 重复关闭时返回ErrDuplicateClose错误
func TestBuildInMapCache_Close(t *testing.T) {
	c := NewBuildInMapCache(time.Minute)

	// 第一次关闭，应该成功
	err := c.Close()
	assert.Nil(t, err)

	// 第二次关闭，应该返回重复关闭错误
	err = c.Close()
	assert.Equal(t, ErrDuplicateClose, err)
}

// TestBuildInMapCache_OnEvicted 测试OnEvicted方法
func TestBuildInMapCache_OnEvicted_Method(t *testing.T) {
	c := NewBuildInMapCache(time.Minute)

	evictedKeys := make([]string, 0)
	c.OnEvicted(func(key string, val any) {
		evictedKeys = append(evictedKeys, key)
	})

	// 设置一个值然后删除，应该触发回调
	err := c.Set(context.Background(), "key1", "value1", time.Minute)
	assert.Nil(t, err)

	err = c.Delete(context.Background(), "key1")
	assert.Nil(t, err)

	assert.Contains(t, evictedKeys, "key1")
}

// TestBuildInMapCache_LoadAndDelete_NotFound 测试LoadAndDelete方法处理不存在的key
func TestBuildInMapCache_LoadAndDelete_NotFound(t *testing.T) {
	c := NewBuildInMapCache(time.Minute)

	val, err := c.LoadAndDelete(context.Background(), "not_exist")
	assert.Equal(t, ErrCacheKeyNotFound, err)
	assert.Nil(t, val)
}

// TestBuildInMapCache_Get_EdgeCases 测试Get方法的边界情况
func TestBuildInMapCache_Get_EdgeCases(t *testing.T) {
	c := NewBuildInMapCache(time.Minute)

	// 设置一个即将过期的值
	err := c.Set(context.Background(), "key1", "value1", time.Nanosecond)
	assert.Nil(t, err)

	// 等待过期
	time.Sleep(time.Millisecond)

	// 获取过期的值，应该返回错误
	val, err := c.Get(context.Background(), "key1")
	assert.Error(t, err)
	assert.Nil(t, val)
	assert.Contains(t, err.Error(), ErrCacheKeyNotFound.Error())
}

// TestBuildInMapCache_Delete_NonExistent 测试删除不存在的key
func TestBuildInMapCache_Delete_NonExistent(t *testing.T) {
	c := NewBuildInMapCache(time.Minute)

	evictedKeys := make([]string, 0)
	c.OnEvicted(func(key string, val any) {
		evictedKeys = append(evictedKeys, key)
	})

	// 删除不存在的key
	err := c.Delete(context.Background(), "not_exist")
	assert.Nil(t, err)

	// 回调不应该被触发
	assert.Empty(t, evictedKeys)
}

// TestBuildInMapCache_ZeroInterval 测试零间隔时间的情况
func TestBuildInMapCache_ZeroInterval(t *testing.T) {
	c := NewBuildInMapCache(0) // 零间隔，不启动清理goroutine

	// 设置一个值
	err := c.Set(context.Background(), "key1", "value1", time.Minute)
	assert.Nil(t, err)

	// 获取值
	val, err := c.Get(context.Background(), "key1")
	assert.Nil(t, err)
	assert.Equal(t, "value1", val)

	// 关闭缓存
	err = c.Close()
	assert.Nil(t, err)
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
		c.Get(context.Background(), "key1")
	}()

	val, err := c.Get(context.Background(), "key1")
	assert.Error(t, err)
	assert.Nil(t, val)
}
