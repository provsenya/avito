package httpapi

import (
	"context"
)

type readiness interface{ Ping(context.Context) error }
