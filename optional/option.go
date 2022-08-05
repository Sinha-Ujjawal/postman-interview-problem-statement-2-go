package optional

import (
	"errors"
	"github_apis/result"
)

var ErrOptional = errors.New("Optional? Value")

type Optional[T any] struct {
	res result.Result[T]
}

func Some[T any](value T) Optional[T] {
	return Optional[T]{result.Ok(value)}
}

func None[T any]() Optional[T] {
	return Optional[T]{result.Err[T](ErrOptional)}
}

func (o Optional[T]) Unwrap() (T, error) {
	return o.res.Unwrap()
}

func Map[T any, U any](f func(T) U, o Optional[T]) Optional[U] {
	var ret Optional[U]
	ret.res = result.MapOk(f, o.res)
	return ret
}

func (o Optional[T]) Do(f func(T), g func(error)) {
	o.res.Do(f, g)
}
