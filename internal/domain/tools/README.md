# Tools 工具包

工具包提供了项目中使用的通用数据结构和算法实现，包括双向循环链表和LRU缓存算法。这些工具类为上层业务逻辑提供高效的数据结构支持。

## 📁 包结构

```
tools/
├── linked_list.go      # 双向循环链表实现
├── linked_list_test.go # 双向循环链表测试
├── lru.go             # LRU算法实现
└── lru_test.go        # LRU算法测试
```

## 🚀 主要功能

### 双向循环链表 (LinkedList)

- **泛型支持**: 支持存储任意类型的数据
- **双向遍历**: 支持前向和后向遍历
- **循环结构**: 简化边界条件处理
- **高效操作**: O(1)时间复杂度的插入和删除

### LRU缓存算法 (LRU)

- **最近最少使用**: 实现LRU淘汰策略
- **泛型支持**: 支持任意类型的键值对
- **高效访问**: O(1)时间复杂度的查找和更新
- **容量控制**: 支持固定容量限制

## 🔧 安装使用

```go
import "github.com/justinwongcn/hamster/internal/domain/tools"
```

## 📖 快速上手

### 双向循环链表使用

```go
// 创建链表
list := tools.NewLinkedList[int]()

// 添加元素
list.Append(1, 2, 3)
list.Add(1, 4) // 在索引1插入4

// 获取元素
val, err := list.Get(2)
if err != nil {
    log.Printf("获取失败: %v", err)
}

// 删除元素
err = list.Delete(0)
if err != nil {
    log.Printf("删除失败: %v", err)
}

// 遍历链表
list.Range(func(index int, val int) error {
    fmt.Printf("索引%d: 值%d\n", index, val)
    return nil
})
```

### LRU缓存使用

```go
// 创建LRU缓存
lru := tools.NewLRU[string](100) // 容量100

// 添加元素
lru.Put("key1", "value1")
lru.Put("key2", "value2")

// 获取元素
value, ok := lru.Get("key1")
if ok {
    fmt.Printf("获取到值: %s\n", value)
}

// 检查是否存在
if lru.Exist("key2") {
    fmt.Println("key2存在")
}

// 删除元素
lru.Delete("key1")
```

## 🎯 设计特点

### 1. 泛型设计

- 使用Go 1.18+的泛型特性
- 类型安全，避免类型断言
- 支持任意类型的数据存储

### 2. 高性能

- 双向链表提供O(1)插入删除
- LRU算法提供O(1)访问更新
- 内存友好的数据结构设计

### 3. 易用性

- 简洁的API设计
- 丰富的操作方法
- 完善的错误处理

### 4. 可靠性

- 完整的单元测试覆盖
- 边界条件处理
- 线程安全考虑

## ⚠️ 注意事项

### 双向循环链表

1. **索引范围**: 索引从0开始，注意边界检查
2. **内存管理**: 删除元素时会自动清理内存
3. **并发安全**: 非线程安全，需要外部同步
4. **性能考虑**: 随机访问为O(n)，顺序访问为O(1)

### LRU缓存

1. **容量限制**: 超过容量时会自动淘汰最久未使用的元素
2. **访问更新**: Get操作会更新元素的使用时间
3. **并发安全**: 非线程安全，需要外部同步
4. **内存占用**: 需要额外的链表结构维护访问顺序

## 🔄 扩展指南

### 添加新的数据结构

```go
// 1. 定义接口
type Stack[T any] interface {
    Push(item T)
    Pop() (T, error)
    Peek() (T, error)
    IsEmpty() bool
    Size() int
}

// 2. 实现结构体
type ArrayStack[T any] struct {
    items []T
}

// 3. 实现方法
func (s *ArrayStack[T]) Push(item T) {
    s.items = append(s.items, item)
}

func (s *ArrayStack[T]) Pop() (T, error) {
    if len(s.items) == 0 {
        var zero T
        return zero, errors.New("栈为空")
    }
    
    item := s.items[len(s.items)-1]
    s.items = s.items[:len(s.items)-1]
    return item, nil
}
```

### 性能优化建议

```go
// 1. 预分配容量
list := NewLinkedList[int]()
// 如果知道大概大小，可以考虑预分配

// 2. 批量操作
items := []int{1, 2, 3, 4, 5}
list.Append(items...) // 比逐个添加更高效

// 3. 避免频繁的随机访问
// ❌ 避免
for i := 0; i < list.Len(); i++ {
    val, _ := list.Get(i) // O(n²)复杂度
}

// ✅ 推荐
list.Range(func(index int, val int) error {
    // 处理元素
    return nil
}) // O(n)复杂度
```

## 📊 性能基准

### 双向循环链表性能

- **插入操作**: O(1) - 头部/尾部插入
- **删除操作**: O(1) - 已知节点删除
- **查找操作**: O(n) - 按索引查找
- **遍历操作**: O(n) - 顺序遍历

### LRU缓存性能

- **Get操作**: O(1) - 哈希表查找
- **Put操作**: O(1) - 插入和更新
- **Delete操作**: O(1) - 删除指定键
- **空间复杂度**: O(n) - n为容量大小

## 🧪 测试覆盖

### 测试策略

- **单元测试**: 覆盖所有公开方法
- **边界测试**: 空集合、满容量等边界情况
- **错误测试**: 无效索引、空栈等错误情况
- **性能测试**: 基准测试验证时间复杂度

### 运行测试

```bash
# 运行所有测试
go test ./internal/domain/tools/

# 运行基准测试
go test -bench=. ./internal/domain/tools/

# 查看测试覆盖率
go test -cover ./internal/domain/tools/
```
