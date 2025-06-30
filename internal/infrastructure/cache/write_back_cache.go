package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/justinwongcn/hamster/internal/domain/cache"
)

// WriteBackCache 实现写回缓存模式
// 写入时只更新缓存，不立即写入持久化存储
// 通过异步批量写入或定时刷新的方式将脏数据写入持久化存储
type WriteBackCache struct {
	cache.Repository                 // 嵌入领域仓储接口
	dirtyKeys        map[string]bool // 脏数据键集合
	dirtyMutex       sync.RWMutex    // 脏数据锁
	flushInterval    time.Duration   // 刷新间隔
	batchSize        int             // 批量大小
	lastFlushTime    time.Time       // 上次刷新时间
	flushMutex       sync.Mutex      // 刷新锁
}

// NewWriteBackCache 创建写回缓存实例
// repository: 底层缓存仓储
// flushInterval: 刷新间隔
// batchSize: 批量大小
// 返回: WriteBackCache实例
func NewWriteBackCache(repository cache.Repository, flushInterval time.Duration, batchSize int) *WriteBackCache {
	return &WriteBackCache{
		Repository:    repository,
		dirtyKeys:     make(map[string]bool),
		flushInterval: flushInterval,
		batchSize:     batchSize,
		lastFlushTime: time.Now(),
	}
}

// SetDirty 设置缓存值并标记为脏数据
// 只写入缓存，不立即写入持久化存储
// ctx: 上下文
// key: 缓存键
// val: 缓存值
// expiration: 过期时间
// 返回: 操作错误
func (w *WriteBackCache) SetDirty(ctx context.Context, key string, val any, expiration time.Duration) error {
	// 先写入缓存
	err := w.Repository.Set(ctx, key, val, expiration)
	if err != nil {
		return fmt.Errorf("写入缓存失败: %w", err)
	}

	// 标记为脏数据
	w.dirtyMutex.Lock()
	w.dirtyKeys[key] = true
	w.dirtyMutex.Unlock()

	return nil
}

// FlushKey 强制将指定键的脏数据写入持久化存储
// ctx: 上下文
// key: 缓存键
// storer: 数据存储函数
// 返回: 操作错误
func (w *WriteBackCache) FlushKey(ctx context.Context, key string, storer func(ctx context.Context, key string, val any) error) error {
	// 检查键是否为脏数据
	w.dirtyMutex.RLock()
	isDirty := w.dirtyKeys[key]
	w.dirtyMutex.RUnlock()

	if !isDirty {
		return fmt.Errorf("键 %s 不存在或不是脏数据", key)
	}

	// 从缓存获取值
	val, err := w.Repository.Get(ctx, key)
	if err != nil {
		return fmt.Errorf("从缓存获取键 %s 失败: %w", key, err)
	}

	// 写入持久化存储
	err = storer(ctx, key, val)
	if err != nil {
		return fmt.Errorf("写入持久化存储失败: %w", err)
	}

	// 标记为干净数据
	w.dirtyMutex.Lock()
	delete(w.dirtyKeys, key)
	w.dirtyMutex.Unlock()

	return nil
}

// Flush 强制将所有脏数据写入持久化存储
// ctx: 上下文
// storer: 数据存储函数
// 返回: 操作错误
func (w *WriteBackCache) Flush(ctx context.Context, storer func(ctx context.Context, key string, val any) error) error {
	w.flushMutex.Lock()
	defer w.flushMutex.Unlock()

	// 获取所有脏数据键
	w.dirtyMutex.RLock()
	dirtyKeys := make([]string, 0, len(w.dirtyKeys))
	for key := range w.dirtyKeys {
		dirtyKeys = append(dirtyKeys, key)
	}
	w.dirtyMutex.RUnlock()

	if len(dirtyKeys) == 0 {
		return nil // 没有脏数据需要刷新
	}

	var errors []error
	successKeys := make([]string, 0, len(dirtyKeys))

	// 批量写入持久化存储
	for _, key := range dirtyKeys {
		val, err := w.Repository.Get(ctx, key)
		if err != nil {
			errors = append(errors, fmt.Errorf("获取键 %s 失败: %w", key, err))
			continue
		}

		err = storer(ctx, key, val)
		if err != nil {
			errors = append(errors, fmt.Errorf("存储键 %s 失败: %w", key, err))
			continue
		}

		successKeys = append(successKeys, key)
	}

	// 清理成功写入的脏数据标记
	if len(successKeys) > 0 {
		w.dirtyMutex.Lock()
		for _, key := range successKeys {
			delete(w.dirtyKeys, key)
		}
		w.dirtyMutex.Unlock()

		w.lastFlushTime = time.Now()
	}

	// 如果有错误，返回组合错误
	if len(errors) > 0 {
		return fmt.Errorf("刷新过程中发生 %d 个错误: %v", len(errors), errors)
	}

	return nil
}

