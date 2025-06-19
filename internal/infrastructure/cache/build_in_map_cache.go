package cache

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

const errKeyNotFoundFormat = "%w, key: %s"

var (
	errKeyNotFound    = errors.New("cache：键不存在")
	errDuplicateClose = errors.New("重复关闭")
)

// BuildInMapCacheOption 定义缓存配置选项函数类型
type BuildInMapCacheOption func(cache *BuildInMapCache)

// BuildInMapCache 基于内置map实现的缓存结构体
// 该结构体包含了缓存操作所需的核心数据结构和控制元素。
type BuildInMapCache struct {
	// data 存储缓存项的映射，键为字符串类型，值为指向item结构体的指针
	// 读写操作需通过互斥锁保护以确保并发安全
	data  map[string]*item
	mutex sync.RWMutex
	// close 用于关闭缓存的通道，发送信号后会停止后台清理goroutine
	// 重复关闭会返回errDuplicateClose错误
	close chan struct{}
	// onEvicted 缓存项被驱逐时的回调函数
	// 当缓存项因过期、删除或内存淘汰被移除时触发
	onEvicted func(key string, val any)
}

// item 缓存项结构体，包含值和过期时间
type item struct {
	val      any
	deadline time.Time
}

// NewBuildInMapCache 创建新的内置map缓存实例，interval 为过期检查间隔时间，opts 为可选配置项。
// 该函数会初始化一个新的 BuildInMapCache 实例，设置初始容量为 100 的 data 映射，创建关闭通道，
// 并设置默认的驱逐回调函数，然后应用所有可选配置项，最后启动一个 goroutine 用于定期清理过期的缓存项。
func NewBuildInMapCache(interval time.Duration, opts ...BuildInMapCacheOption) *BuildInMapCache {
	res := &BuildInMapCache{
		data:  make(map[string]*item, 100),
		close: make(chan struct{}),
		onEvicted: func(key string, val any) {
			// 默认的onEvicted回调为空函数
			// 避免外部未设置回调时调用nil函数导致panic
			// 如需监听驱逐事件，请使用 BuildInMapCacheWithEvictedCallback 配置选项设置具体逻辑。
		},
	}

	// 遍历并应用所有可选配置项到新创建的缓存实例上
	for _, opt := range opts {
		opt(res)
	}

	// 启动 goroutine 定期清理过期缓存项
	go func() {
		// 创建按指定间隔时间触发的定时器
		ticker := time.NewTicker(interval)
		for {
			select {
			case t := <-ticker.C:
				// 加写锁保证清理过程中缓存数据不被其他 goroutine 修改
				res.mutex.Lock()
				// 计数器限制每次清理检查的缓存项数量，避免长时间占用锁
				i := 0
				// 遍历缓存项，检查并删除过期项
				for key, val := range res.data {
					if i > 10000 {
						break
					}
					if val.deadlineBefore(t) {
						res.delete(key)
					}
					i++
				}
				// 解锁允许其他 goroutine 访问缓存数据
				res.mutex.Unlock()
			case <-res.close:
				return
			}
		}
	}()

	return res
}

// BuildInMapCacheWithEvictedCallback 设置缓存项被删除时的回调函数
// fn: 回调函数，当缓存项因过期被删除时调用
func BuildInMapCacheWithEvictedCallback(fn func(key string, val any)) BuildInMapCacheOption {
	return func(cache *BuildInMapCache) {
		cache.onEvicted = fn
	}
}

// deadlineBefore 检查缓存项是否在指定时间前过期
// t: 要比较的时间点
// 返回: true表示已过期，false表示未过期
func (i *item) deadlineBefore(t time.Time) bool {
	return !i.deadline.IsZero() && i.deadline.Before(t)
}

// Set 设置缓存值
// ctx: 上下文，可用于取消操作
// key: 缓存键，必须是唯一标识
// val: 要缓存的值，可以是任意类型
// expiration: 过期时间，0表示永不过期
// 返回: 错误信息，nil表示成功
func (b *BuildInMapCache) Set(_ context.Context, key string, val any, expiration time.Duration) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.set(key, val, expiration)
}

// set 内部实现方法，设置缓存值
// 注意: 此方法应在持有锁的情况下调用
// key: 缓存键
// val: 缓存值
// expiration: 过期时间
// 返回: 错误信息，nil表示成功
func (b *BuildInMapCache) set(key string, val any, expiration time.Duration) error {
	var dl time.Time
	if expiration > 0 {
		dl = time.Now().Add(expiration)
	}
	b.data[key] = &item{
		val:      val,
		deadline: dl,
	}
	return nil
}

// Get 获取缓存值
// ctx: 上下文，可用于取消操作
// key: 缓存键
// 返回: (缓存值, 错误信息)
// 注意: 如果缓存项已过期会自动删除并返回错误
func (b *BuildInMapCache) Get(_ context.Context, key string) (any, error) {
	// 加读锁以允许其他 goroutine 同时读取缓存数据，然后从缓存中获取指定键的值，最后释放读锁。
	b.mutex.RLock()
	res, ok := b.data[key]
	b.mutex.RUnlock()

	// 如果缓存中不存在该键，返回错误。
	if !ok {
		return nil, fmt.Errorf(errKeyNotFoundFormat, errKeyNotFound, key)
	}

	// 获取当前时间，检查缓存项是否已过期。
	now := time.Now()
	if res.deadlineBefore(now) {
		// 加写锁确保删除过期缓存项时数据一致性，函数返回时释放写锁。再次获取键值防止数据被修改，若不存在则返回错误，若仍过期则删除并返回错误。
		b.mutex.Lock()
		defer b.mutex.Unlock()
		res, ok = b.data[key]
		if !ok {
			return nil, fmt.Errorf(errKeyNotFoundFormat, errKeyNotFound, key)
		}
		if res.deadlineBefore(now) {
			b.delete(key)
			return nil, fmt.Errorf(errKeyNotFoundFormat, errKeyNotFound, key)
		}
	}
	// 返回缓存值。
	return res.val, nil
}

// Delete 删除缓存值
// ctx: 上下文，可用于取消操作
// key: 缓存键
// 返回: 错误信息，nil表示成功
func (b *BuildInMapCache) Delete(_ context.Context, key string) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.delete(key)
	return nil
}

// LoadAndDelete 获取并删除缓存值
// ctx: 上下文，可用于取消操作
// key: 缓存键
// 返回: (被删除的缓存值, 错误信息)
func (b *BuildInMapCache) LoadAndDelete(_ context.Context, key string) (any, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	val, ok := b.data[key]
	if !ok {
		return nil, errKeyNotFound
	}
	b.delete(key)
	return val.val, nil
}

// delete 内部实现方法，删除缓存项
// 注意: 此方法应在持有锁的情况下调用
// key: 缓存键
// 会触发onEvicted回调函数
func (b *BuildInMapCache) delete(key string) {
	itm, ok := b.data[key]
	if !ok {
		return
	}
	delete(b.data, key)
	b.onEvicted(key, itm.val)
}

// Close 关闭缓存，停止后台清理goroutine
// 返回: 错误信息，nil表示成功
// 注意: 重复关闭会返回错误
func (b *BuildInMapCache) Close() error {
	select {
	case b.close <- struct{}{}:
	default:
		return errDuplicateClose
	}
	return nil
}
