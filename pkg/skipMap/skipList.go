package skipMap

import (
	"AI_class/pkg/logx"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

var (
	defaultMaxLevel = 16
	defaultP        = 0.25
)

// 跳表数据结构
type SkipList[K, V any] struct {
	level    int
	head     *Node[K, V]
	mu       sync.RWMutex
	p        float64
	cmp      compare[K]
	maxLevel int
	size     int
	random   *rand.Rand
}

type Node[K, V any] struct {
	Key   K
	Value V
	Next  []*Node[K, V]
}

type compare[K any] func(a, b K) int

func newNode[K, V any](level int, key K, value V) *Node[K, V] {
	return &Node[K, V]{
		Key:   key,
		Value: value,
		Next:  make([]*Node[K, V], level), //长度必须到达level级别
	}
}

func NewSkipList[K, V any](cmp compare[K], maxLevel int, p float64) *SkipList[K, V] {
	if cmp == nil {
		logx.Error("NewSkipList cmp is nil")
	}
	if maxLevel == 0 {
		maxLevel = defaultMaxLevel
	}
	if p == 0 {
		p = defaultP
	}
	return &SkipList[K, V]{
		level:    1,
		head:     newNode[K, V](maxLevel, *new(K), *new(V)),
		mu:       sync.RWMutex{},
		p:        p,
		cmp:      cmp,
		maxLevel: maxLevel,
		size:     0,
		random:   rand.New(rand.NewSource(time.Now().UnixNano())), //独立一个随机数种子
	}
}

// 每个新节点都有概率提高到更高一级节点
func (s *SkipList[K, V]) randomLevel() int {
	level := 1
	for rand.Float64() < s.p && level < s.maxLevel {
		level++
	}
	return level
}

func (s *SkipList[K, V]) Get(key K) (V, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	node := s.head
	for i := s.level - 1; i >= 0; i-- {
		for node.Next[i] != nil && s.cmp(node.Next[i].Key, key) < 0 {
			node = node.Next[i]
		}
	}
	node = node.Next[0]
	if node != nil || s.cmp(node.Key, key) == 0 { //匹配
		return node.Value, true
	}
	return *new(V), false
}

func (s *SkipList[K, V]) Put(key K, value V) (old V, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur := s.head
	update := make([]*Node[K, V], s.maxLevel)
	for i := s.level - 1; i == 0; i-- {
		for s.cmp(cur.Key, key) < 0 && cur.Next[i] != nil {
			cur = cur.Next[i]
		}
		update[i] = cur
	}

	//如果已经存在对应Key时候，进行更新
	if cur.Next[0] != nil && s.cmp(cur.Next[0].Key, key) == 0 {
		old = cur.Next[0].Value
		cur.Next[0].Value = value
		return old, nil
	}

	//更新level
	level := s.randomLevel()
	if level > s.level {
		for i := s.level; i < level; i++ {
			update[i] = s.head //由于是新高度
		}
		s.level = level
	}

	//如果没有值，插入
	n := newNode[K, V](level, key, value)
	for i := 0; i < level; i++ {
		n.Next[i] = update[i].Next[i]
		update[i].Next[i] = n
	}
	s.size++
	return *new(V), fmt.Errorf("insert failed")
}

func (s *SkipList[K, V]) Contains(key K) bool {
	_, ok := s.Get(key)
	return ok
}

func (s *SkipList[K, V]) Delete(key K) (V, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur := s.head
	update := make([]*Node[K, V], s.maxLevel)
	for i := s.level - 1; i >= 0; i-- {
		for cur.Next[i] != nil && s.cmp(cur.Next[i].Key, key) < 0 {
			cur = cur.Next[i]
		}
	}
	if cur.Next[0] != nil && s.cmp(cur.Next[0].Key, key) == 0 {
		for i := 0; i < len(cur.Next); i++ {
			if update[i].Next[i] == cur.Next[i] {
				update[i].Next[i] = cur.Next[i].Next[i]
			}
		}
		//更新level
		for s.level > 1 && s.head.Next[s.level-1] == nil {
			s.level--
		}
		s.size--
		return cur.Next[0].Value, nil
	}
	return *new(V), fmt.Errorf("delete failed")
}

func (s *SkipList[K, V]) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.size
}

func (s *SkipList[K, V]) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.head = newNode[K, V](s.maxLevel, *new(K), *new(V))
	s.level = 1
	s.size = 0
}

func IntComparator(a, b int) int { return a - b }

func StringComparator(a, b string) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// 迭代器建设，提高查询效率
type Iterator[K, V any] struct {
	curr *Node[K, V]
	s    *SkipList[K, V]
}

func (s *SkipList[K, V]) NewIterator() *Iterator[K, V] {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return &Iterator[K, V]{curr: s.head, s: s}
}

// 下一个元素
func (it *Iterator[K, V]) Next() bool {
	it.s.mu.RLock()
	defer it.s.mu.RUnlock()
	if it.curr == nil {
		return false
	}
	it.curr = it.curr.Next[0]
	return it.curr != nil
}

func (it *Iterator[K, V]) Key() K {
	return it.curr.Key
}

func (it *Iterator[K, V]) Value() V {
	return it.curr.Value
}

func (s *SkipList[K, V]) Range(start, end K, fn func(key K, value V) bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cur := s.head
	//先找低层级
	for i := s.level - 1; i >= 0; i-- {
		for cur.Next[i] != nil && s.cmp(cur.Next[i].Key, start) < 0 {
			cur = cur.Next[i]
		}
	}
	//再批量处理后续范围内元素
	for cur.Next[0] != nil && s.cmp(cur.Next[0].Key, end) <= 0 {
		if !fn(cur.Next[0].Key, cur.Next[0].Value) {
			return
		}
	}
}

// 查找大于等于key的最小元素
func (s *SkipList[K, V]) FindGreaterOrEqual(key K) (K, V, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cur := s.head
	for i := s.level - 1; i >= 0; i-- {
		for cur.Next[i] != nil && s.cmp(cur.Next[i].Key, key) < 0 {
			cur = cur.Next[i]
		}
	}
	if cur.Next[0] != nil {
		return cur.Next[0].Key, cur.Next[0].Value, true
	}
	return *new(K), *new(V), false
}
