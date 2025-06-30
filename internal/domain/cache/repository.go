package cache

import (
	"context"
	"time"
)

// Repository 定义缓存仓储接口
// 这是领域层的核心接口，定义了缓存的基本操作
// 遵循DDD原则，接口在领域层定义，实现在基础设施层
type Repository interface {
	// Set 设置缓存值
	// ctx: 上下文，用于传递请求级别的信息和控制超时
	// key: 缓存键，必须是有效的CacheKey
	// val: 缓存值，可以是任意类型
	// expiration: 过期时间，0表示永不过期
	// 返回: 操作错误，nil表示成功
	Set(ctx context.Context, key string, val any, expiration time.Duration) error

	// Get 获取缓存值
	// ctx: 上下文，用于传递请求级别的信息和控制超时
	// key: 缓存键
	// 返回: 缓存值和错误信息，如果键不存在返回ErrKeyNotFound
	Get(ctx context.Context, key string) (any, error)

	// Delete 删除缓存值
	// ctx: 上下文，用于传递请求级别的信息和控制超时
	// key: 缓存键
	// 返回: 操作错误，nil表示成功，键不存在不算错误
	Delete(ctx context.Context, key string) error

	// LoadAndDelete 原子性地获取并删除缓存值
	// ctx: 上下文，用于传递请求级别的信息和控制超时
	// key: 缓存键
	// 返回: 被删除的缓存值和错误信息
	LoadAndDelete(ctx context.Context, key string) (any, error)

	// OnEvicted 设置缓存项淘汰时的回调函数
	// fn: 回调函数，当缓存项被淘汰时调用
	// 用于实现缓存淘汰的业务逻辑，如日志记录、统计等
	OnEvicted(fn func(key string, val any))
}

// ReadThroughRepository 定义读透缓存仓储接口
// 扩展基本的Repository接口，添加读透缓存的特性
type ReadThroughRepository interface {
	Repository
	
	// GetWithLoader 使用加载器获取缓存值
	// 如果缓存未命中，会调用loader函数加载数据并更新缓存
	// ctx: 上下文
	// key: 缓存键
	// loader: 数据加载函数，用于从数据源加载数据
	// expiration: 缓存过期时间
	// 返回: 缓存值和错误信息
	GetWithLoader(ctx context.Context, key string, loader func(ctx context.Context, key string) (any, error), expiration time.Duration) (any, error)
}

// WriteThroughRepository 定义写透缓存仓储接口
// 扩展基本的Repository接口，添加写透缓存的特性
type WriteThroughRepository interface {
	Repository
	
	// SetWithStore 使用存储器设置缓存值
	// 先写入持久化存储，再写入缓存
	// ctx: 上下文
	// key: 缓存键
	// val: 缓存值
	// storer: 数据存储函数，用于写入持久化存储
	// expiration: 缓存过期时间
	// 返回: 操作错误
	SetWithStore(ctx context.Context, key string, val any, storer func(ctx context.Context, key string, val any) error, expiration time.Duration) error
}

// WriteBackRepository 定义写回缓存仓储接口
// 扩展基本的Repository接口，添加写回缓存的特性
type WriteBackRepository interface {
	Repository
	
	// SetDirty 设置缓存值并标记为脏数据
	// 只写入缓存，不立即写入持久化存储
	// ctx: 上下文
	// key: 缓存键
	// val: 缓存值
	// expiration: 缓存过期时间
	// 返回: 操作错误
	SetDirty(ctx context.Context, key string, val any, expiration time.Duration) error
	
	// Flush 强制将脏数据写入持久化存储
	// ctx: 上下文
	// storer: 数据存储函数
	// 返回: 操作错误
	Flush(ctx context.Context, storer func(ctx context.Context, key string, val any) error) error
	
	// FlushKey 强制将指定键的脏数据写入持久化存储
	// ctx: 上下文
	// key: 缓存键
	// storer: 数据存储函数
	// 返回: 操作错误
	FlushKey(ctx context.Context, key string, storer func(ctx context.Context, key string, val any) error) error
}
