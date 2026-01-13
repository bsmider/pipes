package factory

import (
	pstatus "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func NewError(status *pstatus.Status) *Error {
	return &Error{
		Status: status,
	}
}

func (err *Error) ToGoError() error {
	if err == nil {
		return nil
	}
	s := err.Status
	if s == nil {
		return nil
	}
	return status.FromProto(s).Err()
}

// FromGoError populates the factory.Error struct from a standard Go error.
// It handles both gRPC status errors and generic Go errors.
func (m *Error) FromGoError(err error) *Error {
	if err == nil {
		return nil
	}

	// If the receiver is nil, create a new one
	if m == nil {
		m = &Error{}
	}

	// 1. Check if it's already a gRPC status error
	st, ok := status.FromError(err)
	if ok {
		m.Status = st.Proto()
		return m
	}

	// 2. Fallback: Wrap generic error as "Internal"
	m.Status = status.New(codes.Internal, err.Error()).Proto()
	return m
}
