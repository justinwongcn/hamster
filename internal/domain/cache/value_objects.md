# value_objects.go - 缓存值对象定义

## 文件概述

`value_objects.go` 定义了缓存系统中的值对象（Value Objects），包括缓存键、缓存值、过期时间和统计信息等。值对象是不可变的，通过属性值来标识，封装了相关的业务规则和验证逻辑。

## 核心功能

### 1. CacheKey - 缓存键值对象

封装缓存键的业务规则和验证逻辑：

```go
type CacheKey struct {
    value string
}

func NewCacheKey(key string) (CacheKey, error)
func (k CacheKey) String() string
func (k CacheKey) IsEmpty() bool
func (k CacheKey) Length() int
func (k CacheKey) Hash() uint64
func (k CacheKey) Equals(other CacheKey) bool
```

**验证规则：**

- 键不能为空字符串
- 键长度不能超过250个字符
- 键不能包含控制字符（\x00-\x1F, \x7F）

### 2. CacheValue - 缓存值值对象

封装缓存值的类型安全和序列化逻辑：

```go
type CacheValue struct {
    data any
    size int64
}

func NewCacheValue(value any) (CacheValue, error)
func (v CacheValue) Data() any
func (v CacheValue) Size() int64
func (v CacheValue) IsNil() bool
func (v CacheValue) String() string
func (v CacheValue) Equals(other CacheValue) bool
```

**特点：**

- 支持任意类型的数据
- 自动计算数据大小
- 提供类型安全的访问方法

### 3. Expiration - 过期时间值对象

封装过期时间的计算和判断逻辑：

```go
type Expiration struct {
    duration time.Duration
}

func NewExpiration(duration time.Duration) Expiration
func (e Expiration) Duration() time.Duration
func (e Expiration) IsZero() bool
func (e Expiration) ExpiresAt(from time.Time) time.Time
func (e Expiration) IsExpired(createdAt, now time.Time) bool
func (e Expiration) RemainingTime(createdAt, now time.Time) time.Duration
```

**特点：**

- 零值表示永不过期
- 提供过期判断和剩余时间计算
- 支持基于创建时间的过期检查

### 4. CacheStats - 缓存统计信息值对象

封装缓存的运行时统计数据：

```go
type CacheStats struct {
    hits        int64
    misses      int64
    sets        int64
    deletes     int64
    evictions   int64
    size        int64
    memoryUsage int64
}

func NewCacheStats() CacheStats
func (s CacheStats) Hits() int64
func (s CacheStats) Misses() int64
func (s CacheStats) HitRate() float64
func (s CacheStats) TotalRequests() int64
func (s CacheStats) IncrementHits() CacheStats
func (s CacheStats) IncrementMisses() CacheStats
```

**统计指标：**

- 命中次数和未命中次数
- 设置、删除、淘汰次数
- 缓存大小和内存使用量
- 命中率计算

## 使用示例

### 创建和使用缓存键

```go
// 创建缓存键
key, err := NewCacheKey("user:123")
if err != nil {
    log.Printf("创建缓存键失败: %v", err)
    return
}

// 使用缓存键
fmt.Printf("键值: %s\n", key.String())
fmt.Printf("键长度: %d\n", key.Length())
fmt.Printf("键哈希: %d\n", key.Hash())

// 比较缓存键
key2, _ := NewCacheKey("user:123")
if key.Equals(key2) {
    fmt.Println("键相等")
}
```

### 创建和使用缓存值

```go
// 创建不同类型的缓存值
stringValue, _ := NewCacheValue("Hello World")
intValue, _ := NewCacheValue(42)
structValue, _ := NewCacheValue(User{ID: 123, Name: "John"})

// 获取值信息
fmt.Printf("字符串值大小: %d bytes\n", stringValue.Size())
fmt.Printf("整数值: %v\n", intValue.Data())
fmt.Printf("结构体值: %s\n", structValue.String())

// 检查值是否为nil
if !stringValue.IsNil() {
    fmt.Println("值不为空")
}
```

### 使用过期时间

```go
// 创建过期时间
expiration := NewExpiration(time.Hour)

// 检查过期状态
createdAt := time.Now()
now := time.Now().Add(30 * time.Minute)

if expiration.IsExpired(createdAt, now) {
    fmt.Println("已过期")
} else {
    remaining := expiration.RemainingTime(createdAt, now)
    fmt.Printf("剩余时间: %v\n", remaining)
}

// 永不过期的设置
neverExpire := NewExpiration(0)
fmt.Printf("是否永不过期: %v\n", neverExpire.IsZero())
```

