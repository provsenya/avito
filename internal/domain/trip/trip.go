package trip

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusActive    Status = "active"
	StatusCompleted Status = "completed"
)

var (
	ErrNotFound   = errors.New("trip not found")
	ErrCompleted  = errors.New("trip already completed")
	ErrDriverBusy = errors.New("driver already has an active trip")
)

type Coordinates struct {
	Latitude  float64
	Longitude float64
}

type Trip struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	DriverID       uuid.UUID
	StartPoint     Coordinates
	EndPoint       Coordinates
	Price          int64
	Status         Status
	StartedAt      time.Time
	FinishedAt     *time.Time
	LastPositionAt *time.Time
}

func (t *Trip) Finish(at time.Time) error {
	if t.Status != StatusActive {
		return ErrCompleted
	}
	t.Status = StatusCompleted
	t.FinishedAt = &at
	return nil
}

type ValidationError struct{ Field, Message string }

func (e *ValidationError) Error() string { return e.Field + ": " + e.Message }

type CreateInput struct {
	UserID, DriverID     uuid.UUID
	StartPoint, EndPoint Coordinates
	Price                int64
}

func (v CreateInput) Validate() error {
	if v.UserID == uuid.Nil {
		return &ValidationError{"user_id", "must be a nonempty UUID"}
	}
	if v.DriverID == uuid.Nil {
		return &ValidationError{"driver_id", "must be a nonempty UUID"}
	}
	if err := v.StartPoint.Validate("start_point"); err != nil {
		return err
	}
	if err := v.EndPoint.Validate("end_point"); err != nil {
		return err
	}
	if v.Price < 0 {
		return &ValidationError{"price", "must be nonnegative"}
	}
	return nil
}

func (v Coordinates) Validate(field string) error {
	if !(v.Latitude >= -90 && v.Latitude <= 90) {
		return &ValidationError{field + ".latitude", "must be between -90 and 90"}
	}
	if !(v.Longitude >= -180 && v.Longitude <= 180) {
		return &ValidationError{field + ".longitude", "must be between -180 and 180"}
	}
	return nil
}
