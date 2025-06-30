# lru_policy.go - LRU淘汰策略实现

## 文件概述

`lru_policy.go` 实现了LRU（Least Recently Used）缓存淘汰策略。采用双向链表+哈希表的经典实现方案，提供O(1)
时间复杂度的访问、插入和删除操作。该实现线程安全，支持并发访问，并可选择性地设置容量限制。

## 核心功能

### 1. 数据结构设计

#### lruNode 链表节点

```go
type lruNode struct {
    key  string   // 节点存储的键
    prev *lruNode // 前驱节点指针
    next *lruNode // 后继节点指针
}
```

**设计特点：**

- 双向链表节点，支持O(1)插入删除
- 只存储键，值存储在外部缓存中
- 简洁的结构设计，减少内存占用

#### LRUPolicy 策略结构

```go
type LRUPolicy struct {
    capacity int                 // 容量限制，0表示无限制
    size     int                 // 当前大小
    cache    map[string]*lruNode // 哈希表，快速定位节点
    head     *lruNode            // 头节点（最近使用）
    tail     *lruNode            // 尾节点（最久未使用）
    mutex    sync.RWMutex        // 读写锁，保证并发安全
}
```

**设计特点：**

- 哈希表提供O(1)查找性能
- 双向链表维护访问顺序
- 头尾哨兵节点简化边界处理
- 读写锁保证线程安全

### 2. 构造函数

#### NewLRUPolicy

```go
func NewLRUPolicy(capacity ...int) *LRUPolicy
```

创建新的LRU策略实例，支持可选的容量限制。

**参数：**

- `capacity`: 可选参数，容量限制，0或不传表示无限制

**实现逻辑：**

1. 解析容量参数
2. 创建头尾哨兵节点
3. 建立双向链表连接
4. 初始化哈希表和元数据

**示例：**

```go
// 无限制容量
policy := NewLRUPolicy()

// 限制容量为100
policy := NewLRUPolicy(100)
```

### 3. 核心操作

#### KeyAccessed - 记录访问

```go
func (l *LRUPolicy) KeyAccessed(ctx context.Context, key string) error
```

记录键被访问，将其移动到链表头部（最近使用位置）。

**实现逻辑：**

1. 获取写锁保证线程安全
2. 检查键是否已存在
3. 如果存在，移动到头部
4. 如果不存在，创建新节点并添加到头部
5. 检查容量限制，必要时自动淘汰

**示例：**

```go
err := policy.KeyAccessed(ctx, "user:123")
if err != nil {
    log.Printf("记录访问失败: %v", err)
}
```

#### Evict - 执行淘汰

```go
func (l *LRUPolicy) Evict(ctx context.Context) (string, error)
```

执行淘汰操作，移除最久未使用的键（链表尾部）。

**实现逻辑：**

1. 获取写锁
2. 检查是否有可淘汰的节点
3. 移除尾部节点
4. 从哈希表中删除对应条目
5. 更新大小计数

**示例：**

```go
evictedKey, err := policy.Evict(ctx)
if err != nil {
    log.Printf("淘汰失败: %v", err)
} else if evictedKey != "" {
    log.Printf("淘汰了键: %s", evictedKey)
}
```

#### Remove - 移除指定键

```go
func (l *LRUPolicy) Remove(ctx context.Context, key string) error
```

从策略中移除指定的键。

**示例：**

```go
err := policy.Remove(ctx, "user:123")
if err != nil {
    log.Printf("移除失败: %v", err)
}
```

### 4. 查询操作

#### Has - 检查键是否存在

```go
func (l *LRUPolicy) Has(ctx context.Context, key string) (bool, error)
```

检查指定键是否在策略中。

#### Size - 获取当前大小

```go
func (l *LRUPolicy) Size(ctx context.Context) (int, error)
```

返回当前策略中键的数量。

#### Keys - 获取所有键

```go
func (l *LRUPolicy) Keys(ctx context.Context) ([]string, error)
```

返回所有键的列表，按访问顺序排序（最近使用的在前）。

### 5. 内部辅助方法

#### addToHead - 添加到头部

```go
func (l *LRUPolicy) addToHead(node *lruNode)
```

将节点添加到链表头部。

#### removeNode - 移除节点

```go
func (l *LRUPolicy) removeNode(node *lruNode)
```

从链表中移除指定节点。

#### moveToHead - 移动到头部

```go
func (l *LRUPolicy) moveToHead(node *lruNode)
```

将节点移动到链表头部。

#### removeTail - 移除尾部

```go
func (l *LRUPolicy) removeTail() *lruNode
```

移除并返回链表尾部节点。

## 算法原理

### LRU算法核心思想

1. **最近使用原则**: 最近被访问的数据更可能再次被访问
2. **淘汰策略**: 当容量不足时，优先淘汰最久未使用的数据
3. **访问更新**: 每次访问都会更新数据的使用时间

### 数据结构选择

1. **双向链表**: 维护访问顺序，支持O(1)插入删除
2. **哈希表**: 提供O(1)查找性能
3. **哨兵节点**: 简化边界条件处理

### 操作复杂度

