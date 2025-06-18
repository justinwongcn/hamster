// Copyright 2021 gotomicro
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tools

import "github.com/justinwongcn/hamster/internal/domain/errs"

// List 通用链表接口
// 定义了链表的基本操作方法
type List[T any] interface{}

var _ List[any] = &LinkedList[any]{}

// node 双向循环链表结点
// 包含前驱指针、后继指针和存储的值
type node[T any] struct {
	prev *node[T] // 前驱结点指针
	next *node[T] // 后继结点指针
	val  T        // 结点存储的值
}

// LinkedList 双向循环链表实现
// 包含头结点、尾结点和长度信息
type LinkedList[T any] struct {
	head   *node[T] // 头结点(哨兵)
	tail   *node[T] // 尾结点(哨兵)
	length int      // 链表长度
}

// NewLinkedList 创建一个空的双向循环链表
// 返回值:
//   - *LinkedList[T]: 新建的链表实例
func NewLinkedList[T any]() *LinkedList[T] {
	head := &node[T]{}
	tail := &node[T]{next: head, prev: head}
	head.next, head.prev = tail, tail
	return &LinkedList[T]{
		head: head,
		tail: tail,
	}
}

// NewLinkedListOf 将切片转换为双向循环链表
// 参数:
//   - ts: 要转换的切片
//
// 返回值:
//   - *LinkedList[T]: 包含切片元素的链表实例
func NewLinkedListOf[T any](ts []T) *LinkedList[T] {
	list := NewLinkedList[T]()
	if err := list.Append(ts...); err != nil {
		panic(err)
	}
	return list
}

// findNode 根据索引查找对应的结点
// 参数:
//   - index: 要查找的索引位置
//
// 返回值:
//   - *node[T]: 找到的结点指针
func (l *LinkedList[T]) findNode(index int) *node[T] {
	var cur *node[T]
	if index <= l.Len()/2 {
		cur = l.head
		for i := -1; i < index; i++ {
			cur = cur.next
		}
	} else {
		cur = l.tail
		for i := l.Len(); i > index; i-- {
			cur = cur.prev
		}
	}

	return cur
}

// Get 获取链表中指定位置的元素
// 参数:
//   - index: 要获取的索引位置
//
// 返回值:
//   - T: 找到的元素值
//   - error: 索引越界错误
func (l *LinkedList[T]) Get(index int) (T, error) {
	if !l.checkIndex(index) {
		var zeroValue T
		return zeroValue, errs.NewErrIndexOutOfRange(l.Len(), index)
	}
	n := l.findNode(index)
	return n.val, nil
}

// checkIndex 检查索引是否有效
// 参数:
//   - index: 要检查的索引
//
// 返回值:
//   - bool: true表示索引有效，false表示无效
func (l *LinkedList[T]) checkIndex(index int) bool {
	return 0 <= index && index < l.Len()
}

// Append 往链表最后添加元素
// 参数:
//   - ts: 要添加的元素(可变参数)
//
// 返回值:
//   - error: 操作错误信息
func (l *LinkedList[T]) Append(ts ...T) error {
	for _, t := range ts {
		node := &node[T]{prev: l.tail.prev, next: l.tail, val: t}
		node.prev.next, node.next.prev = node, node
		l.length++
	}
	return nil
}

// Add 在链表指定位置插入元素
// 参数:
//   - index: 要插入的位置索引
//   - t: 要插入的元素
//
// 返回值:
//   - error: 索引越界错误
func (l *LinkedList[T]) Add(index int, t T) error {
	if index < 0 || index > l.length {
		return errs.NewErrIndexOutOfRange(l.length, index)
	}
	if index == l.length {
		return l.Append(t)
	}
	next := l.findNode(index)
	node := &node[T]{prev: next.prev, next: next, val: t}
	node.prev.next, node.next.prev = node, node
	l.length++
	return nil
}

// Set 设置链表中指定位置的元素值
// 参数:
//   - index: 要设置的位置索引
//   - t: 要设置的新值
//
// 返回值:
//   - error: 索引越界错误
func (l *LinkedList[T]) Set(index int, t T) error {
	if !l.checkIndex(index) {
		return errs.NewErrIndexOutOfRange(l.Len(), index)
	}
	node := l.findNode(index)
	node.val = t
	return nil
}

// Delete 删除链表中指定位置的元素
// 参数:
//   - index: 要删除的位置索引
//
// 返回值:
//   - T: 被删除的元素值
//   - error: 索引越界错误
func (l *LinkedList[T]) Delete(index int) (T, error) {
	if !l.checkIndex(index) {
		var zeroValue T
		return zeroValue, errs.NewErrIndexOutOfRange(l.Len(), index)
	}
	node := l.findNode(index)
	node.prev.next = node.next
	node.next.prev = node.prev
	node.prev, node.next = nil, nil
	l.length--
	return node.val, nil
}

// Len 获取链表的长度
// 返回值:
//   - int: 链表当前长度
func (l *LinkedList[T]) Len() int {
	return l.length
}

// Cap 获取链表的容量(与长度相同)
// 返回值:
//   - int: 链表当前长度
func (l *LinkedList[T]) Cap() int {
	return l.Len()
}

// Range 遍历链表中的每个元素
// 参数:
//   - fn: 遍历函数，接收索引和元素值
//
// 返回值:
//   - error: 遍历过程中遇到的错误
func (l *LinkedList[T]) Range(fn func(index int, t T) error) error {
	for cur, i := l.head.next, 0; i < l.length; i++ {
		err := fn(i, cur.val)
		if err != nil {
			return err
		}
		cur = cur.next
	}
	return nil
}

// AsSlice 将链表转换为切片
// 返回值:
//   - []T: 包含链表所有元素的切片
func (l *LinkedList[T]) AsSlice() []T {
	slice := make([]T, l.length)
	for cur, i := l.head.next, 0; i < l.length; i++ {
		slice[i] = cur.val
		cur = cur.next
	}
	return slice
}