// GetDirtyKeys 获取所有脏数据键
// 返回: 脏数据键列表
func (w *WriteBackCache) GetDirtyKeys() []string {
	w.dirtyMutex.RLock()
	defer w.dirtyMutex.RUnlock()

	keys := make([]string, 0, len(w.dirtyKeys))
	for key := range w.dirtyKeys {
		keys = append(keys, key)
	}
	return keys
}

// GetDirtyCount 获取脏数据数量
// 返回: 脏数据数量
func (w *WriteBackCache) GetDirtyCount() int {
	w.dirtyMutex.RLock()
	defer w.dirtyMutex.RUnlock()
	return len(w.dirtyKeys)
}

// ShouldFlush 判断是否需要刷新
// 基于时间间隔或批量大小判断
// 返回: 是否需要刷新
func (w *WriteBackCache) ShouldFlush() bool {
	w.dirtyMutex.RLock()
	dirtyCount := len(w.dirtyKeys)
	w.dirtyMutex.RUnlock()

	// 检查批量大小
	if dirtyCount >= w.batchSize {
		return true
	}

	// 检查时间间隔
	if time.Since(w.lastFlushTime) >= w.flushInterval {
		return dirtyCount > 0 // 有脏数据且到了刷新时间
	}

	return false
}

// StartAutoFlush 启动自动刷新
// 在后台定期检查并刷新脏数据
// ctx: 上下文，用于控制停止
// storer: 数据存储函数
func (w *WriteBackCache) StartAutoFlush(ctx context.Context, storer func(ctx context.Context, key string, val any) error) {
	// 使用更短的检查间隔，确保能及时响应批量大小触发
	checkInterval := w.flushInterval / 10
	if checkInterval > 50*time.Millisecond {
		checkInterval = 50 * time.Millisecond // 最大检查间隔50ms
	}

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// 上下文取消，执行最后一次刷新
			_ = w.Flush(ctx, storer)
			return
		case <-ticker.C:
			// 定期检查是否需要刷新
			if w.ShouldFlush() {
				_ = w.Flush(ctx, storer)
			}
		}
	}
}

// Set 重写Set方法，使其表现为写回模式
// 实际调用SetDirty方法
func (w *WriteBackCache) Set(ctx context.Context, key string, val any, expiration time.Duration) error {
	return w.SetDirty(ctx, key, val, expiration)
}

// Delete 重写Delete方法，删除时也要清理脏数据标记
func (w *WriteBackCache) Delete(ctx context.Context, key string) error {
	// 先从底层缓存删除
	err := w.Repository.Delete(ctx, key)

	// 清理脏数据标记（无论删除是否成功）
	w.dirtyMutex.Lock()
	delete(w.dirtyKeys, key)
	w.dirtyMutex.Unlock()

	return err
}

// LoadAndDelete 重写LoadAndDelete方法，删除时也要清理脏数据标记
func (w *WriteBackCache) LoadAndDelete(ctx context.Context, key string) (any, error) {
	// 先从底层缓存获取并删除
	val, err := w.Repository.LoadAndDelete(ctx, key)

	// 清理脏数据标记（无论操作是否成功）
	w.dirtyMutex.Lock()
	delete(w.dirtyKeys, key)
	w.dirtyMutex.Unlock()

	return val, err
}

// OnEvicted 设置淘汰回调函数
// 当缓存项被淘汰时，如果是脏数据需要强制刷新
func (w *WriteBackCache) OnEvicted(fn func(key string, val any)) {
	// 包装原始回调函数，添加脏数据处理逻辑
	wrappedFn := func(key string, val any) {
		// 检查是否为脏数据
		w.dirtyMutex.RLock()
		isDirty := w.dirtyKeys[key]
		w.dirtyMutex.RUnlock()

		if isDirty {
			// 脏数据被淘汰，清理标记
			// 注意：这里应该记录日志或触发告警，因为脏数据丢失了
			w.dirtyMutex.Lock()
			delete(w.dirtyKeys, key)
			w.dirtyMutex.Unlock()
		}

		// 调用原始回调函数
		if fn != nil {
			fn(key, val)
		}
	}

	w.Repository.OnEvicted(wrappedFn)
}
