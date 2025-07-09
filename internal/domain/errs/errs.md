# errs.go - 通用错误类型定义

## 文件概述

`errs.go` 定义了项目中使用的通用错误类型和错误创建函数。该文件遵循Go语言的错误处理最佳实践，提供了语义化的错误创建方法，便于在整个项目中进行统一的错误处理。

## 核心功能

### 1. 错误创建函数

#### NewErrIndexOutOfRange - 索引越界错误

```go
func NewErrIndexOutOfRange(length int, index int) error
```

创建一个表示下标超出范围的错误。

**参数：**

- `length`: 集合的长度
- `index`: 访问的索引

**返回值：**

- `error`: 格式化的索引越界错误

**实现逻辑：**

1. 使用fmt.Errorf创建格式化错误
2. 包含长度和索引信息
3. 提供中文错误描述

**示例：**

```go
// 创建索引越界错误
err := errs.NewErrIndexOutOfRange(5, 10)
fmt.Println(err) // 输出: ekit: 下标超出范围，长度 5, 下标 10

// 在函数中使用
func GetElement(slice []int, index int) (int, error) {
    if index < 0 || index >= len(slice) {
        return 0, errs.NewErrIndexOutOfRange(len(slice), index)
    }
    return slice[index], nil
}
```

## 设计特点

### 1. 语义化命名

- 函数名清晰表达错误类型
- 使用New前缀表示构造函数
- 错误类型名称具有描述性

### 2. 信息完整性

- 包含导致错误的具体参数
- 提供足够的上下文信息
- 便于调试和问题定位

### 3. 国际化支持

- 使用中文错误消息
- 便于本地化用户理解
- 保持错误信息的一致性

### 4. 标准兼容

- 返回标准error接口
- 兼容Go语言错误处理模式
- 支持错误包装和链式处理

## 使用示例

### 1. 基本使用

```go
package main

import (
    "fmt"
    "github.com/justinwongcn/hamster/internal/domain/errs"
)

func main() {
    // 模拟数组访问
    arr := []int{1, 2, 3, 4, 5}
    index := 10
    
    if index >= len(arr) {
        err := errs.NewErrIndexOutOfRange(len(arr), index)
        fmt.Printf("错误: %v\n", err)
        // 输出: 错误: ekit: 下标超出范围，长度 5, 下标 10
    }
}
```

### 2. 在数据结构中使用

```go
type SafeSlice struct {
    data []interface{}
}

func (s *SafeSlice) Get(index int) (interface{}, error) {
    if index < 0 || index >= len(s.data) {
        return nil, errs.NewErrIndexOutOfRange(len(s.data), index)
    }
    return s.data[index], nil
}

func (s *SafeSlice) Set(index int, value interface{}) error {
    if index < 0 || index >= len(s.data) {
        return errs.NewErrIndexOutOfRange(len(s.data), index)
    }
    s.data[index] = value
    return nil
}

// 使用示例
func main() {
    slice := &SafeSlice{data: []interface{}{1, 2, 3}}
    
    // 正常访问
    val, err := slice.Get(1)
    if err == nil {
        fmt.Printf("值: %v\n", val) // 输出: 值: 2
    }
    
    // 越界访问
    _, err = slice.Get(5)
    if err != nil {
        fmt.Printf("错误: %v\n", err)
        // 输出: 错误: ekit: 下标超出范围，长度 3, 下标 5
    }
}
```

### 3. 错误处理和包装

```go
func ProcessArray(arr []int, indices []int) error {
    for _, index := range indices {
        if index < 0 || index >= len(arr) {
            // 包装错误，添加更多上下文
            return fmt.Errorf("处理数组时发生错误: %w", 
                errs.NewErrIndexOutOfRange(len(arr), index))
        }
        
        // 处理元素
        fmt.Printf("处理元素: %d\n", arr[index])
    }
    return nil
}

// 使用示例
func main() {
    arr := []int{10, 20, 30}
    indices := []int{0, 1, 5} // 索引5超出范围
    
    err := ProcessArray(arr, indices)
    if err != nil {
        fmt.Printf("处理失败: %v\n", err)
        // 输出: 处理失败: 处理数组时发生错误: ekit: 下标超出范围，长度 3, 下标 5
    }
}
```

### 4. 错误类型判断