### 使用统计信息

```go
// 创建统计信息
stats := NewCacheStats()

// 更新统计
stats = stats.IncrementHits()
stats = stats.IncrementMisses()
stats = stats.IncrementSets()

// 获取统计数据
fmt.Printf("命中次数: %d\n", stats.Hits())
fmt.Printf("未命中次数: %d\n", stats.Misses())
fmt.Printf("命中率: %.2f%%\n", stats.HitRate()*100)
fmt.Printf("总请求数: %d\n", stats.TotalRequests())
```

## 注意事项

### 1. 不可变性

```go
// ✅ 正确：值对象是不可变的
stats := NewCacheStats()
newStats := stats.IncrementHits() // 返回新实例
fmt.Printf("原始命中数: %d\n", stats.Hits())     // 0
fmt.Printf("新的命中数: %d\n", newStats.Hits())   // 1

// ❌ 错误：不能直接修改值对象
// stats.hits++ // 编译错误，字段不可访问
```

### 2. 键的验证

```go
// ✅ 正确：处理验证错误
key, err := NewCacheKey("user:123")
if err != nil {
    return fmt.Errorf("无效的缓存键: %w", err)
}

// ❌ 错误：忽略验证错误
key, _ := NewCacheKey("") // 会返回错误但被忽略
```

### 3. 值的大小计算

```go
// 大对象的大小计算可能不准确
largeObject := make([]byte, 1024*1024) // 1MB
value, _ := NewCacheValue(largeObject)

// 注意：Size()返回的是估算值，不是精确的内存占用
fmt.Printf("估算大小: %d bytes\n", value.Size())
```

### 4. 过期时间的精度

```go
// 过期检查基于时间比较，存在精度限制
expiration := NewExpiration(time.Nanosecond)
createdAt := time.Now()
time.Sleep(time.Microsecond)
now := time.Now()

// 可能由于时间精度问题导致判断不准确
isExpired := expiration.IsExpired(createdAt, now)
```

### 5. 统计信息的线程安全

```go
// 值对象本身是不可变的，但在并发环境下需要注意
var stats CacheStats
var mu sync.Mutex

// 并发更新统计信息
go func() {
    mu.Lock()
    stats = stats.IncrementHits()
    mu.Unlock()
}()

go func() {
    mu.Lock()
    stats = stats.IncrementMisses()
    mu.Unlock()
}()
```

## 扩展指南

### 添加新的值对象

```go
// 1. 定义值对象结构
type CacheMetadata struct {
    tags        []string
    priority    int
    source      string
    lastAccess  time.Time
}

// 2. 实现构造函数
func NewCacheMetadata(tags []string, priority int, source string) (CacheMetadata, error) {
    if priority < 0 {
        return CacheMetadata{}, errors.New("优先级不能为负数")
    }
    
    return CacheMetadata{
        tags:       append([]string(nil), tags...), // 复制切片
        priority:   priority,
        source:     source,
        lastAccess: time.Now(),
    }, nil
}

// 3. 实现访问方法
func (m CacheMetadata) Tags() []string {
    return append([]string(nil), m.tags...) // 返回副本
}

func (m CacheMetadata) Priority() int {
    return m.priority
}

// 4. 实现业务方法
func (m CacheMetadata) HasTag(tag string) bool {
    for _, t := range m.tags {
        if t == tag {
            return true
        }
    }
    return false
}

func (m CacheMetadata) UpdateLastAccess() CacheMetadata {
    return CacheMetadata{
        tags:       m.tags,
        priority:   m.priority,
        source:     m.source,
        lastAccess: time.Now(),
    }
}
```

### 自定义验证规则

```go
// 扩展CacheKey的验证规则
func NewRestrictedCacheKey(key string, allowedPrefixes []string) (CacheKey, error) {
    // 先进行基本验证
    cacheKey, err := NewCacheKey(key)
    if err != nil {
        return CacheKey{}, err
    }
    
    // 添加自定义验证
    hasValidPrefix := false
    for _, prefix := range allowedPrefixes {
        if strings.HasPrefix(key, prefix) {
            hasValidPrefix = true
            break
        }
    }
    
    if !hasValidPrefix {
        return CacheKey{}, errors.New("键必须以允许的前缀开头")
    }
    
    return cacheKey, nil
}
```
