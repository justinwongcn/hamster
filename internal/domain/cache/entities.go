package cache

import (
	"time"
)

// Entry 缓存条目实体
// 代表缓存中的一个条目，包含键、值、过期时间等信息
// 这是缓存领域的核心实体，封装了缓存条目的业务逻辑
type Entry struct {
	key        CacheKey
	value      CacheValue
	expiration Expiration
	createdAt  time.Time
	accessedAt time.Time
	accessCount int64
}

// NewEntry 创建新的缓存条目
// key: 缓存键
// value: 缓存值
// expiration: 过期时间
// 返回: Entry实例
func NewEntry(key CacheKey, value CacheValue, expiration Expiration) *Entry {
	now := time.Now()
	return &Entry{
		key:         key,
		value:       value,
		expiration:  expiration,
		createdAt:   now,
		accessedAt:  now,
		accessCount: 0,
	}
}

// Key 获取缓存键
func (e *Entry) Key() CacheKey {
	return e.key
}

// Value 获取缓存值
func (e *Entry) Value() CacheValue {
	return e.value
}

// Expiration 获取过期时间
func (e *Entry) Expiration() Expiration {
	return e.expiration
}

// CreatedAt 获取创建时间
func (e *Entry) CreatedAt() time.Time {
	return e.createdAt
}

// AccessedAt 获取最后访问时间
func (e *Entry) AccessedAt() time.Time {
	return e.accessedAt
}

// AccessCount 获取访问次数
func (e *Entry) AccessCount() int64 {
	return e.accessCount
}

// IsExpired 检查是否已过期
// now: 当前时间
// 返回: 是否已过期
func (e *Entry) IsExpired(now time.Time) bool {
	return e.expiration.IsExpired(e.createdAt, now)
}

// IsDirty 检查是否为脏数据
func (e *Entry) IsDirty() bool {
	return e.value.IsDirty()
}

// UpdateValue 更新缓存值
// newValue: 新的缓存值
func (e *Entry) UpdateValue(newValue CacheValue) {
	e.value = newValue
	e.accessedAt = time.Now()
}

// MarkAccessed 标记为已访问
// 更新访问时间和访问次数
func (e *Entry) MarkAccessed() {
	e.accessedAt = time.Now()
	e.accessCount++
}

// MarkClean 标记为干净数据
func (e *Entry) MarkClean() {
	e.value = e.value.MarkClean()
}

// MarkDirty 标记为脏数据
func (e *Entry) MarkDirty() {
	e.value = e.value.MarkDirty()
}

// Age 计算条目的年龄
// now: 当前时间
// 返回: 条目年龄
func (e *Entry) Age(now time.Time) time.Duration {
	return now.Sub(e.createdAt)
}

// IdleTime 计算条目的空闲时间
// now: 当前时间
// 返回: 空闲时间
func (e *Entry) IdleTime(now time.Time) time.Duration {
	return now.Sub(e.accessedAt)
}

// Clone 克隆缓存条目
// 返回: 新的Entry实例
func (e *Entry) Clone() *Entry {
	return &Entry{
		key:         e.key,
		value:       e.value,
		expiration:  e.expiration,
		createdAt:   e.createdAt,
		accessedAt:  e.accessedAt,
		accessCount: e.accessCount,
	}
}

// CacheInstance 缓存实例实体
// 代表一个缓存实例，管理多个缓存条目
// 这是缓存领域的聚合根，负责协调缓存条目的操作
type CacheInstance struct {
	name     string
	entries  map[string]*Entry
	stats    CacheStats
	maxSize  int64
	maxMemory int64
	currentSize int64
	currentMemory int64
}

// NewCacheInstance 创建新的缓存实例
// name: 缓存实例名称
// maxSize: 最大条目数，0表示无限制
// maxMemory: 最大内存使用量，0表示无限制
// 返回: CacheInstance实例
func NewCacheInstance(name string, maxSize, maxMemory int64) *CacheInstance {
	return &CacheInstance{
		name:          name,
		entries:       make(map[string]*Entry),
		stats:         NewCacheStats(),
		maxSize:       maxSize,
		maxMemory:     maxMemory,
		currentSize:   0,
		currentMemory: 0,
	}
}

