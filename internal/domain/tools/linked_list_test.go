package tools

import (
	"errors"
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestLinkedList_Add 测试链表Add方法
// 验证以下场景:
// 1. 在链表头部添加元素
// 2. 在链表中间添加元素
// 3. 在链表尾部添加元素
// 4. 索引越界情况
func TestLinkedList_Add(t *testing.T) {
	testCases := []struct {
		name           string
		list           func() *LinkedList[int] // 使用函数返回新实例
		index          int
		newVal         int
		wantLinkedList *LinkedList[int]
		wantErr        error
	}{
		{
			name:           "add num to index left",
			list:           func() *LinkedList[int] { return NewLinkedListOf([]int{1, 2, 3}) },
			newVal:         100,
			index:          0,
			wantLinkedList: NewLinkedListOf([]int{100, 1, 2, 3}),
		},
		{
			name:           "add num to index left1",
			list:           func() *LinkedList[int] { return NewLinkedListOf([]int{1, 2, 3, 44, 55, 66, 77}) },
			newVal:         100,
			index:          3,
			wantLinkedList: NewLinkedListOf([]int{1, 2, 3, 100, 44, 55, 66, 77}),
		},
		{
			name:           "add num to index right",
			list:           func() *LinkedList[int] { return NewLinkedListOf([]int{1, 2, 3}) },
			newVal:         100,
			index:          3,
			wantLinkedList: NewLinkedListOf([]int{1, 2, 3, 100}),
		},
		{
			name:           "add num to index right1",
			list:           func() *LinkedList[int] { return NewLinkedListOf([]int{1, 2, 3, 44, 55, 66, 77}) },
			newVal:         100,
			index:          5,
			wantLinkedList: NewLinkedListOf([]int{1, 2, 3, 44, 55, 100, 66, 77}),
		},
		{
			name:           "add num to index mid",
			list:           func() *LinkedList[int] { return NewLinkedListOf([]int{1, 2, 3}) },
			newVal:         100,
			index:          1,
			wantLinkedList: NewLinkedListOf([]int{1, 100, 2, 3}),
		},
		{
			name:    "add num to index -1",
			list:    func() *LinkedList[int] { return NewLinkedListOf([]int{1, 2, 3}) },
			newVal:  100,
			index:   -1,
			wantErr: fmt.Errorf("ekit: 下标超出范围，长度 %d, 下标 %d", 3, -1),
		},
		{
			name:    "add num to index OutOfRange",
			list:    func() *LinkedList[int] { return NewLinkedListOf([]int{1, 2, 3}) },
			newVal:  100,
			index:   4,
			wantErr: fmt.Errorf("ekit: 下标超出范围，长度 %d, 下标 %d", 3, 4),
		},
		{
			name:           "add num to index 0",
			list:           func() *LinkedList[int] { return NewLinkedListOf([]int{}) },
			newVal:         100,
			index:          0,
			wantErr:        nil,
			wantLinkedList: NewLinkedListOf([]int{100}),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			list := tc.list() // 获取新的链表实例
			err := list.Add(tc.index, tc.newVal)
			assert.Equal(t, tc.wantErr, err)
			if err != nil {
				return
			}
			assert.Equal(t, tc.wantLinkedList.AsSlice(), list.AsSlice())
		})
	}
}

