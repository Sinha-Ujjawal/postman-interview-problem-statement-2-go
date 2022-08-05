package result

type Result[T any] struct {
	value T
	err   error
}

func Ok[T any](value T) Result[T] {
	return Result[T]{value: value}
}

func Err[T any](err error) Result[T] {
	return Result[T]{err: err}
}

func (o Result[T]) Unwrap() (T, error) {
	return o.value, o.err
}

func MapOk[T any, U any](f func(T) U, o Result[T]) Result[U] {
	if o.err != nil {
		return Result[U]{err: o.err}
	} else {
		return Result[U]{value: f(o.value)}
	}
}

func (o Result[T]) Do(f func(T), g func(error)) {
	if o.err == nil {
		f(o.value)
	} else {
		g(o.err)
	}
}
