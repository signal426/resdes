package resdes

import (
	"context"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"
)

type Request[T proto.Message] struct {
	Msg T
}

func NewRequest[T proto.Message](_ context.Context, msg T) *Request[T] {
	return &Request[T]{
		Msg: msg,
	}
}

type Response[U any] struct {
	data     U
	meta     map[string]any
	err      *Error
	errStage Stage
}

func NewEmptyResponse[U any](_ context.Context) *Response[U] {
	return &Response[U]{}
}

func (r *Response[U]) Data() U {
	return r.data
}

func (r *Response[U]) Meta() map[string]any {
	return r.meta
}

func (r *Response[U]) Error() *Error {
	return r.err
}

func (r *Response[U]) SetError(stage Stage, err *Error) *Response[U] {
	r.errStage = stage
	r.err = err
	return r
}

func (r *Response[U]) SetData(data U) *Response[U] {
	r.data = data
	return r
}

func (r *Response[U]) SetMetadata(md map[string]any) *Response[U] {
	r.meta = md
	return r
}

func (r *Response[U]) ToConnect() (*connect.Response[U], *connect.Error) {
	if r.err != nil {
		return nil, connectErrFromResdesErr(r.err)
	}
	return connect.NewResponse(&r.data), nil
}

func connectErrFromResdesErr(err *Error) *connect.Error {
	switch {
	case err.GetAuthError() != nil:
		return connect.NewError(connect.CodeUnauthenticated, err.GetAuthError())
	case err.GetValidationErrors() != nil:
		return connect.NewError(connect.CodeInvalidArgument, err.GetValidationErrors())
	default:
		return connect.NewError(connect.CodeInternal, err.GetServeError())
	}
}