// TestLinkedList_Delete 测试链表Delete方法
// 验证以下场景:
// 1. 删除链表头部元素
// 2. 删除链表中间元素
// 3. 删除链表尾部元素
// 4. 删除不存在的元素
func TestLinkedList_Delete(t *testing.T) {
	testCases := []struct {
		name           string
		list           func() *LinkedList[int] // 使用函数返回新实例
		wantLinkedList *LinkedList[int]
		delVal         int
		index          int
		wantErr        error
	}{
		{
			name:    "delete num to index -1",
			list:    func() *LinkedList[int] { return NewLinkedListOf([]int{1, 2, 3}) },
			index:   -1,
			wantErr: fmt.Errorf("ekit: 下标超出范围，长度 %d, 下标 %d", 3, -1),
		},
		{
			name:    "delete beyond length index 99",
			list:    func() *LinkedList[int] { return NewLinkedListOf([]int{1, 2, 3}) },
			index:   99,
			wantErr: fmt.Errorf("ekit: 下标超出范围，长度 %d, 下标 %d", 3, 99),
		},
		{
			name:    "delete beyond length index 3",
			list:    func() *LinkedList[int] { return NewLinkedListOf([]int{1, 2, 3}) },
			index:   3,
			wantErr: fmt.Errorf("ekit: 下标超出范围，长度 %d, 下标 %d", 3, 3),
		},
		{
			name:    "delete empty node",
			list:    func() *LinkedList[int] { return NewLinkedListOf([]int{}) },
			index:   3,
			wantErr: fmt.Errorf("ekit: 下标超出范围，长度 %d, 下标 %d", 0, 3),
		},
		{
			name:           "delete num to index 0",
			list:           func() *LinkedList[int] { return NewLinkedListOf([]int{1, 2, 3}) },
			index:          0,
			delVal:         1,
			wantLinkedList: NewLinkedListOf([]int{2, 3}),
		},
		{
			name:           "delete num to index by tail",
			list:           func() *LinkedList[int] { return NewLinkedListOf([]int{1, 2, 3, 4, 5}) },
			index:          4,
			delVal:         5,
			wantLinkedList: NewLinkedListOf([]int{1, 2, 3, 4}),
		},
		{
			name:           "delete num to index 1",
			list:           func() *LinkedList[int] { return NewLinkedListOf([]int{11, 22, 33, 44, 55}) },
			index:          1,
			delVal:         22,
			wantLinkedList: NewLinkedListOf([]int{11, 33, 44, 55}),
		},
		{
			name:           "deleting an element with only one",
			list:           func() *LinkedList[int] { return NewLinkedListOf([]int{888}) },
			index:          0,
			delVal:         888,
			wantLinkedList: NewLinkedListOf([]int{}),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			list := tc.list() // 获取新的链表实例
			delVal, err := list.Delete(tc.index)
			if err != nil {
				assert.Equal(t, tc.wantErr, err)
			} else {
				assert.Equal(t, tc.delVal, delVal)
				assert.Equal(t, tc.wantLinkedList.AsSlice(), list.AsSlice())
			}
		})
	}
}

func TestLinkedList_Append(t *testing.T) {
	testCases := []struct {
		name           string
		list           func() *LinkedList[int] // 使用函数返回新实例
		index          int
		newVal         []int
		wantLinkedList *LinkedList[int]
		wantErr        error
	}{
		{
			name:           "append non-empty values to non-empty list",
			list:           func() *LinkedList[int] { return NewLinkedListOf([]int{123}) },
			newVal:         []int{234, 456},
			wantLinkedList: NewLinkedListOf([]int{123, 234, 456}),
		},
		{
			name:           "append empty values to non-empty list",
			list:           func() *LinkedList[int] { return NewLinkedListOf([]int{123}) },
			newVal:         []int{},
			wantLinkedList: NewLinkedListOf([]int{123}),
		},
		{
			name:           "append nil to non-empty list",
			list:           func() *LinkedList[int] { return NewLinkedListOf([]int{123}) },
			newVal:         nil,
			wantLinkedList: NewLinkedListOf([]int{123}),
		},
		{
			name:           "append non-empty values to empty list",
			list:           func() *LinkedList[int] { return NewLinkedListOf([]int{}) },
			newVal:         []int{234, 456},
			wantLinkedList: NewLinkedListOf([]int{234, 456}),
		},
		{
			name:           "append empty values to empty list",
			list:           func() *LinkedList[int] { return NewLinkedListOf([]int{}) },
			newVal:         []int{},
			wantLinkedList: NewLinkedListOf([]int{}),
		},
		{
			name:           "append nil to empty list",
			list:           func() *LinkedList[int] { return NewLinkedListOf([]int{}) },
			newVal:         nil,
			wantLinkedList: NewLinkedListOf([]int{}),
		},
		{
			name:           "append non-empty values to nil list",
			list:           func() *LinkedList[int] { return NewLinkedListOf[int](nil) },
			newVal:         []int{234, 456},
			wantLinkedList: NewLinkedListOf([]int{234, 456}),
		},
		{
			name:           "append empty values to nil list",
			list:           func() *LinkedList[int] { return NewLinkedListOf[int](nil) },
			newVal:         []int{},
			wantLinkedList: NewLinkedListOf([]int{}),
		},
		{
			name:           "append nil to nil list",
			list:           func() *LinkedList[int] { return NewLinkedListOf[int](nil) },
			newVal:         nil,
			wantLinkedList: NewLinkedListOf([]int{}),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			list := tc.list() // 获取新的链表实例
			err := list.Append(tc.newVal...)
			assert.Equal(t, tc.wantErr, err)
			assert.Equal(t, tc.wantLinkedList.AsSlice(), list.AsSlice())
		})
	}
}