```go
func HandleError(err error) {
    // 检查是否包含索引越界错误
    if strings.Contains(err.Error(), "下标超出范围") {
        fmt.Println("检测到索引越界错误，执行特殊处理")
        // 执行特殊的错误恢复逻辑
        return
    }
    
    // 处理其他类型的错误
    fmt.Printf("其他错误: %v\n", err)
}

// 更好的做法是定义错误变量进行比较
var ErrIndexOutOfRange = errors.New("索引越界")

func NewErrIndexOutOfRangeV2(length int, index int) error {
    return fmt.Errorf("%w: 长度 %d, 索引 %d", ErrIndexOutOfRange, length, index)
}

func HandleErrorV2(err error) {
    if errors.Is(err, ErrIndexOutOfRange) {
        fmt.Println("检测到索引越界错误")
        return
    }
    
    fmt.Printf("其他错误: %v\n", err)
}
```

## 扩展建议

### 1. 添加更多错误类型

```go
// 建议添加的错误类型
func NewErrInvalidArgument(argName string, value interface{}) error {
    return fmt.Errorf("ekit: 无效参数 %s，值: %v", argName, value)
}

func NewErrNilPointer(varName string) error {
    return fmt.Errorf("ekit: 空指针异常，变量: %s", varName)
}

func NewErrCapacityExceeded(current, max int) error {
    return fmt.Errorf("ekit: 容量超限，当前: %d，最大: %d", current, max)
}

func NewErrKeyNotFound(key string) error {
    return fmt.Errorf("ekit: 键不存在: %s", key)
}
```

### 2. 定义错误变量

```go
// 定义可比较的错误变量
var (
    ErrIndexOutOfRange   = errors.New("索引越界")
    ErrInvalidArgument   = errors.New("无效参数")
    ErrNilPointer        = errors.New("空指针")
    ErrCapacityExceeded  = errors.New("容量超限")
    ErrKeyNotFound       = errors.New("键不存在")
)

// 使用错误包装
func NewErrIndexOutOfRangeWrapped(length int, index int) error {
    return fmt.Errorf("%w: 长度 %d, 索引 %d", ErrIndexOutOfRange, length, index)
}
```

### 3. 创建错误结构体

```go
// 定义结构化错误
type IndexError struct {
    Length int
    Index  int
}

func (e *IndexError) Error() string {
    return fmt.Sprintf("ekit: 下标超出范围，长度 %d, 下标 %d", e.Length, e.Index)
}

func (e *IndexError) Is(target error) bool {
    _, ok := target.(*IndexError)
    return ok
}

// 创建函数
func NewIndexError(length, index int) *IndexError {
    return &IndexError{
        Length: length,
        Index:  index,
    }
}
```

## 注意事项

### 1. 错误信息格式

```go
// ✅ 推荐：提供详细的错误信息
err := errs.NewErrIndexOutOfRange(len(slice), index)

// ❌ 避免：错误信息过于简单
err := errors.New("索引错误")
```

### 2. 错误处理

```go
// ✅ 推荐：检查和处理错误
val, err := slice.Get(index)
if err != nil {
    log.Printf("获取元素失败: %v", err)
    return err
}

// ❌ 避免：忽略错误
val, _ := slice.Get(index) // 可能导致程序异常
```

### 3. 错误传播

```go
// ✅ 推荐：包装错误，添加上下文
if err != nil {
    return fmt.Errorf("处理数据时发生错误: %w", err)
}

// ❌ 避免：直接返回原始错误，丢失上下文
if err != nil {
    return err
}
```

### 4. 性能考虑

```go
// ✅ 推荐：在错误路径中创建错误
if index >= len(slice) {
    return errs.NewErrIndexOutOfRange(len(slice), index)
}

// ❌ 避免：预先创建错误对象
var indexErr = errs.NewErrIndexOutOfRange(0, 0) // 不必要的预创建
```

## 最佳实践

### 1. 错误命名规范

- 使用New前缀表示构造函数
- 错误类型名称要具有描述性
- 保持命名的一致性

### 2. 错误信息内容

- 包含足够的调试信息
- 使用用户友好的语言
- 避免暴露内部实现细节

### 3. 错误处理策略

- 在适当的层级处理错误
- 保留错误的上下文信息
- 提供错误恢复机制

### 4. 测试覆盖

- 测试错误创建函数
- 验证错误信息格式
- 检查错误处理逻辑
