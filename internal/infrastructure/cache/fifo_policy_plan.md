# FIFO策略实现计划

## 1. 文件位置
`internal/infrastructure/cache/fifo_policy.go`

## 2. 包和导入
```go
package cache

import (
    "github.com/justinwongcn/hamster/internal/domain/tools"
)
```

## 3. FIFOPolicy结构体
```go
// FIFOPolicy 实现FIFO淘汰策略
type FIFOPolicy struct {
    list *tools.LinkedList[string] // 维护key的添加顺序
    keys map[string]int            // 记录key在链表中的位置
}
```

## 4. 构造函数
```go
func NewFIFOPolicy() *FIFOPolicy {
    return &FIFOPolicy{
        list: tools.NewLinkedList[string](),
        keys: make(map[string]int),
    }
}
```

## 5. KeyAccessed方法
```go
func (f *FIFOPolicy) KeyAccessed(key string) {
    // 如果key已存在，不更新位置（FIFO特性）
    if _, exists := f.keys[key]; exists {
        return
    }
    // 新key添加到链表尾部
    f.list.Append(key)
    f.keys[key] = f.list.Len() - 1
}
```

## 6. Evict方法
```go
func (f *FIFOPolicy) Evict() string {
    if f.list.Len() == 0 {
        return ""
    }
    // 移除链表头部的key（最早添加）
    key, _ := f.list.Get(0)
    f.list.Delete(0)
    delete(f.keys, key)
    return key
}
```

## 7. Remove方法
```go
func (f *FIFOPolicy) Remove(key string) {
    if idx, exists := f.keys[key]; exists {
        f.list.Delete(idx)
        delete(f.keys, key)
    }
}
```

## 8. Has方法
```go
func (f *FIFOPolicy) Has(key string) bool {
    _, exists := f.keys[key]
    return exists
}
```

## 9. 与LRU策略的关键区别
| 方法         | LRU策略行为               | FIFO策略行为               |
|--------------|--------------------------|---------------------------|
| KeyAccessed  | 移动key到链表尾部         | 仅添加新key，不移动已存在key |
| Evict        | 移除链表头部（最近最少使用）| 移除链表头部（最早添加）    |

## 10. 测试计划
1. 创建FIFOPolicy实例
2. 测试KeyAccessed：
   - 添加新key应出现在链表尾部
   - 重复添加已存在key不应改变位置
3. 测试Evict：
   - 空链表返回空字符串
   - 非空链表返回最早添加的key
4. 测试Remove：
   - 移除存在的key
   - 移除不存在的key（无操作）
5. 测试Has：
   - 检查存在的key返回true
   - 检查不存在的key返回false