func TestNewLinkedListOf(t *testing.T) {
	testCases := []struct {
		name        string
		slice       func() []int
		wantedSlice []int
	}{
		{
			name:        "nil",
			slice:       func() []int { return nil },
			wantedSlice: []int{},
		},
		{
			name:        "vacant",
			slice:       func() []int { return []int{} },
			wantedSlice: []int{},
		},
		{
			name:        "single",
			slice:       func() []int { return []int{1} },
			wantedSlice: []int{1},
		},
		{
			name:        "normal",
			slice:       func() []int { return []int{1, 2, 3} },
			wantedSlice: []int{1, 2, 3},
		},
	}

	for _, tc := range testCases {
		slice := tc.slice() // 获取新的切片实例
		list := NewLinkedListOf(slice)
		// 在这里断言你的元素，可以利用 Get 方法，也可以直接用 AsSlice 来断言
		assert.Equal(t, tc.wantedSlice, list.AsSlice())
	}
}

// TestLinkedList_AsSlice 测试链表AsSlice方法
// 验证以下场景:
// 1. 空链表转换为切片
// 2. 非空链表转换为切片
// 3. 验证切片元素顺序与链表一致
func TestLinkedList_AsSlice(t *testing.T) {
	vals := []int{1, 2, 3}
	a := NewLinkedListOf(vals)
	slice := a.AsSlice()
	// 内容相同
	assert.Equal(t, slice, vals)
	aAddr := fmt.Sprintf("%p", vals)
	sliceAddr := fmt.Sprintf("%p", slice)
	// 但是地址不同，也就是意味着 slice 必须是一个新创建的
	assert.NotEqual(t, aAddr, sliceAddr)
}

func TestLinkedList_Cap(t *testing.T) {
	t.Run("empty list", func(t *testing.T) {
		list := NewLinkedList[int]()
		assert.Equal(t, 0, list.Cap())
	})

	t.Run("after append", func(t *testing.T) {
		list := NewLinkedList[int]()
		err := list.Append(12)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, 1, list.Cap())
	})
}

func TestLinkedList_Get(t *testing.T) {
	tests := []struct {
		name    string
		list    func() *LinkedList[int]
		index   int
		wantVal int
		wantErr error
	}{
		{
			name:    "get left",
			list:    func() *LinkedList[int] { return NewLinkedListOf([]int{1, 2, 3, 4, 5}) },
			index:   0,
			wantVal: 1,
		},
		{
			name:    "get right",
			list:    func() *LinkedList[int] { return NewLinkedListOf([]int{1, 2, 3, 4, 5}) },
			index:   4,
			wantVal: 5,
		},
		{
			name:    "get middle",
			list:    func() *LinkedList[int] { return NewLinkedListOf([]int{1, 2, 3, 4, 5}) },
			index:   2,
			wantVal: 3,
		},
		{
			name:    "over left",
			list:    func() *LinkedList[int] { return NewLinkedListOf([]int{1, 2, 3, 4, 5}) },
			index:   -1,
			wantErr: fmt.Errorf("ekit: 下标超出范围，长度 %d, 下标 %d", 5, -1),
		},
		{
			name:    "over right",
			list:    func() *LinkedList[int] { return NewLinkedListOf([]int{1, 2, 3, 4, 5}) },
			index:   5,
			wantErr: fmt.Errorf("ekit: 下标超出范围，长度 %d, 下标 %d", 5, 5),
		},
		{
			name:    "empty list",
			list:    func() *LinkedList[int] { return NewLinkedListOf([]int{}) },
			index:   0,
			wantErr: fmt.Errorf("ekit: 下标超出范围，长度 %d, 下标 %d", 0, 0),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			list := tc.list()
			get, err := list.Get(tc.index)
			assert.Equal(t, tc.wantErr, err)
			assert.Equal(t, tc.wantVal, get)
		})
	}
}

