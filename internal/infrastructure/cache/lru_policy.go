package cache

import (
    "github.com/justinwongcn/hamster/internal/domain/tools"
)

// LRUPolicy 实现LRU淘汰策略
type LRUPolicy struct {
    list *tools.LinkedList[string] // 用于维护key的访问顺序
    keys map[string]int            // 记录key在链表中的位置
}

// NewLRUPolicy 创建新的LRU策略实例
func NewLRUPolicy() *LRUPolicy {
    return &LRUPolicy{
        list: tools.NewLinkedList[string](),
        keys: make(map[string]int),
    }
}

// KeyAccessed 记录key被访问
func (l *LRUPolicy) KeyAccessed(key string) {
    // 如果key已存在，先移除
    if idx, exists := l.keys[key]; exists {
        l.list.Delete(idx)
    }
    // 将key添加到链表尾部
    l.list.Append(key)
    l.keys[key] = l.list.Len() - 1
}

// Evict 执行淘汰并返回被淘汰的key
func (l *LRUPolicy) Evict() string {
    if l.list.Len() == 0 {
        return ""
    }
    // 移除链表头部的key
    key, _ := l.list.Get(0)
    l.list.Delete(0)
    delete(l.keys, key)
    return key
}

// Remove 移除指定key
func (l *LRUPolicy) Remove(key string) {
    if idx, exists := l.keys[key]; exists {
        l.list.Delete(idx)
        delete(l.keys, key)
    }
}

func (l *LRUPolicy) Has(key string) bool {
    _, exists := l.keys[key]
    return exists
}