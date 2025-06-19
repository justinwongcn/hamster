package cache

import (
	"github.com/justinwongcn/hamster/internal/domain/tools"
)

// FIFOPolicy 实现FIFO淘汰策略
type FIFOPolicy struct {
	list *tools.LinkedList[string] // 维护key的添加顺序
	keys map[string]int            // 记录key在链表中的位置
}

// NewFIFOPolicy 创建新的FIFO策略实例
func NewFIFOPolicy() *FIFOPolicy {
	return &FIFOPolicy{
		list: tools.NewLinkedList[string](),
		keys: make(map[string]int),
	}
}

// KeyAccessed 记录key被访问
func (f *FIFOPolicy) KeyAccessed(key string) {
	// 如果key已存在，不更新位置（FIFO特性）
	if _, exists := f.keys[key]; exists {
		return
	}
	// 新key添加到链表尾部
	f.list.Append(key)
	f.keys[key] = f.list.Len() - 1
}

// Evict 执行淘汰并返回被淘汰的key
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

// Remove 移除指定key
func (f *FIFOPolicy) Remove(key string) {
	if idx, exists := f.keys[key]; exists {
		f.list.Delete(idx)
		delete(f.keys, key)
	}
}

// Has 检查key是否存在
func (f *FIFOPolicy) Has(key string) bool {
	_, exists := f.keys[key]
	return exists
}