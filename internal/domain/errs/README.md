# Errs 错误定义包

错误定义包提供了项目中使用的通用错误类型和错误处理工具。遵循Go语言的错误处理最佳实践，定义了领域特定的错误类型，便于错误识别和处理。

## 📁 包结构

```
errs/
└── errs.go    # 通用错误类型定义
```

## 🚀 主要功能

### 错误类型定义

- **领域错误**: 定义业务逻辑相关的错误类型
- **通用错误**: 提供常用的错误变量和函数
- **错误包装**: 支持错误链和上下文信息
- **错误判断**: 提供便捷的错误类型判断方法

## 🔧 安装使用

```go
import "github.com/justinwongcn/hamster/internal/domain/errs"
```

## 📖 快速上手

### 基本错误使用

```go
// 使用预定义错误
if err := someOperation(); err != nil {
    if errors.Is(err, errs.ErrIndexOutOfRange) {
        // 处理索引越界错误
        fmt.Println("索引超出范围")
    }
}

// 创建新的错误
err := errs.NewError("操作失败", "详细错误信息")
if err != nil {
    log.Printf("错误: %v", err)
}
```

### 错误包装和链式处理

```go
// 包装错误
originalErr := errors.New("原始错误")
wrappedErr := errs.WrapError(originalErr, "包装信息")

// 检查错误链
if errors.Is(wrappedErr, originalErr) {
    fmt.Println("包含原始错误")
}
```

## 🎯 设计特点

### 1. 语义化错误

- 使用有意义的错误名称
- 提供详细的错误描述
- 支持错误分类和层次

### 2. 错误链支持

- 支持错误包装和展开
- 保留错误上下文信息
- 便于错误追踪和调试

### 3. 类型安全

- 使用强类型错误定义
- 支持错误类型判断
- 避免字符串比较错误

### 4. 扩展性

- 易于添加新的错误类型
- 支持自定义错误格式
- 兼容标准库错误接口

## ⚠️ 注意事项

### 错误处理原则

1. **及时处理**: 在适当的层级处理错误
2. **信息保留**: 保留足够的错误上下文
3. **类型判断**: 使用errors.Is()而不是字符串比较
4. **错误包装**: 在传递错误时添加上下文信息

### 最佳实践

1. **错误定义**: 在包级别定义错误变量
2. **错误返回**: 总是检查和处理错误
3. **错误日志**: 记录重要的错误信息
4. **错误恢复**: 在可能的情况下提供错误恢复机制

## 🔄 扩展指南

### 添加新的错误类型

```go
// 1. 定义错误变量
var (
    ErrNewErrorType = errors.New("新的错误类型")
    ErrCustomError  = &CustomError{Code: 1001, Message: "自定义错误"}
)

// 2. 定义自定义错误结构
type CustomError struct {
    Code    int
    Message string
    Cause   error
}

func (e *CustomError) Error() string {
    return fmt.Sprintf("错误代码: %d, 消息: %s", e.Code, e.Message)
}

func (e *CustomError) Unwrap() error {
    return e.Cause
}
```

### 错误处理模式

```go
// 1. 简单错误检查
if err != nil {
    return fmt.Errorf("操作失败: %w", err)
}

// 2. 错误类型判断
switch {
case errors.Is(err, errs.ErrIndexOutOfRange):
    // 处理索引错误
case errors.Is(err, errs.ErrInvalidArgument):
    // 处理参数错误
default:
    // 处理其他错误
}

// 3. 错误恢复
if err := riskyOperation(); err != nil {
    log.Printf("操作失败，尝试恢复: %v", err)
    if recoveryErr := recoverOperation(); recoveryErr != nil {
        return fmt.Errorf("恢复失败: %w", recoveryErr)
    }
}
```

## 📊 错误分类

### 系统错误

- 内存不足
- 文件操作失败
- 网络连接错误

### 业务错误

- 参数验证失败
- 业务规则违反
- 状态不一致

### 用户错误

- 输入格式错误
- 权限不足
- 资源不存在

## 🧪 测试策略

### 错误测试

- 测试错误创建和格式化
- 验证错误类型判断
- 检查错误包装和展开

### 运行测试

```bash
# 运行错误包测试
go test ./internal/domain/errs/

# 查看测试覆盖率
go test -cover ./internal/domain/errs/
```