// TestLinkedList_Range 测试链表Range方法
// 验证以下场景:
// 1. 遍历空链表
// 2. 遍历非空链表
// 3. 验证遍历顺序
// 4. 测试提前终止遍历
func TestLinkedList_Range(t *testing.T) {
	testCases := []struct {
		name    string
		list    func() *LinkedList[int]
		wantVal int
		wantErr error
	}{
		{
			name:    "计算全部元素的和",
			list:    func() *LinkedList[int] { return NewLinkedListOf([]int{1, 2, 3, 4, 5}) },
			wantVal: 15,
			wantErr: nil,
		},
		{
			name:    "测试中断",
			list:    func() *LinkedList[int] { return NewLinkedListOf([]int{1, 2, 3, 4, -5, 6, 7, 8, -9, 10}) },
			wantErr: errors.New("index 4 is error"),
		},
		{
			name:    "测试数组为nil",
			list:    func() *LinkedList[int] { return NewLinkedListOf([]int{}) },
			wantVal: 0,
			wantErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			list := tc.list()
			result := 0
			err := list.Range(func(index int, num int) error {
				if num < 0 {
					return fmt.Errorf("index %d is error", index)
				}
				result += num
				return nil
			})

			assert.Equal(t, tc.wantErr, err)
			if err != nil {
				return
			}
			assert.Equal(t, tc.wantVal, result)
		})
	}
}

func TestLinkedList_Set(t *testing.T) {
	testCases := []struct {
		name           string
		list           func() *LinkedList[int]
		wantLinkedList *LinkedList[int]
		index          int
		setVal         int
		wantErr        error
	}{
		{
			name:    "set num to index -1",
			list:    func() *LinkedList[int] { return NewLinkedListOf([]int{1, 2, 3}) },
			index:   -1,
			wantErr: fmt.Errorf("ekit: 下标超出范围，长度 %d, 下标 %d", 3, -1),
		},
		{
			name:    "set beyond length index 99",
			list:    func() *LinkedList[int] { return NewLinkedListOf([]int{1, 2, 3}) },
			index:   99,
			wantErr: fmt.Errorf("ekit: 下标超出范围，长度 %d, 下标 %d", 3, 99),
		},
		{
			name:    "set empty node",
			list:    func() *LinkedList[int] { return NewLinkedListOf([]int{}) },
			index:   3,
			wantErr: fmt.Errorf("ekit: 下标超出范围，长度 %d, 下标 %d", 0, 3),
		},
		{
			name:           "set num to index 3",
			list:           func() *LinkedList[int] { return NewLinkedListOf([]int{11, 22, 33, 44, 55}) },
			index:          2,
			setVal:         999,
			wantLinkedList: NewLinkedListOf([]int{11, 22, 999, 44, 55}),
		},
		{
			name:           "set num to head",
			list:           func() *LinkedList[int] { return NewLinkedListOf([]int{11, 22, 33, 44, 55}) },
			index:          0,
			setVal:         -200,
			wantLinkedList: NewLinkedListOf([]int{-200, 22, 33, 44, 55}),
		},
		{
			name:           "set num to tail",
			list:           func() *LinkedList[int] { return NewLinkedListOf([]int{-11, 22, -33, 44, -55, 999, -888}) },
			index:          6,
			setVal:         888,
			wantLinkedList: NewLinkedListOf([]int{-11, 22, -33, 44, -55, 999, 888}),
		},
		{
			name:    "index == len(*node)",
			list:    func() *LinkedList[int] { return NewLinkedListOf([]int{-11, 22, -33, 44, -55, 999, -888}) },
			index:   7,
			setVal:  888,
			wantErr: fmt.Errorf("ekit: 下标超出范围，长度 %d, 下标 %d", 7, 7),
		},
		{
			name:    "len(*node) == 0",
			list:    func() *LinkedList[int] { return NewLinkedListOf([]int{}) },
			index:   0,
			setVal:  888,
			wantErr: fmt.Errorf("ekit: 下标超出范围，长度 %d, 下标 %d", 0, 0),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			list := tc.list()
			err := list.Set(tc.index, tc.setVal)
			if err != nil {
				assert.Equal(t, tc.wantErr, err)
			} else {
				assert.Equal(t, tc.wantLinkedList, list)
			}
		})
	}
}

func BenchmarkLinkedList_Add(b *testing.B) {
	l := NewLinkedListOf([]int{1, 2, 3})
	testCase := make([]int, 0, b.N)
	for i := 1; i <= b.N; i++ {
		testCase = append(testCase, rand.Intn(i))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = l.Add(testCase[i], testCase[i])
	}
}

func BenchmarkLinkedList_Get(b *testing.B) {
	l := NewLinkedListOf([]int{1, 2, 3})
	for i := 1; i <= b.N; i++ {
		err := l.Add(i, i)
		if err != nil {
			b.Fatal(err)
		}
	}
	
	for i := 0; b.Loop(); i++ {
		_, _ = l.Get(i)
	}
}