- **查找**: O(1) - 哈希表查找
- **插入**: O(1) - 链表头部插入
- **删除**: O(1) - 直接节点删除
- **更新**: O(1) - 移动到头部

## 使用示例

### 1. 基本使用

```go
// 创建LRU策略
policy := NewLRUPolicy(3) // 容量限制为3

ctx := context.Background()

// 记录访问
policy.KeyAccessed(ctx, "key1")
policy.KeyAccessed(ctx, "key2")
policy.KeyAccessed(ctx, "key3")

// 检查大小
size, _ := policy.Size(ctx)
fmt.Printf("当前大小: %d\n", size) // 输出: 3

// 再次访问key1，将其移动到头部
policy.KeyAccessed(ctx, "key1")

// 添加新键，触发淘汰
policy.KeyAccessed(ctx, "key4")

// key2应该被淘汰了
has, _ := policy.Has(ctx, "key2")
fmt.Printf("key2是否存在: %v\n", has) // 输出: false
```

### 2. 手动淘汰

```go
policy := NewLRUPolicy()

// 添加一些键
policy.KeyAccessed(ctx, "user:1")
policy.KeyAccessed(ctx, "user:2")
policy.KeyAccessed(ctx, "user:3")

// 手动淘汰最久未使用的键
evictedKey, err := policy.Evict(ctx)
if err == nil && evictedKey != "" {
    fmt.Printf("淘汰了: %s\n", evictedKey) // 输出: user:1
}
```

### 3. 获取访问顺序

```go
policy := NewLRUPolicy()

// 添加键
policy.KeyAccessed(ctx, "a")
policy.KeyAccessed(ctx, "b")
policy.KeyAccessed(ctx, "c")
policy.KeyAccessed(ctx, "a") // 重新访问a

// 获取所有键（按访问顺序）
keys, err := policy.Keys(ctx)
if err == nil {
    fmt.Printf("访问顺序: %v\n", keys) // 输出: [a c b]
}
```

### 4. 容量管理

```go
// 测试容量限制
policy := NewLRUPolicy(2)

policy.KeyAccessed(ctx, "key1")
policy.KeyAccessed(ctx, "key2")

size, _ := policy.Size(ctx)
fmt.Printf("添加2个键后大小: %d\n", size) // 输出: 2

// 添加第3个键，应该自动淘汰最久未使用的
policy.KeyAccessed(ctx, "key3")

size, _ = policy.Size(ctx)
fmt.Printf("添加第3个键后大小: %d\n", size) // 输出: 2

// key1应该被自动淘汰
has, _ := policy.Has(ctx, "key1")
fmt.Printf("key1是否还存在: %v\n", has) // 输出: false
```

### 5. 并发使用

```go
policy := NewLRUPolicy(100)

// 并发访问
var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        
        for j := 0; j < 100; j++ {
            key := fmt.Sprintf("key_%d_%d", id, j)
            policy.KeyAccessed(ctx, key)
        }
    }(i)
}

wg.Wait()

size, _ := policy.Size(ctx)
fmt.Printf("并发访问后大小: %d\n", size)
```

## 性能特性

### 时间复杂度

- **KeyAccessed**: O(1) - 哈希查找 + 链表操作
- **Evict**: O(1) - 直接移除尾部节点
- **Remove**: O(1) - 哈希查找 + 链表删除
- **Has**: O(1) - 哈希查找
- **Size**: O(1) - 直接返回计数

### 空间复杂度

- **存储空间**: O(n) - n为键的数量
- **额外空间**: O(1) - 固定的哨兵节点和元数据

### 并发性能

- **读操作**: 使用读锁，支持并发读取
- **写操作**: 使用写锁，保证数据一致性
- **锁粒度**: 整个策略级别的锁

## 注意事项

### 1. 容量设置

```go
// ✅ 推荐：根据实际需求设置合理容量
policy := NewLRUPolicy(1000) // 根据内存和性能需求设置

// ❌ 避免：容量过小导致频繁淘汰
policy := NewLRUPolicy(1) // 容量太小

// ❌ 避免：无限制容量可能导致内存泄漏
policy := NewLRUPolicy() // 在高负载场景下要小心
```

### 2. 并发安全

```go
// ✅ 正确：LRU策略本身是线程安全的
go func() {
    policy.KeyAccessed(ctx, "key1")
}()

go func() {
    policy.Evict(ctx)
}()

// ❌ 避免：不要在外部加锁
var mu sync.Mutex
mu.Lock()
policy.KeyAccessed(ctx, "key") // 不必要的锁
mu.Unlock()
```

### 3. 错误处理

```go
// ✅ 推荐：检查操作结果
evictedKey, err := policy.Evict(ctx)
if err != nil {
    log.Printf("淘汰失败: %v", err)
    return err
}

if evictedKey == "" {
    log.Println("没有可淘汰的键")
}
```

### 4. 内存管理

```go
// ✅ 推荐：及时清理不需要的策略
policy := NewLRUPolicy(100)
// 使用完毕后
policy = nil // 帮助GC回收

// ⚠️ 注意：避免在策略中存储大对象的引用
// LRU策略只存储键，值应该存储在外部缓存中
```
