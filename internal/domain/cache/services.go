package cache

import (
	"context"
	"fmt"
	"time"
)

// EvictionStrategy 淘汰策略接口
// 定义缓存淘汰的策略行为
type EvictionStrategy interface {
	// SelectForEviction 选择要淘汰的条目
	// entries: 候选条目列表
	// 返回: 要淘汰的条目
	SelectForEviction(entries []*Entry) *Entry
	
	// OnAccess 条目被访问时的回调
	// entry: 被访问的条目
	OnAccess(entry *Entry)
	
	// OnAdd 添加新条目时的回调
	// entry: 新添加的条目
	OnAdd(entry *Entry)
	
	// OnRemove 移除条目时的回调
	// entry: 被移除的条目
	OnRemove(entry *Entry)
}

// LRUEvictionStrategy LRU淘汰策略
// 淘汰最近最少使用的条目
type LRUEvictionStrategy struct{}

// NewLRUEvictionStrategy 创建LRU淘汰策略
func NewLRUEvictionStrategy() *LRUEvictionStrategy {
	return &LRUEvictionStrategy{}
}

// SelectForEviction 选择最近最少使用的条目进行淘汰
func (s *LRUEvictionStrategy) SelectForEviction(entries []*Entry) *Entry {
	if len(entries) == 0 {
		return nil
	}
	
	oldest := entries[0]
	for _, entry := range entries[1:] {
		if entry.AccessedAt().Before(oldest.AccessedAt()) {
			oldest = entry
		}
	}
	return oldest
}

// OnAccess LRU策略不需要特殊处理访问事件
func (s *LRUEvictionStrategy) OnAccess(entry *Entry) {
	// LRU策略通过访问时间自动处理
}

// OnAdd LRU策略不需要特殊处理添加事件
func (s *LRUEvictionStrategy) OnAdd(entry *Entry) {
	// LRU策略通过访问时间自动处理
}

// OnRemove LRU策略不需要特殊处理移除事件
func (s *LRUEvictionStrategy) OnRemove(entry *Entry) {
	// LRU策略不需要特殊处理
}

// FIFOEvictionStrategy FIFO淘汰策略
// 淘汰最先进入的条目
type FIFOEvictionStrategy struct{}

// NewFIFOEvictionStrategy 创建FIFO淘汰策略
func NewFIFOEvictionStrategy() *FIFOEvictionStrategy {
	return &FIFOEvictionStrategy{}
}

// SelectForEviction 选择最先创建的条目进行淘汰
func (s *FIFOEvictionStrategy) SelectForEviction(entries []*Entry) *Entry {
	if len(entries) == 0 {
		return nil
	}
	
	oldest := entries[0]
	for _, entry := range entries[1:] {
		if entry.CreatedAt().Before(oldest.CreatedAt()) {
			oldest = entry
		}
	}
	return oldest
}

// OnAccess FIFO策略不需要处理访问事件
func (s *FIFOEvictionStrategy) OnAccess(entry *Entry) {
	// FIFO策略不关心访问时间
}

// OnAdd FIFO策略不需要特殊处理添加事件
func (s *FIFOEvictionStrategy) OnAdd(entry *Entry) {
	// FIFO策略通过创建时间自动处理
}

// OnRemove FIFO策略不需要特殊处理移除事件
func (s *FIFOEvictionStrategy) OnRemove(entry *Entry) {
	// FIFO策略不需要特殊处理
}

// CacheService 缓存领域服务
// 封装缓存的核心业务逻辑和规则
type CacheService struct {
	evictionStrategy EvictionStrategy
}

// NewCacheService 创建缓存服务
// evictionStrategy: 淘汰策略
func NewCacheService(evictionStrategy EvictionStrategy) *CacheService {
	return &CacheService{
		evictionStrategy: evictionStrategy,
	}
}

// ValidateKey 验证缓存键
// key: 要验证的键
// 返回: 验证错误
func (s *CacheService) ValidateKey(key string) error {
	_, err := NewCacheKey(key)
	return err
}

// ValidateExpiration 验证过期时间
// duration: 过期时间间隔
// 返回: 验证错误
func (s *CacheService) ValidateExpiration(duration time.Duration) error {
	_, err := NewExpiration(duration)
	return err
}

// ShouldEvict 判断是否需要淘汰
// instance: 缓存实例
// 返回: 是否需要淘汰
func (s *CacheService) ShouldEvict(instance *CacheInstance) bool {
	return instance.IsFull()
}

// SelectForEviction 选择要淘汰的条目
// instance: 缓存实例
// 返回: 要淘汰的条目
func (s *CacheService) SelectForEviction(instance *CacheInstance) *Entry {
	var entries []*Entry
	for _, entry := range instance.entries {
		entries = append(entries, entry)
	}
	return s.evictionStrategy.SelectForEviction(entries)
}

// CleanExpiredEntries 清理过期条目
// instance: 缓存实例
// 返回: 清理的条目数
func (s *CacheService) CleanExpiredEntries(instance *CacheInstance) int {
	return instance.CleanExpiredEntries(time.Now())
}

// WriteBackService 写回缓存领域服务
// 专门处理写回缓存的业务逻辑
type WriteBackService struct {
	*CacheService
	flushInterval time.Duration
	batchSize     int
}

