package resdes

import (
	"context"

	"google.golang.org/protobuf/proto"
)

// TODO(signal426)
type Request[T proto.Message] struct {
	Msg     T
	Headers map[string]string
	Claims  map[string]any
	Extras  map[string]any
}

func NewRequest[T proto.Message](ctx context.Context, msg T) *Request[T] {
	return &Request[T]{
		Msg: msg,
	}
}

type Response[U any] struct {
	Data  U
	Meta  map[string]any
	Error *Error
}

func NewResponse[U any](data U, err *Error, meta map[string]any) *Response[U] {
	return &Response[U]{
		Data:  data,
		Error: err,
		Meta:  meta,
	}
}
