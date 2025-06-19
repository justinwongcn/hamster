# LRU缓存淘汰策略

## 概述
LRU（Least Recently Used）策略根据数据最近被访问的时间来进行淘汰，最近最少使用的数据会最先被淘汰。适用于需要频繁访问最新数据的场景。

## 核心功能
- 维护key的访问顺序
- 淘汰最近最少使用的key
- 支持动态更新key的访问时间
- 线程安全的key访问和移除操作

## 主要类型
### LRUPolicy
实现LRU淘汰策略的核心结构体

```go
type LRUPolicy struct {
    list *tools.LinkedList[string] // 维护key的访问顺序
    keys map[string]int            // 记录key在链表中的位置
}
```

## 主要方法
### NewLRUPolicy()
创建新的LRU策略实例

```go
func NewLRUPolicy() *LRUPolicy
```

### KeyAccessed(key string)
记录key被访问

```go
func (l *LRUPolicy) KeyAccessed(key string)
```

### Evict()
执行淘汰并返回被淘汰的key

```go
func (l *LRUPolicy) Evict() string
```

### Remove(key string)
移除指定key

```go
func (l *LRUPolicy) Remove(key string)
```

### Has(key string)
检查key是否存在

```go
func (l *LRUPolicy) Has(key string) bool
```

## 使用示例
```go
policy := NewLRUPolicy()
policy.KeyAccessed("key1")
policy.KeyAccessed("key2")
policy.KeyAccessed("key1") // key1被移动到链表尾部
// 淘汰key2
evicted := policy.Evict()
```

## 注意事项
1. 每次访问key都会将其移动到链表尾部
2. 淘汰总是从链表头部开始
3. 删除或淘汰key后会自动更新其他key的位置信息