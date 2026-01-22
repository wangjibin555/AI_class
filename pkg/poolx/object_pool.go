package poolx

import "sync"

type ObjectPool[T any] struct {
	pool sync.Pool
	newF func() T //对象词内部对象的创建方法
}

func NewObjectPool[T any](newF func() T) *ObjectPool[T] {
	p := &ObjectPool[T]{
		newF: newF,
		pool: sync.Pool{
			New: func() any {
				data := newF()
				return newPoolObj(data)
			},
		},
	}
	return p
}

func (p *ObjectPool[T]) Put(obj PoolObj[T]) {
	p.pool.Put(obj)
}

func (p *ObjectPool[T]) Get() PoolObj[T] {
	return p.pool.Get().(PoolObj[T])
}
