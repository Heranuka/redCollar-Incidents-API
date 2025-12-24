package e

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func Wrap(message string, err error) error {
	return fmt.Errorf("%s: %w", message, err)
}

var (
	ErrNotFound           = errors.New("not found")
	ErrConflict           = errors.New("conflict")
	ErrInvalidInput       = errors.New("invalid input")
	ErrInternal           = errors.New("internal error")
	ErrDeadline           = errors.New("deadline exceeded")
	ErrCanceled           = errors.New("context canceled")
	ErrUniqueViolation    = errors.New("unique violation")
	ErrInvalidCoordinates = errors.New("invalid coordinates")
	ErrInvalidUserID      = errors.New("invalid user_id")
	ErrWebHookEmpty       = errors.New("webhook queue is empty")
)

func WrapError(ctx context.Context, op string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("%s: %w", op, ErrDeadline)
	}
	if errors.Is(err, context.Canceled) {
		return fmt.Errorf("%s: %w", op, ErrCanceled)
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return fmt.Errorf("%s: %w", op, ErrUniqueViolation)
		case "23503", "23514":
			return fmt.Errorf("%s: %w", op, ErrInvalidInput)
		default:
			return fmt.Errorf("%s: pg error %s: %w", op, pgErr.Code, ErrInternal)
		}
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%s: %w", op, ErrNotFound)
	}
	return fmt.Errorf("%s: %w", op, ErrInternal)
}
