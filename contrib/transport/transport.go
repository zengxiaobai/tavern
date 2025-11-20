package transport

import "context"

// Server is transport server.
type Server interface {
	Start(context.Context) error
	Stop(context.Context) error
}

type AppContext interface {
	Kind() Kind
}

type Kind string

func (k Kind) String() string {
	return string(k)
}

type (
	serverAppContext struct{}
)

func NewContext(ctx context.Context, appCtx AppContext) context.Context {
	return context.WithValue(ctx, serverAppContext{}, appCtx)
}

func FromContext(ctx context.Context) AppContext {
	return nil
}
