package service_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"calculator/internal/domain"
	"calculator/internal/service"
)

// fakeRepository is a deliberately dumb store: it matches requests exactly and counts the
// calls it receives. Key normalisation belongs to the real repository, not here.
type fakeRepository struct {
	mu        sync.Mutex
	stored    map[string]domain.Calculation
	findCalls int
	saveCalls int
	findErr   error
	saveErr   error
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{stored: map[string]domain.Calculation{}}
}

func (f *fakeRepository) key(op domain.Operation, operands []domain.Number) string {
	return fmt.Sprint(op, operands)
}

func (f *fakeRepository) Find(_ context.Context, req domain.Request) (domain.Calculation, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.findCalls++
	if f.findErr != nil {
		return domain.Calculation{}, false, f.findErr
	}
	calc, ok := f.stored[f.key(req.Operation, req.Operands)]
	return calc, ok, nil
}

func (f *fakeRepository) Save(_ context.Context, calc domain.Calculation) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.saveCalls++
	if f.saveErr != nil {
		return f.saveErr
	}
	f.stored[f.key(calc.Operation, calc.Operands)] = calc
	return nil
}

func newCalculator(repo service.Repository) *service.Calculator {
	return service.NewCalculator(repo, slog.New(slog.DiscardHandler))
}

func TestCalculator(t *testing.T) {
	t.Parallel()

	addTwoAndThree := domain.Request{Operation: domain.OpAdd, Operands: []domain.Number{2, 3}}

	t.Run("computes a result that is not yet stored", func(t *testing.T) {
		t.Parallel()
		repo := newFakeRepository()

		got, err := newCalculator(repo).Calculate(context.Background(), addTwoAndThree)

		require.NoError(t, err)
		require.InDelta(t, 5, got.Calculation.Result, 1e-9)
		require.Equal(t, domain.OpAdd, got.Calculation.Operation)
		require.False(t, got.Cached)
		require.Equal(t, 1, repo.saveCalls)
	})

	t.Run("recalls a stored result instead of recomputing it", func(t *testing.T) {
		t.Parallel()
		repo := newFakeRepository()
		calculator := newCalculator(repo)

		first, err := calculator.Calculate(context.Background(), addTwoAndThree)
		require.NoError(t, err)
		require.False(t, first.Cached)

		second, err := calculator.Calculate(context.Background(), addTwoAndThree)

		require.NoError(t, err)
		require.True(t, second.Cached)
		require.InDelta(t, first.Calculation.Result, second.Calculation.Result, 1e-9)
		// Still one save: the second request never reached the engine.
		require.Equal(t, 1, repo.saveCalls)
	})

	t.Run("does not store a failed calculation", func(t *testing.T) {
		t.Parallel()
		repo := newFakeRepository()
		divideByZero := domain.Request{Operation: domain.OpDivide, Operands: []domain.Number{10, 0}}

		_, err := newCalculator(repo).Calculate(context.Background(), divideByZero)

		require.ErrorIs(t, err, domain.ErrDivisionByZero)
		require.Zero(t, repo.saveCalls)
	})

	// The cache is an optimisation, never a dependency: a broken store must not turn a
	// valid calculation into a failed request.
	t.Run("computes anyway when the repository cannot be read", func(t *testing.T) {
		t.Parallel()
		repo := newFakeRepository()
		repo.findErr = errors.New("store unavailable")

		got, err := newCalculator(repo).Calculate(context.Background(), addTwoAndThree)

		require.NoError(t, err)
		require.InDelta(t, 5, got.Calculation.Result, 1e-9)
		require.False(t, got.Cached)
	})

	t.Run("succeeds anyway when the repository cannot be written", func(t *testing.T) {
		t.Parallel()
		repo := newFakeRepository()
		repo.saveErr = errors.New("store unavailable")

		got, err := newCalculator(repo).Calculate(context.Background(), addTwoAndThree)

		require.NoError(t, err)
		require.InDelta(t, 5, got.Calculation.Result, 1e-9)
	})
}
