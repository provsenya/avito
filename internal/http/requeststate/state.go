package requeststate

import (
	"context"
)

type State struct {
	Stack   string
	Err     error
	Status  int
	Context context.Context
	Route   string
}
type key struct{}

func With(ctx context.Context, s *State) context.Context { return context.WithValue(ctx, key{}, s) }
func From(ctx context.Context) *State                    { s, _ := ctx.Value(key{}).(*State); return s }
