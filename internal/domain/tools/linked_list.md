# linked_list.go - 双向循环链表实现

## 文件概述

`linked_list.go` 实现了一个泛型的双向循环链表数据结构。该实现使用哨兵节点简化边界条件处理，支持高效的插入、删除和遍历操作。作为基础数据结构，它为LRU缓存等上层算法提供支持。

## 核心功能

### 1. 类型定义

#### List 接口

```go
type List[T any] interface{}
```

通用链表接口，为将来扩展其他链表实现预留接口。

#### node 节点结构

```go
type node[T any] struct {
    prev *node[T] // 前驱结点指针
    next *node[T] // 后继结点指针
    val  T        // 结点存储的值
}
```

**设计特点：**

- 双向指针支持前后遍历
- 泛型支持任意类型数据
- 简洁的结构设计

#### LinkedList 链表结构

```go
type LinkedList[T any] struct {
    head   *node[T] // 头结点(哨兵)
    tail   *node[T] // 尾结点(哨兵)
    length int      // 链表长度
}
```

**设计特点：**

- 使用哨兵节点简化边界处理
- 维护长度信息提高查询效率
- 循环结构简化操作逻辑

### 2. 构造函数

#### NewLinkedList

```go
func NewLinkedList[T any]() *LinkedList[T]
```

创建一个空的双向循环链表，初始化哨兵节点。

**实现逻辑：**

1. 创建头尾哨兵节点
2. 建立循环连接关系
3. 初始化长度为0

**示例：**

```go
list := NewLinkedList[int]()
fmt.Printf("初始长度: %d\n", list.Len()) // 输出: 0
```

#### NewLinkedListOf

```go
func NewLinkedListOf[T any](ts []T) *LinkedList[T]
```

从切片创建链表，批量初始化数据。

**示例：**

```go
data := []int{1, 2, 3, 4, 5}
list := NewLinkedListOf(data)
fmt.Printf("链表长度: %d\n", list.Len()) // 输出: 5
```

### 3. 基本操作

#### Get - 获取元素

```go
func (l *LinkedList[T]) Get(index int) (T, error)
```

根据索引获取元素，支持负索引（从尾部开始）。

**实现逻辑：**

1. 检查索引有效性
2. 选择最优遍历方向（前向或后向）
3. 遍历到目标位置
4. 返回元素值

**示例：**

```go
list := NewLinkedListOf([]int{1, 2, 3})
val, err := list.Get(1)    // 获取索引1的元素: 2
val, err = list.Get(-1)    // 获取最后一个元素: 3
```

#### Set - 设置元素

```go
func (l *LinkedList[T]) Set(index int, t T) error
```

设置指定索引位置的元素值。

**示例：**

```go
list := NewLinkedListOf([]int{1, 2, 3})
err := list.Set(1, 10)     // 将索引1的元素设为10
// 链表变为: [1, 10, 3]
```

### 4. 插入操作

#### Append - 尾部追加

```go
func (l *LinkedList[T]) Append(ts ...T) error
```

在链表尾部追加一个或多个元素。

**实现逻辑：**

1. 遍历所有要添加的元素
2. 在尾部哨兵前插入新节点
3. 更新链表长度

**示例：**

```go
list := NewLinkedList[int]()
list.Append(1, 2, 3)       // 添加多个元素
list.Append(4)             // 添加单个元素
// 链表: [1, 2, 3, 4]
```

#### Add - 指定位置插入

```go
func (l *LinkedList[T]) Add(index int, t T) error
```

在指定索引位置插入元素。

**实现逻辑：**

1. 检查索引有效性（允许等于长度）
2. 找到插入位置的前一个节点
3. 创建新节点并插入
4. 更新链表长度

**示例：**

```go
list := NewLinkedListOf([]int{1, 3, 5})
list.Add(1, 2)             // 在索引1插入2
list.Add(3, 4)             // 在索引3插入4
// 链表: [1, 2, 3, 4, 5]
```

### 5. 删除操作

#### Delete - 删除指定索引

```go
func (l *LinkedList[T]) Delete(index int) error
```

删除指定索引位置的元素。

**实现逻辑：**

1. 检查索引有效性
2. 找到要删除的节点
3. 更新前后节点的连接
4. 更新链表长度

**示例：**

```go
list := NewLinkedListOf([]int{1, 2, 3, 4, 5})
list.Delete(2)             // 删除索引2的元素(3)
// 链表: [1, 2, 4, 5]
```

### 6. 查询操作

#### Len - 获取长度

```go
func (l *LinkedList[T]) Len() int
```

返回链表当前长度。

#### Cap - 获取容量

```go
func (l *LinkedList[T]) Cap() int
```

返回链表容量（与长度相同）。

### 7. 遍历操作

#### Range - 遍历元素

