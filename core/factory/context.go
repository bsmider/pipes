package factory

import (
	"context"
	"fmt"
	sync "sync"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

type ContextWrapper struct {
	mu  sync.Mutex
	ctx *Context
}

type contextKey string

const protoContexWrappertKey = "protoContextWrapper"
const hopsKey contextKey = "hops"
const traceIdKey contextKey = "traceId"
const defaultTimeout = 30 * time.Second

func (ctx *Context) AddHop(binaryId string) error {
	if ctx == nil {
		return fmt.Errorf("context is nil")
	}

	newHop := &Hop{
		BinaryId:  binaryId,
		Timestamp: timestamppb.Now(),
	}

	ctx.Hops = append(ctx.Hops, newHop)

	return nil
}

func AddHop(ctx context.Context, binaryId string) error {
	if ctx == nil {
		return fmt.Errorf("context is nil")
	}

	// 1. Restore Proto Context
	if protoContextWrapper, ok := ctx.Value(protoContexWrappertKey).(*ContextWrapper); ok {
		return protoContextWrapper.ctx.AddHop(binaryId)
	}

	return fmt.Errorf("context is not a proto context")
}

func UpdateContext(ctx context.Context, newContext *Context) error {
	if ctx == nil {
		return fmt.Errorf("context is nil")
	}

	// 1. Restore Proto Context
	if protoContextWrapper, ok := ctx.Value(protoContexWrappertKey).(*ContextWrapper); ok {
		protoContextWrapper.ctx = newContext
		return nil
	}

	return fmt.Errorf("context is not a proto context")
}

// ToGoContext converts the Proto Context into a Go context.Context.
// It preserves the deadline, trace ID, and hop history.
func (ctx *Context) ToGoContext() (context.Context, context.CancelFunc) {
	// 1. Start with Background
	goCtx := context.Background()

	// 2. Handle Deadline
	// Note: 'fctx' is the pointer to the Context struct
	if ctx.Deadline != nil {
		deadline := ctx.Deadline.AsTime()
		goCtx, _ = context.WithDeadline(goCtx, deadline)
	} else {
		// Default fallback if no deadline is provided in the packet
		goCtx, _ = context.WithTimeout(goCtx, defaultTimeout)
	}

	// store the factory context
	goCtx = context.WithValue(goCtx, protoContexWrappertKey, ctx)

	// Return the wrapped context and the cancel function
	return context.WithCancel(goCtx)
}

// FromGoContext populates the receiver with values extracted from a standard context.
// Usage: ctx := (&factory.Context{}).FromGoContext(goCtx)
func (m *Context) FromGoContext(ctx context.Context) *Context {
	if m == nil {
		m = &Context{}
	}

	// 1. Restore Proto Context
	if protoContextWrapper, ok := ctx.Value(protoContexWrappertKey).(*ContextWrapper); ok {
		m = protoContextWrapper.ctx
	}

	return m
}