// NewWriteBackService 创建写回缓存服务
// evictionStrategy: 淘汰策略
// flushInterval: 刷新间隔
// batchSize: 批量大小
func NewWriteBackService(evictionStrategy EvictionStrategy, flushInterval time.Duration, batchSize int) *WriteBackService {
	return &WriteBackService{
		CacheService:  NewCacheService(evictionStrategy),
		flushInterval: flushInterval,
		batchSize:     batchSize,
	}
}

// FlushInterval 获取刷新间隔
func (s *WriteBackService) FlushInterval() time.Duration {
	return s.flushInterval
}

// BatchSize 获取批量大小
func (s *WriteBackService) BatchSize() int {
	return s.batchSize
}

// ShouldFlush 判断是否需要刷新
// instance: 缓存实例
// lastFlushTime: 上次刷新时间
// 返回: 是否需要刷新
func (s *WriteBackService) ShouldFlush(instance *CacheInstance, lastFlushTime time.Time) bool {
	// 检查是否到了刷新时间
	if time.Since(lastFlushTime) >= s.flushInterval {
		return true
	}
	
	// 检查脏数据数量是否达到批量大小
	dirtyEntries := instance.GetDirtyEntries()
	return len(dirtyEntries) >= s.batchSize
}

// GetFlushBatch 获取要刷新的批次
// instance: 缓存实例
// 返回: 要刷新的条目列表
func (s *WriteBackService) GetFlushBatch(instance *CacheInstance) []*Entry {
	dirtyEntries := instance.GetDirtyEntries()
	
	// 如果脏数据少于批量大小，返回所有脏数据
	if len(dirtyEntries) <= s.batchSize {
		return dirtyEntries
	}
	
	// 返回批量大小的脏数据
	return dirtyEntries[:s.batchSize]
}

// ValidateFlushOperation 验证刷新操作
// entries: 要刷新的条目
// storer: 存储函数
// 返回: 验证错误
func (s *WriteBackService) ValidateFlushOperation(entries []*Entry, storer func(ctx context.Context, key string, val any) error) error {
	if len(entries) == 0 {
		return fmt.Errorf("没有需要刷新的条目")
	}
	
	if storer == nil {
		return fmt.Errorf("存储函数不能为空")
	}
	
	return nil
}

// ProcessFlushResult 处理刷新结果
// instance: 缓存实例
// entries: 已刷新的条目
// errors: 刷新错误列表
func (s *WriteBackService) ProcessFlushResult(instance *CacheInstance, entries []*Entry, errors []error) {
	successCount := 0
	
	for i, entry := range entries {
		if i < len(errors) && errors[i] != nil {
			// 刷新失败，保持脏状态
			continue
		}
		
		// 刷新成功，标记为干净
		entry.MarkClean()
		successCount++
	}
	
	// 更新统计信息
	if successCount > 0 {
		instance.MarkFlush()
	}
}

// CachePolicy 缓存策略
// 定义缓存的各种策略和配置
type CachePolicy struct {
	maxSize       int64
	maxMemory     int64
	defaultTTL    time.Duration
	evictionStrategy EvictionStrategy
	enableWriteBack  bool
	writeBackConfig  *WriteBackConfig
}

// WriteBackConfig 写回配置
type WriteBackConfig struct {
	FlushInterval time.Duration
	BatchSize     int
	MaxRetries    int
	RetryDelay    time.Duration
}

// NewCachePolicy 创建缓存策略
func NewCachePolicy() *CachePolicy {
	return &CachePolicy{
		maxSize:          1000,
		maxMemory:        100 * 1024 * 1024, // 100MB
		defaultTTL:       time.Hour,
		evictionStrategy: NewLRUEvictionStrategy(),
		enableWriteBack:  false,
	}
}

// WithMaxSize 设置最大条目数
func (p *CachePolicy) WithMaxSize(maxSize int64) *CachePolicy {
	p.maxSize = maxSize
	return p
}

// WithMaxMemory 设置最大内存
func (p *CachePolicy) WithMaxMemory(maxMemory int64) *CachePolicy {
	p.maxMemory = maxMemory
	return p
}

// WithDefaultTTL 设置默认TTL
func (p *CachePolicy) WithDefaultTTL(ttl time.Duration) *CachePolicy {
	p.defaultTTL = ttl
	return p
}

// WithEvictionStrategy 设置淘汰策略
func (p *CachePolicy) WithEvictionStrategy(strategy EvictionStrategy) *CachePolicy {
	p.evictionStrategy = strategy
	return p
}

// WithWriteBack 启用写回模式
func (p *CachePolicy) WithWriteBack(config *WriteBackConfig) *CachePolicy {
	p.enableWriteBack = true
	p.writeBackConfig = config
	return p
}

// MaxSize 获取最大条目数
func (p *CachePolicy) MaxSize() int64 {
	return p.maxSize
}

// MaxMemory 获取最大内存
func (p *CachePolicy) MaxMemory() int64 {
	return p.maxMemory
}

// DefaultTTL 获取默认TTL
func (p *CachePolicy) DefaultTTL() time.Duration {
	return p.defaultTTL
}

// EvictionStrategy 获取淘汰策略
func (p *CachePolicy) EvictionStrategy() EvictionStrategy {
	return p.evictionStrategy
}

// IsWriteBackEnabled 检查是否启用写回
func (p *CachePolicy) IsWriteBackEnabled() bool {
	return p.enableWriteBack
}

// WriteBackConfig 获取写回配置
func (p *CachePolicy) GetWriteBackConfig() *WriteBackConfig {
	return p.writeBackConfig
}
