package factory

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ContextWrapper struct {
	mu  sync.Mutex
	ctx *Context
}

type contextKey string

const protoContexWrappertKey = "protoContextWrapper"
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

func NewContext(deadline *timestamppb.Timestamp, traceId string, hops []*Hop) *Context {
	return &Context{
		Deadline: deadline,
		TraceId:  traceId,
		Hops:     hops,
	}
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
	goCtx = context.WithValue(goCtx, protoContexWrappertKey, &ContextWrapper{ctx: ctx})

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

// PrintDetails outputs the context structure to the console in a readable format.
func (m *Context) PrintDetails() {
	if m == nil {
		fmt.Fprintln(os.Stderr, "[Context] <nil>")
		return
	}

	// Configure the marshaler for "pretty printing"
	options := protojson.MarshalOptions{
		Multiline:       true,
		Indent:          "  ",
		EmitUnpopulated: true,
	}

	jsonBytes, err := options.Marshal(m)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[Context Error] Could not format context: %v\n", err)
		return
	}

	fmt.Fprintf(os.Stderr, "--- Context Detail ---\n%s\n----------------------\n", string(jsonBytes))
}

// PrintDetails extracts the factory context from the Go context and prints its details.
func PrintDetails(ctx context.Context) {
	if wrapper, ok := ctx.Value(protoContexWrappertKey).(*ContextWrapper); ok {
		wrapper.ctx.PrintDetails()
	} else {
		fmt.Fprintln(os.Stderr, "[factory.PrintDetails] Warning: context does not contain a factory context")
	}
}