```go
func (l *LinkedList[T]) Range(fn func(index int, t T) error) error
```

遍历链表中的每个元素，支持早期退出。

**示例：**

```go
list := NewLinkedListOf([]int{1, 2, 3, 4, 5})

// 打印所有元素
list.Range(func(index int, val int) error {
    fmt.Printf("索引%d: 值%d\n", index, val)
    return nil
})

// 查找特定元素
list.Range(func(index int, val int) error {
    if val == 3 {
        fmt.Printf("找到3在索引%d\n", index)
        return errors.New("找到了") // 提前退出
    }
    return nil
})
```

#### AsSlice - 转换为切片

```go
func (l *LinkedList[T]) AsSlice() []T
```

将链表转换为切片，返回新的切片副本。

**示例：**

```go
list := NewLinkedListOf([]int{1, 2, 3})
slice := list.AsSlice()
fmt.Printf("切片: %v\n", slice) // 输出: [1, 2, 3]
```

## 性能特性

### 时间复杂度

- **插入操作**: O(1) - 头部/尾部插入，O(n) - 中间插入
- **删除操作**: O(1) - 头部/尾部删除，O(n) - 中间删除
- **查找操作**: O(n) - 按索引查找
- **遍历操作**: O(n) - 顺序遍历

### 空间复杂度

- **存储空间**: O(n) - n为元素数量
- **额外空间**: O(1) - 固定的哨兵节点和元数据

### 优化特性

1. **双向遍历**: 根据索引位置选择最优遍历方向
2. **哨兵节点**: 简化边界条件，减少特殊情况处理
3. **循环结构**: 统一插入删除逻辑

## 使用示例

### 1. 基本操作示例

```go
// 创建链表
list := NewLinkedList[string]()

// 添加元素
list.Append("hello", "world")
list.Add(1, "beautiful")
// 链表: ["hello", "beautiful", "world"]

// 获取元素
val, err := list.Get(1)
if err == nil {
    fmt.Printf("索引1的值: %s\n", val) // 输出: beautiful
}

// 修改元素
list.Set(0, "hi")
// 链表: ["hi", "beautiful", "world"]

// 删除元素
list.Delete(1)
// 链表: ["hi", "world"]
```

### 2. 遍历操作示例

```go
list := NewLinkedListOf([]int{10, 20, 30, 40, 50})

// 正向遍历
fmt.Println("正向遍历:")
list.Range(func(index int, val int) error {
    fmt.Printf("  [%d] = %d\n", index, val)
    return nil
})

// 查找最大值
maxVal := 0
list.Range(func(index int, val int) error {
    if val > maxVal {
        maxVal = val
    }
    return nil
})
fmt.Printf("最大值: %d\n", maxVal)

// 转换为切片进行其他操作
slice := list.AsSlice()
sort.Ints(slice)
fmt.Printf("排序后: %v\n", slice)
```

### 3. 错误处理示例

```go
list := NewLinkedListOf([]int{1, 2, 3})

// 处理索引越界
val, err := list.Get(10)
if err != nil {
    if errors.Is(err, errs.ErrIndexOutOfRange) {
        fmt.Println("索引超出范围")
    }
}

// 处理删除错误
err = list.Delete(-10)
if err != nil {
    fmt.Printf("删除失败: %v\n", err)
}
```

## 注意事项

### 1. 索引处理

```go
// ✅ 正确：检查返回的错误
val, err := list.Get(index)
if err != nil {
    // 处理错误
    return err
}

// ❌ 错误：忽略错误可能导致程序异常
val, _ := list.Get(index) // 可能获取到零值
```

### 2. 并发安全

```go
// ❌ 错误：链表不是线程安全的
go func() {
    list.Append(1)
}()
go func() {
    list.Delete(0) // 可能导致数据竞争
}()

// ✅ 正确：使用互斥锁保护
var mu sync.Mutex
go func() {
    mu.Lock()
    list.Append(1)
    mu.Unlock()
}()
```

### 3. 内存管理

```go
// ✅ 正确：及时清理不需要的链表
list := NewLinkedList[*LargeObject]()
// 使用完毕后，链表会自动清理
list = nil // 帮助GC回收

// ⚠️ 注意：避免循环引用
type Node struct {
    list *LinkedList[*Node]
    // 其他字段
}
```

### 4. 性能考虑

```go
// ❌ 避免：频繁的随机访问
for i := 0; i < list.Len(); i++ {
    val, _ := list.Get(i) // O(n²)复杂度
    process(val)
}

// ✅ 推荐：使用Range遍历
list.Range(func(index int, val T) error {
    process(val) // O(n)复杂度
    return nil
})

// ✅ 推荐：批量操作
items := []int{1, 2, 3, 4, 5}
list.Append(items...) // 比逐个添加更高效
```
