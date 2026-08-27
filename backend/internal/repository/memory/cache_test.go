package memory_test

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"calculator/internal/domain"
	"calculator/internal/repository/memory"
)

func calculation(op domain.Operation, result domain.Number, operands ...domain.Number) domain.Calculation {
	return domain.Calculation{Operation: op, Operands: operands, Result: result}
}

func request(op domain.Operation, operands ...domain.Number) domain.Request {
	return domain.Request{Operation: op, Operands: operands}
}

func newCache(t *testing.T, capacity int) *memory.Cache {
	t.Helper()

	cache, err := memory.NewCache(capacity)
	require.NoError(t, err)
	return cache
}

func TestCacheStoresAndRecalls(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cache := newCache(t, 8)

	require.NoError(t, cache.Save(ctx, calculation(domain.OpAdd, 5, 2, 3)))
	got, found, err := cache.Find(ctx, request(domain.OpAdd, 2, 3))

	require.NoError(t, err)
	require.True(t, found)
	require.InDelta(t, 5, got.Result, 1e-9)
}

func TestCacheMissOnEmpty(t *testing.T) {
	t.Parallel()

	_, found, err := newCache(t, 8).Find(context.Background(), request(domain.OpAdd, 2, 3))

	require.NoError(t, err)
	require.False(t, found)
}

func TestCacheKeyIsDeterministicAcrossEquivalentFloats(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cache := newCache(t, 8)

	// 2 and 2.0 are the same float64 and must not produce two entries.
	require.NoError(t, cache.Save(ctx, calculation(domain.OpMultiply, 6, 2, 3)))
	_, found, err := cache.Find(ctx, request(domain.OpMultiply, 2.0, 3.0))

	require.NoError(t, err)
	require.True(t, found)
}

func TestCacheOperandOrder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		operation domain.Operation
		result    domain.Number
		saved     []domain.Number
		lookup    []domain.Number
		wantFound bool
	}{
		// Commutative operations share one entry, which roughly doubles the hit rate.
		{"addition is commutative", domain.OpAdd, 5, []domain.Number{2, 3}, []domain.Number{3, 2}, true},
		{"multiplication is commutative", domain.OpMultiply, 6, []domain.Number{2, 3}, []domain.Number{3, 2}, true},
		// For the rest the operand order changes the answer, so it must change the key.
		{"subtraction is not commutative", domain.OpSubtract, 6, []domain.Number{10, 4}, []domain.Number{4, 10}, false},
		// 2^3 and 3^2 differ, which is the case a reader is most likely to doubt.
		{"exponentiation is not commutative", domain.OpPower, 8, []domain.Number{2, 3}, []domain.Number{3, 2}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			cache := newCache(t, 8)

			require.NoError(t, cache.Save(ctx, calculation(tt.operation, tt.result, tt.saved...)))
			_, found, err := cache.Find(ctx, request(tt.operation, tt.lookup...))

			require.NoError(t, err)
			require.Equal(t, tt.wantFound, found)
		})
	}
}

func TestCacheEvictsLeastRecentlyUsed(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cache := newCache(t, 2)

	require.NoError(t, cache.Save(ctx, calculation(domain.OpAdd, 3, 1, 2)))
	require.NoError(t, cache.Save(ctx, calculation(domain.OpAdd, 7, 3, 4)))
	require.NoError(t, cache.Save(ctx, calculation(domain.OpAdd, 11, 5, 6)))

	_, found, err := cache.Find(ctx, request(domain.OpAdd, 1, 2))
	require.NoError(t, err)
	require.False(t, found, "the oldest entry should have been evicted")

	_, found, err = cache.Find(ctx, request(domain.OpAdd, 5, 6))
	require.NoError(t, err)
	require.True(t, found)
}

func TestCacheFindRefreshesRecency(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cache := newCache(t, 2)

	require.NoError(t, cache.Save(ctx, calculation(domain.OpAdd, 3, 1, 2)))
	require.NoError(t, cache.Save(ctx, calculation(domain.OpAdd, 7, 3, 4)))

	// Touching the oldest entry makes the other one the eviction candidate.
	_, found, err := cache.Find(ctx, request(domain.OpAdd, 1, 2))
	require.NoError(t, err)
	require.True(t, found)

	require.NoError(t, cache.Save(ctx, calculation(domain.OpAdd, 11, 5, 6)))

	_, found, err = cache.Find(ctx, request(domain.OpAdd, 1, 2))
	require.NoError(t, err)
	require.True(t, found, "the refreshed entry should have survived")

	_, found, err = cache.Find(ctx, request(domain.OpAdd, 3, 4))
	require.NoError(t, err)
	require.False(t, found)
}

func TestCacheRejectsNonPositiveCapacity(t *testing.T) {
	t.Parallel()

	_, err := memory.NewCache(0)

	require.Error(t, err)
}

func TestCacheCopiesOperands(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cache := newCache(t, 8)
	operands := []domain.Number{2, 3}

	require.NoError(t, cache.Save(ctx, domain.Calculation{
		Operation: domain.OpAdd, Operands: operands, Result: 5,
	}))
	operands[0] = 99 // the caller reusing its slice must not corrupt the stored entry

	got, found, err := cache.Find(ctx, request(domain.OpAdd, 2, 3))

	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, []domain.Number{2, 3}, got.Operands)
}

func TestCacheIsSafeForConcurrentUse(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cache := newCache(t, 16)

	var wg sync.WaitGroup
	for i := range 64 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			operand := domain.Number(i)
			_ = cache.Save(ctx, calculation(domain.OpAdd, operand+1, operand, 1))
			_, _, _ = cache.Find(ctx, request(domain.OpAdd, operand, 1))
		}()
	}
	wg.Wait()
}
