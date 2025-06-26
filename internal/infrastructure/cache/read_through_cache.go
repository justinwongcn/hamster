package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/justinwongcn/hamster/internal/interfaces"
)

var (
	ErrFailedToRefreshCache = errors.New("刷新缓存失败")
	ErrKeyNotFound          = errors.New("键未找到")
)

// ReadThroughCache 实现读透缓存模式
// 当缓存未命中时自动从数据源加载数据并更新缓存
// 使用single flight.Group防止缓存击穿
type ReadThroughCache struct {
	interfaces.Cache
	LoadFunc   func(ctx context.Context, key string) (any, error)
	Expiration time.Duration
	logFunc    func(format string, args ...any)
	g          singleflight.Group
}

// RateLimitReadThroughCache 带限流功能的读透缓存
// 必须赋值 LoadFunc 和 Expiration 字段
// Expiration 是缓存过期时间
type RateLimitReadThroughCache struct {
	interfaces.Cache
	LoadFunc   func(ctx context.Context, key string) (any, error)
	Expiration time.Duration
	g          singleflight.Group
}

// Get 实现读透缓存获取逻辑
// 参数:
//   - ctx: 上下文
//   - key: 缓存键
//
// 返回值:
//   - any: 缓存值
//   - error: 错误信息
//
// 功能:
//   - 优先从缓存获取数据
//   - 缓存未命中时调用handleCacheMiss处理
func (r *ReadThroughCache) Get(ctx context.Context, key string) (any, error) {
	cachedVal, err := r.Cache.Get(ctx, key)
	if err != nil {
		if errors.Is(err, ErrKeyNotFound) {
			return r.handleCacheMiss(ctx, key)
		}
		return nil, err
	}
	return cachedVal, nil
}

// Get 实现带限流功能的缓存获取逻辑
// 当缓存未命中且未被限流时，调用LoadFunc加载数据并更新缓存
func (r *RateLimitReadThroughCache) Get(ctx context.Context, key string) (any, error) {
	val, err := r.Cache.Get(ctx, key)
	if errors.Is(err, ErrKeyNotFound) && ctx.Value("limited") == nil {
		// 使用single flight防止缓存击穿
		loadedVal, loadErr, _ := r.g.Do(key, func() (any, error) {
			newVal, loadErr := r.LoadFunc(ctx, key)
			if loadErr != nil {
				return nil, loadErr
			}

			// 更新缓存
			if loadErr2 := r.Cache.Set(ctx, key, newVal, r.Expiration); loadErr2 != nil {
				return newVal, fmt.Errorf("%w, 原因：%s", ErrFailedToRefreshCache, loadErr2.Error())
			}
			return newVal, nil
		})
		return loadedVal, loadErr
	}
	return val, err
}

// handleCacheMiss 处理缓存未命中时的数据加载和缓存更新
// handleCacheMiss 处理缓存未命中时的数据加载
// 参数:
//   - ctx: 上下文
//   - key: 缓存键
//
// 返回值:
//   - any: 加载的值
//   - error: 错误信息
//
// 功能:
//   - 使用single flight防止缓存击穿
//   - 调用LoadFunc从数据源加载数据
//   - 更新缓存并处理可能的错误
func (r *ReadThroughCache) handleCacheMiss(ctx context.Context, key string) (any, error) {
	// 使用single flight防止缓存击穿
	loadedVal, loadErr, _ := r.g.Do(key, func() (any, error) {
		// 记录日志
		if r.logFunc != nil {
			r.logFunc("缓存未命中，从数据源加载数据 key: %s", key)
		}

		// 从数据源加载数据
		newVal, loadErr := r.LoadFunc(ctx, key)
		if loadErr != nil {
			return nil, loadErr
		}

		// 尝试更新缓存（即使失败也返回加载的值）
		if setErr := r.Cache.Set(ctx, key, newVal, r.Expiration); setErr != nil {
			if r.logFunc != nil {
				r.logFunc("刷新缓存失败，键：%s，错误：%v", key, setErr)
			}
			// 返回加载的值和错误
			return newVal, fmt.Errorf("%w, 原因：%s", ErrFailedToRefreshCache, setErr.Error())
		}
		return newVal, nil
	})

	// 当有错误时，返回加载的值和错误（符合测试预期）
	if loadErr != nil {
		return loadedVal, loadErr
	}
	return loadedVal, nil
}

// SetLogFunc 设置日志记录函数
// SetLogFunc 设置日志记录函数
// 参数:
//   - logFunc: 日志记录函数
//
// 功能:
//   - 设置缓存操作中的日志记录方式
func (r *ReadThroughCache) SetLogFunc(logFunc func(format string, args ...any)) {
	r.logFunc = logFunc
}
