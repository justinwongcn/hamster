package hamster

import (
	"context"
	"time"
)

// Cache 定义缓存接口
// 提供基本的缓存操作：设置、获取、删除和获取并删除
type Cache interface {
	// Set 设置缓存值
	// key: 缓存键
	// val: 缓存值
	// expiration: 过期时间
	Set(ctx context.Context, key string, val any, expiration time.Duration) error

	// Get 获取缓存值
	// key: 缓存键
	// 返回: 缓存值和错误信息
	Get(ctx context.Context, key string) (any, error)

	// Delete 删除缓存值
	// key: 缓存键
	// 返回: 错误信息
	Delete(ctx context.Context, key string) error

	// LoadAndDelete 获取并删除缓存值
	// key: 缓存键
	// 返回: 被删除的缓存值和错误信息
	LoadAndDelete(ctx context.Context, key string) (any, error)

	// OnEvicted 设置淘汰回调函数
	// fn: 回调函数，当缓存项被淘汰时调用
	OnEvicted(fn func(key string, val any))
}

// 注意：具体的配置类型和选项函数已移至各自的子包中
// 例如：cache.Config, cache.Option, hash.Config, hash.Option, lock.Config, lock.Option
