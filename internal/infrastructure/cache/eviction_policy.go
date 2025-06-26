package cache

import "context"

// EvictionPolicy 定义缓存淘汰策略接口
// 该接口遵循DDD设计原则，提供了缓存淘汰的核心领域行为
// 支持用户自定义淘汰算法的实现
type EvictionPolicy interface {
	// KeyAccessed 记录key被访问
	// 用于更新key的访问状态，不同策略可能有不同的处理方式
	// 参数:
	//   - ctx: 上下文，用于传递请求级别的信息
	//   - key: 被访问的缓存键
	// 返回值:
	//   - error: 操作错误，nil表示成功
	KeyAccessed(ctx context.Context, key string) error

	// Evict 执行淘汰并返回被淘汰的key
	// 根据具体策略选择要淘汰的key
	// 参数:
	//   - ctx: 上下文，用于传递请求级别的信息
	// 返回值:
	//   - string: 被淘汰的key，空字符串表示没有可淘汰的key
	//   - error: 操作错误，nil表示成功
	Evict(ctx context.Context) (string, error)

	// Remove 移除指定key
	// 从策略中移除指定的key，通常在key被删除时调用
	// 参数:
	//   - ctx: 上下文，用于传递请求级别的信息
	//   - key: 要移除的缓存键
	// 返回值:
	//   - error: 操作错误，nil表示成功
	Remove(ctx context.Context, key string) error

	// Has 判断key是否存在于策略中
	// 检查策略是否正在跟踪指定的key
	// 参数:
	//   - ctx: 上下文，用于传递请求级别的信息
	//   - key: 要检查的缓存键
	// 返回值:
	//   - bool: true表示key存在，false表示不存在
	//   - error: 操作错误，nil表示成功
	Has(ctx context.Context, key string) (bool, error)

	// Size 返回策略中跟踪的key数量
	// 用于监控和调试
	// 参数:
	//   - ctx: 上下文，用于传递请求级别的信息
	// 返回值:
	//   - int: 当前跟踪的key数量
	//   - error: 操作错误，nil表示成功
	Size(ctx context.Context) (int, error)

	// Clear 清空策略中的所有key
	// 用于重置策略状态
	// 参数:
	//   - ctx: 上下文，用于传递请求级别的信息
	// 返回值:
	//   - error: 操作错误，nil表示成功
	Clear(ctx context.Context) error
}
