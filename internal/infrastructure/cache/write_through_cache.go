package cache

import (
	"context"
	"time"

	domainCache "github.com/justinwongcn/hamster/internal/domain/cache"
)

// WriteThroughCache 实现写透缓存模式
// 当写入缓存时，同时写入到持久化存储
// 确保缓存和存储的数据一致性
type WriteThroughCache struct {
	domainCache.Repository
	StoreFunc func(ctx context.Context, key string, val any) error
}

// RateLimitWriteThroughCache 带限流功能的写透缓存
// 当写入被限流时，跳过持久化存储的写入
// 必须赋值 StoreFunc 字段
type RateLimitWriteThroughCache struct {
	domainCache.Repository
	StoreFunc func(ctx context.Context, key string, val any) error
}

// Set 实现带限流功能的写透缓存设置逻辑
// 当未被限流时，先写入持久化存储再写入缓存
// 当被限流时，只写入缓存
func (r *RateLimitWriteThroughCache) Set(ctx context.Context, key string, val any, expiration time.Duration) error {
	// 检查是否被限流
	if ctx.Value("limited") == nil {
		// 未被限流，先写入持久化存储
		err := r.StoreFunc(ctx, key, val)
		if err != nil {
			return err
		}
	}
	// 写入缓存（无论是否限流都要写入缓存）
	return r.Repository.Set(ctx, key, val, expiration)
}

// Set 实现写透缓存的设置逻辑
// 参数:
//   - ctx: 上下文
//   - key: 缓存键
//   - val: 缓存值
//   - expiration: 过期时间
//
// 返回值:
//   - error: 错误信息
//
// 功能:
//   - 先写入持久化存储
//   - 再写入缓存
//   - 如果持久化失败，不写入缓存
func (w *WriteThroughCache) Set(ctx context.Context, key string, val any, expiration time.Duration) error {
	// 先写入持久化存储
	err := w.StoreFunc(ctx, key, val)
	if err != nil {
		return err
	}
	// 再写入缓存
	return w.Repository.Set(ctx, key, val, expiration)
}
