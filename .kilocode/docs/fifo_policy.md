# FIFO缓存淘汰策略

## 概述
FIFO（First In First Out）策略按照数据进入缓存的顺序进行淘汰，最早进入的数据会最先被淘汰。适用于对数据访问顺序有严格要求的场景。

## 核心功能
- 按照添加顺序维护缓存key
- 淘汰最早添加的key
- 线程安全的key访问和移除操作

## 主要类型
### FIFOPolicy
实现FIFO淘汰策略的核心结构体

```go
type FIFOPolicy struct {
    list *tools.LinkedList[string] // 维护key的添加顺序
    keys map[string]int            // 记录key在链表中的位置
}
```

## 主要方法
### NewFIFOPolicy()
创建新的FIFO策略实例

```go
func NewFIFOPolicy() *FIFOPolicy
```

### KeyAccessed(key string)
记录key被访问

```go
func (f *FIFOPolicy) KeyAccessed(key string)
```

### Evict()
执行淘汰并返回被淘汰的key

```go
func (f *FIFOPolicy) Evict() string
```

### Remove(key string)
移除指定key

```go
func (f *FIFOPolicy) Remove(key string)
```

### Has(key string)
检查key是否存在

```go
func (f *FIFOPolicy) Has(key string) bool
```

## 使用示例
```go
policy := NewFIFOPolicy()
policy.KeyAccessed("key1")
policy.KeyAccessed("key2")
// 淘汰key1
evicted := policy.Evict()
```

## 注意事项
1. 新添加的key会放在链表尾部
2. 已有key被访问时不会改变其位置
3. 淘汰总是从链表头部开始