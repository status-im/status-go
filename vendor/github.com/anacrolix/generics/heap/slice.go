package heap

type sliceInterface[T any] struct {
	slice *[]T
	less  func(T, T) bool
}

func (me sliceInterface[T]) Less(i, j int) bool {
	return me.less((*me.slice)[i], (*me.slice)[j])
}

func (me sliceInterface[T]) Swap(i, j int) {
	s := *me.slice
	s[i], s[j] = s[j], s[i]
	*me.slice = s
}

func (me sliceInterface[T]) Push(x T) {
	*me.slice = append(*me.slice, x)
}

func (me sliceInterface[T]) Pop() T {
	s := *me.slice
	n := len(s)
	ret := s[n-1]
	*me.slice = s[:n-1]
	return ret
}

func (me sliceInterface[T]) Len() int {
	return len(*me.slice)
}

func InterfaceForSlice[T any](sl *[]T, less func(T, T) bool) Interface[T] {
	return sliceInterface[T]{
		slice: sl,
		less:  less,
	}
}