// Name 获取缓存实例名称
func (c *CacheInstance) Name() string {
	return c.name
}

// Stats 获取缓存统计信息
func (c *CacheInstance) Stats() CacheStats {
	return c.stats
}

// Size 获取当前条目数
func (c *CacheInstance) Size() int64 {
	return c.currentSize
}

// Memory 获取当前内存使用量
func (c *CacheInstance) Memory() int64 {
	return c.currentMemory
}

// MaxSize 获取最大条目数限制
func (c *CacheInstance) MaxSize() int64 {
	return c.maxSize
}

// MaxMemory 获取最大内存限制
func (c *CacheInstance) MaxMemory() int64 {
	return c.maxMemory
}

// IsFull 检查是否已满
func (c *CacheInstance) IsFull() bool {
	if c.maxSize > 0 && c.currentSize >= c.maxSize {
		return true
	}
	if c.maxMemory > 0 && c.currentMemory >= c.maxMemory {
		return true
	}
	return false
}

// GetEntry 获取缓存条目
// key: 缓存键
// 返回: 缓存条目和是否存在
func (c *CacheInstance) GetEntry(key CacheKey) (*Entry, bool) {
	entry, exists := c.entries[key.String()]
	if exists {
		entry.MarkAccessed()
		c.stats = c.stats.IncrementHits()
	} else {
		c.stats = c.stats.IncrementMisses()
	}
	return entry, exists
}

// SetEntry 设置缓存条目
// entry: 缓存条目
func (c *CacheInstance) SetEntry(entry *Entry) {
	keyStr := entry.Key().String()
	
	// 如果条目已存在，先移除旧条目
	if oldEntry, exists := c.entries[keyStr]; exists {
		c.removeEntry(oldEntry)
	}
	
	c.entries[keyStr] = entry
	c.currentSize++
	c.stats = c.stats.IncrementSets()
	
	// 如果是脏数据，增加脏写统计
	if entry.IsDirty() {
		c.stats = c.stats.IncrementDirtyWrites()
	}
}

// RemoveEntry 移除缓存条目
// key: 缓存键
// 返回: 被移除的条目和是否存在
func (c *CacheInstance) RemoveEntry(key CacheKey) (*Entry, bool) {
	entry, exists := c.entries[key.String()]
	if exists {
		c.removeEntry(entry)
		c.stats = c.stats.IncrementDeletes()
	}
	return entry, exists
}

// removeEntry 内部移除条目方法
func (c *CacheInstance) removeEntry(entry *Entry) {
	delete(c.entries, entry.Key().String())
	c.currentSize--
}

// GetDirtyEntries 获取所有脏数据条目
// 返回: 脏数据条目列表
func (c *CacheInstance) GetDirtyEntries() []*Entry {
	var dirtyEntries []*Entry
	for _, entry := range c.entries {
		if entry.IsDirty() {
			dirtyEntries = append(dirtyEntries, entry)
		}
	}
	return dirtyEntries
}

// GetExpiredEntries 获取所有过期条目
// now: 当前时间
// 返回: 过期条目列表
func (c *CacheInstance) GetExpiredEntries(now time.Time) []*Entry {
	var expiredEntries []*Entry
	for _, entry := range c.entries {
		if entry.IsExpired(now) {
			expiredEntries = append(expiredEntries, entry)
		}
	}
	return expiredEntries
}

// CleanExpiredEntries 清理过期条目
// now: 当前时间
// 返回: 被清理的条目数
func (c *CacheInstance) CleanExpiredEntries(now time.Time) int {
	expiredEntries := c.GetExpiredEntries(now)
	for _, entry := range expiredEntries {
		c.removeEntry(entry)
		c.stats = c.stats.IncrementEvictions()
	}
	return len(expiredEntries)
}

// MarkFlush 标记刷新操作
func (c *CacheInstance) MarkFlush() {
	c.stats = c.stats.IncrementFlushes()
}
