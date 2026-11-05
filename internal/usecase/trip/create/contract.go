//go:generate go tool mockgen -source contract.go -destination mocks_test.go -package create_test
package create

import (
	"context"

	"github.com/google/uuid"

	"github.com/provsenya/avito/internal/domain/trip"
	"github.com/provsenya/avito/internal/model"
)

type tripRepository interface {
	Create(context.Context, model.Trip) error
}
type txManager interface {
	Do(context.Context, func(context.Context) error) error
}
type statusHistory interface {
	Add(context.Context, uuid.UUID, *trip.Status, trip.Status, *string) error
}
type events interface{ Record(context.Context, string) }
