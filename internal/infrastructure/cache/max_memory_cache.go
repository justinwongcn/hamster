// Package cache 提供基于内存限制的缓存实现
// 当内存使用超过限制时，会自动淘汰最久未使用的数据
package cache

import (
    "context"
	"errors"
	"sync"
	"time"

	"github.com/justinwongcn/hamster/internal/interfaces"
)

// MaxMemoryCache 实现了带内存限制的缓存
// 当内存使用超过max限制时，会自动淘汰最久未使用的数据
// 线程安全，支持并发访问
type MaxMemoryCache struct {
    Cache  interfaces.Cache // 底层缓存实现
    max    int64          // 最大内存限制(字节)
    used   int64          // 当前已使用内存(字节)
    mutex  *sync.Mutex    // 互斥锁保证并发安全
    policy EvictionPolicy // 淘汰策略
}

// NewMaxMemoryCache 创建新的MaxMemoryCache实例
// max: 最大内存限制(字节)
// cache: 底层缓存实现
func NewMaxMemoryCache(max int64, cache interfaces.Cache) *MaxMemoryCache {
    res := &MaxMemoryCache{
        max:   max,
        Cache: cache,
        mutex: &sync.Mutex{},
        policy: NewLRUPolicy(), // 默认使用LRU策略
    }
    if res.Cache != nil {
        res.Cache.OnEvicted(func(key string, val any) {
            res.evicted(key, val)
        })
    }
    return res
}

// Set 添加或更新缓存项
// 当内存不足时会自动淘汰最久未使用的数据
// 参数:
//   - key: 缓存键
//   - val: 缓存值
//   - expiration: 过期时间
// 返回值:
//   - error: 操作错误信息
func (m *MaxMemoryCache) Set(ctx context.Context, key string, val []byte,
    expiration time.Duration) error {
    m.mutex.Lock()
    defer m.mutex.Unlock()

    _, _ = m.Cache.LoadAndDelete(ctx, key)
    for m.used+int64(len(val)) > m.max {
        k := m.policy.Evict()
        if k == "" {
            break
        }
        _ = m.Cache.Delete(ctx, k)
    }
    err := m.Cache.Set(ctx, key, val, expiration)
    if err == nil {
        m.used = m.used + int64(len(val))
        m.policy.KeyAccessed(key)
    }

    return err
}

// Get 获取缓存值
// 会更新key的访问时间
// 参数:
//   - key: 缓存键
// 返回值:
//   - []byte: 缓存值
//   - error: 操作错误信息
func (m *MaxMemoryCache) Get(ctx context.Context, key string) ([]byte, error) {
    m.mutex.Lock()
    defer m.mutex.Unlock()
    val, err := m.Cache.Get(ctx, key)
    if err == nil {
        m.policy.Remove(key)
        m.policy.KeyAccessed(key)
        if valBytes, ok := val.([]byte); ok {
            return valBytes, nil
        }
        return nil, errors.New("value is not []byte")
    }
    return nil, err
}

// Delete 删除指定缓存项
// 参数:
//   - key: 缓存键
// 返回值:
//   - error: 操作错误信息
func (m *MaxMemoryCache) Delete(ctx context.Context, key string) error {
    m.mutex.Lock()
    defer m.mutex.Unlock()
    return m.Cache.Delete(ctx, key)
}

// LoadAndDelete 获取并删除缓存项
// 参数:
//   - key: 缓存键
// 返回值:
//   - []byte: 缓存值
//   - error: 操作错误信息
func (m *MaxMemoryCache) LoadAndDelete(ctx context.Context, key string) ([]byte, error) {
    m.mutex.Lock()
    defer m.mutex.Unlock()
    val, err := m.Cache.LoadAndDelete(ctx, key)
    if err == nil {
        if valBytes, ok := val.([]byte); ok {
            return valBytes, nil
        }
        return nil, errors.New("value is not []byte")
    }
    return nil, err
}

// OnEvicted 设置淘汰回调函数
// 当缓存项被淘汰时调用
// 参数:
//   - fn: 回调函数
func (m *MaxMemoryCache) OnEvicted(fn func(key string, val any)) {
    m.mutex.Lock()
    defer m.mutex.Unlock()
    m.Cache.OnEvicted(func(key string, val any) {
        m.evicted(key, val)
        fn(key, val)
    })
}

// evicted 处理淘汰逻辑
// 更新已用内存统计并删除key记录
func (m *MaxMemoryCache) evicted(key string, val any) {
    // 将 any 类型转换为 []byte
    if valBytes, ok := val.([]byte); ok {
        m.used = m.used - int64(len(valBytes))
    }
    m.policy.Remove(key)
}
