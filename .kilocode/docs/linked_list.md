# 双向循环链表

## 概述
双向循环链表是一种链式存储结构，每个节点包含指向前驱和后继节点的指针，形成一个循环。支持高效的插入、删除和遍历操作。

## 核心功能
- 双向指针实现高效的前后遍历
- 循环结构简化边界条件处理
- 支持泛型存储任意类型数据
- 提供丰富的操作方法

## 主要类型
### LinkedList[T]
双向循环链表的核心结构体

```go
type LinkedList[T any] struct {
    head   *node[T] // 头结点(哨兵)
    tail   *node[T] // 尾结点(哨兵)
    length int      // 链表长度
}
```

### node[T]
链表节点结构

```go
type node[T any] struct {
    prev *node[T] // 前驱结点指针
    next *node[T] // 后继结点指针
    val  T        // 结点存储的值
}
```

## 主要方法
### NewLinkedList[T]()
创建一个空的双向循环链表

```go
func NewLinkedList[T any]() *LinkedList[T]
```

### NewLinkedListOf[T](ts []T)
将切片转换为双向循环链表

```go
func NewLinkedListOf[T any](ts []T) *LinkedList[T]
```

### Append(ts ...T)
往链表最后添加元素

```go
func (l *LinkedList[T]) Append(ts ...T) error
```

### Add(index int, t T)
在指定位置插入元素

```go
func (l *LinkedList[T]) Add(index int, t T) error
```

### Get(index int)
获取指定位置的元素

```go
func (l *LinkedList[T]) Get(index int) (T, error)
```

### Set(index int, t T)
设置指定位置的元素值

```go
func (l *LinkedList[T]) Set(index int, t T) error
```

### Delete(index int)
删除指定位置的元素

```go
func (l *LinkedList[T]) Delete(index int) (T, error)
```

### Len()
获取链表长度

```go
func (l *LinkedList[T]) Len() int
```

### Range(fn func(index int, t T) error)
遍历链表中的每个元素

```go
func (l *LinkedList[T]) Range(fn func(index int, t T) error) error
```

### AsSlice()
将链表转换为切片

```go
func (l *LinkedList[T]) AsSlice() []T
```

## 使用示例
```go
list := NewLinkedList[int]()
list.Append(1, 2, 3)
list.Add(1, 4) // 在索引1插入4
val, _ := list.Get(2) // 获取索引2的值
list.Delete(0) // 删除索引0的元素
```

## 注意事项
1. 索引从0开始
2. 插入/删除操作会自动维护链表结构
3. 支持负向遍历
4. 注意处理边界条件