// Package service orchestrates a calculation: look for a stored answer, compute one if
// there is none, and remember it. It knows nothing about HTTP and nothing about where the
// answers are kept.
package service

import (
	"context"
	"log/slog"

	"calculator/internal/domain"
)

// Repository stores completed calculations so that an identical request is not computed
// twice.
//
// The interface is declared here, by the consumer, rather than by the package that
// implements it. That keeps the dependency pointing inward: this package names no storage
// type, and swapping the in-memory store for a database touches only main.go.
type Repository interface {
	// Find returns the stored calculation for req, if there is one.
	Find(ctx context.Context, req domain.Request) (domain.Calculation, bool, error)
	// Save stores a completed calculation.
	Save(ctx context.Context, calc domain.Calculation) error
}

// Result is a calculation together with whether it was recalled rather than computed.
type Result struct {
	Calculation domain.Calculation
	Cached      bool
}

// Calculator answers calculation requests.
type Calculator struct {
	repo Repository
	log  *slog.Logger
}

// NewCalculator returns a Calculator backed by repo.
func NewCalculator(repo Repository, log *slog.Logger) *Calculator {
	return &Calculator{repo: repo, log: log}
}

// Calculate answers req, recalling a stored result when one exists.
//
// The repository is an optimisation, never a dependency: if it cannot be read or written,
// the failure is logged and the calculation proceeds. A broken cache must not turn a
// valid request into a failed one.
func (c *Calculator) Calculate(ctx context.Context, req domain.Request) (Result, error) {
	stored, found, err := c.repo.Find(ctx, req)
	switch {
	case err != nil:
		c.log.WarnContext(ctx, "calculation lookup failed", "error", err)
	case found:
		return Result{Calculation: stored, Cached: true}, nil
	}

	result, err := domain.Calculate(req)
	if err != nil {
		// Failures are not stored: they are cheap to reproduce and would only take up
		// room that successful answers can use.
		return Result{}, err
	}

	calculation := domain.Calculation{
		Operation: req.Operation,
		Operands:  req.Operands,
		Result:    result,
	}
	if err := c.repo.Save(ctx, calculation); err != nil {
		c.log.WarnContext(ctx, "calculation store failed", "error", err)
	}
	return Result{Calculation: calculation}, nil
}
