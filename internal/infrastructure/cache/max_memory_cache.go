// Package cache 提供基于内存限制的缓存实现
// 当内存使用超过限制时，会自动淘汰最久未使用的数据
package cache

import (
	"context"
	"sync"
	"time"

	domainCache "github.com/justinwongcn/hamster/internal/domain/cache"
)

// MaxMemoryCache 实现带内存限制的缓存，默认基于LRU策略
// 当内存使用超过max限制时自动淘汰最久未使用数据
// 线程安全，支持并发访问
type MaxMemoryCache struct {
	Cache  domainCache.Repository // 底层缓存实现，需实现domain.Repository接口
	max    int64                  // 最大内存限制(字节)，超过此值将触发淘汰
	used   int64                  // 当前已使用内存(字节)，仅计算缓存值本身大小
	mutex  *sync.Mutex            // 互斥锁保证并发安全
	policy EvictionPolicy         // 淘汰策略
}

// NewMaxMemoryCache 创建新的MaxMemoryCache实例
// 参数:
//
//	max: 最大内存限制(字节)
//	cache: 底层缓存实现
//	policy: 淘汰策略实现，可选，默认为LRU策略
//
// 返回值:
//
//	*MaxMemoryCache: 新的缓存实例
//
// 功能:
//
//	创建带内存限制的缓存实例，支持自定义淘汰策略
func NewMaxMemoryCache(max int64, cache domainCache.Repository, policy ...EvictionPolicy) *MaxMemoryCache {
	res := &MaxMemoryCache{
		max:    max,
		Cache:  cache,
		mutex:  &sync.Mutex{},
		policy: NewLRUPolicy(), // 默认使用LRU策略
	}
	// 如果提供了自定义策略，则使用自定义策略
	if len(policy) > 0 && policy[0] != nil {
		res.policy = policy[0]
	}
	if res.Cache != nil {
		res.Cache.OnEvicted(func(key string, val any) {
			res.evicted(key, val)
		})
	}
	return res
}

// NewMaxMemoryCacheWithLRU 创建使用LRU策略的MaxMemoryCache实例
// 参数:
//
//	max: 最大内存限制(字节)
//	cache: 底层缓存实现
//
// 返回值:
//
//	*MaxMemoryCache: 新的缓存实例
func NewMaxMemoryCacheWithLRU(max int64, cache domainCache.Repository) *MaxMemoryCache {
	return NewMaxMemoryCache(max, cache, NewLRUPolicy())
}

// NewMaxMemoryCacheWithFIFO 创建使用FIFO策略的MaxMemoryCache实例
// 参数:
//
//	max: 最大内存限制(字节)
//	cache: 底层缓存实现
//
// 返回值:
//
//	*MaxMemoryCache: 新的缓存实例
func NewMaxMemoryCacheWithFIFO(max int64, cache domainCache.Repository) *MaxMemoryCache {
	return NewMaxMemoryCache(max, cache, NewFIFOPolicy())
}

// NewMaxMemoryCacheWithRandom 创建使用随机策略的MaxMemoryCache实例
// 参数:
//
//	max: 最大内存限制(字节)
//	cache: 底层缓存实现
//
// 返回值:
//
//	*MaxMemoryCache: 新的缓存实例
func NewMaxMemoryCacheWithRandom(max int64, cache domainCache.Repository) *MaxMemoryCache {
	return NewMaxMemoryCache(max, cache, NewRandomPolicy())
}

// Set 添加或更新缓存项
// 当内存不足时会自动淘汰最久未使用的数据，确保总内存不超过max限制
// 参数:
//   - key: 缓存键
//   - val: 缓存值
//   - expiration: 过期时间
//
// 返回值:
//   - error: 操作错误信息
func (m *MaxMemoryCache) Set(ctx context.Context, key string, val []byte,
	expiration time.Duration,
) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 先删除可能存在的旧键，避免内存泄露
	oldVal, err := m.Cache.LoadAndDelete(ctx, key)
	if err == nil && oldVal != nil {
		m.evicted(key, oldVal)
	}

	// 将新键值对存入底层缓存
	err = m.Cache.Set(ctx, key, val, expiration)
	if err == nil {
		// 更新已使用内存大小
		m.used = m.used + int64(len(val))
		// 通知策略该键已被访问
		_ = m.policy.KeyAccessed(ctx, key)
	}

	// 如果添加新值后超出最大内存限制，则执行淘汰策略
	for m.used > m.max {
		// 调用淘汰策略获取要删除的键
		k, evictErr := m.policy.Evict(ctx)
		if evictErr != nil || k == "" {
			break // 没有可淘汰的键或出错，退出循环
		}
		// 从底层缓存中删除选中的键
		_ = m.Cache.Delete(ctx, k)
	}

	return err
}

// Get 获取缓存值
// 会更新key的访问时间以维护LRU淘汰顺序
// 参数:
//   - ctx: 上下文
//   - key: 缓存键
//
// 返回值:
//   - []byte: 缓存值
//   - error: 操作错误信息
//
// 功能:
//   - 从缓存中获取指定键的值
//   - 更新键的访问时间
//   - 处理类型断言错误
func (m *MaxMemoryCache) Get(ctx context.Context, key string) (any, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 从底层缓存获取值
	val, err := m.Cache.Get(ctx, key)
	if err == nil {
		// 从策略中移除键（用于更新访问顺序）
		_ = m.policy.Remove(ctx, key)
		// 通知策略该键已被访问
		_ = m.policy.KeyAccessed(ctx, key)

		return val, nil
	}
	return nil, err
}

// Delete 删除指定缓存项
// 参数:
//   - ctx: 上下文
//   - key: 缓存键
//
// 返回值:
//   - error: 操作错误信息
//
// 功能:
//   - 从缓存中删除指定键
//   - 更新内存使用统计
func (m *MaxMemoryCache) Delete(ctx context.Context, key string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.Cache.Delete(ctx, key)
}

// LoadAndDelete 获取并删除缓存项
// 参数:
//   - ctx: 上下文
//   - key: 缓存键
//
// 返回值:
//   - []byte: 缓存值
//   - error: 操作错误信息
//
// 功能:
//   - 原子性地获取并删除指定键
//   - 更新内存使用统计
//   - 处理类型断言错误
func (m *MaxMemoryCache) LoadAndDelete(ctx context.Context, key string) (any, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 从底层缓存获取并删除值
	val, err := m.Cache.LoadAndDelete(ctx, key)
	if err == nil {
		return val, nil
	}
	return nil, err
}

// OnEvicted 设置淘汰回调函数
// 当缓存项被淘汰时调用
// 参数:
//   - fn: 回调函数
//
// 功能:
//   - 设置淘汰时的回调处理
//   - 保证线程安全
func (m *MaxMemoryCache) OnEvicted(fn func(key string, val any)) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.Cache.OnEvicted(func(key string, val any) {
		m.evicted(key, val)
		fn(key, val)
	})
}

// evicted 处理缓存项淘汰逻辑
// 当缓存项被淘汰时调用，更新内存统计并从策略中移除key
func (m *MaxMemoryCache) evicted(key string, val any) {
	// 将 any 类型转换为 []byte
	if valBytes, ok := val.([]byte); ok {
		m.used = m.used - int64(len(valBytes))
	}
	// 使用context.Background()，因为这是内部回调
	_ = m.policy.Remove(context.Background(), key)
}
