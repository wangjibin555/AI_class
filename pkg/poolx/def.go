package poolx

type PoolObj[T any] struct {
	data T
}

func newPoolObj[T any](data T) PoolObj[T] {
	return PoolObj[T]{
		data: data,
	}
}

func (p PoolObj[T]) GetData() T {
	return p.data
}
