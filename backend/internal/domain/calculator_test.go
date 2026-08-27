package domain_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"calculator/internal/domain"
)

func req(op domain.Operation, operands ...domain.Number) domain.Request {
	return domain.Request{Operation: op, Operands: operands}
}

// The cases below mirror the edge-case table in SPEC.md §5. Every row there is a row here.
func TestCalculate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		req     domain.Request
		want    domain.Number
		wantErr error
	}{
		{"add", req(domain.OpAdd, 2, 3), 5, nil},
		// Zero is a valid operand. A naive "required" validation rejects this.
		{"add zero to zero", req(domain.OpAdd, 0, 0), 0, nil},
		{"add negative operand", req(domain.OpAdd, -2, 3), 1, nil},

		{"subtract", req(domain.OpSubtract, 10, 4), 6, nil},
		// Subnormal values underflow to zero. That is a result, not an error.
		{"subtract subnormals", req(domain.OpSubtract, 1e-320, 1e-320), 0, nil},

		{"multiply", req(domain.OpMultiply, 6, 7), 42, nil},
		{"multiply overflows", req(domain.OpMultiply, 1e308, 10), 0, domain.ErrOverflow},

		{"divide", req(domain.OpDivide, 1, 3), 0.3333333333333333, nil},
		{"divide by zero", req(domain.OpDivide, 10, 0), 0, domain.ErrDivisionByZero},
		// Guarded before the result could become NaN.
		{"divide zero by zero", req(domain.OpDivide, 0, 0), 0, domain.ErrDivisionByZero},

		// math.Pow(0, 0) is 1 by definition in Go. Documented in the README.
		{"power of zero to zero", req(domain.OpPower, 0, 0), 1, nil},
		{"power overflows", req(domain.OpPower, 2, 10000), 0, domain.ErrOverflow},
		{"power divides by zero", req(domain.OpPower, 0, -1), 0, domain.ErrOverflow},
		{"power of negative base to fractional exponent", req(domain.OpPower, -8, 0.5), 0, domain.ErrUndefinedResult},

		{"square root", req(domain.OpSqrt, 9), 3, nil},
		{"square root of zero", req(domain.OpSqrt, 0), 0, nil},
		{"square root of negative", req(domain.OpSqrt, -4), 0, domain.ErrNegativeSqrt},

		// percentage(a, b) is "a% of b" — see SPEC.md D-04.
		{"percentage", req(domain.OpPercentage, 50, 200), 100, nil},

		{"unsupported operation", req("modulo", 5, 2), 0, domain.ErrUnsupportedOp},

		{"no operands", req(domain.OpAdd), 0, domain.ErrInvalidOperandLen},
		{"too many operands", req(domain.OpAdd, 1, 2, 3), 0, domain.ErrInvalidOperandLen},
		{"square root takes one operand", req(domain.OpSqrt, 2, 3), 0, domain.ErrInvalidOperandLen},

		// "1e400" is valid JSON but decodes to +Inf, so operands are checked after decoding.
		{"infinite operand", req(domain.OpAdd, math.Inf(1), 2), 0, domain.ErrInvalidOperand},
		{"not a number operand", req(domain.OpAdd, math.NaN(), 2), 0, domain.ErrInvalidOperand},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := domain.Calculate(tt.req)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.InDelta(t, tt.want, got, 1e-9)
		})
	}
}

func TestOperandCount(t *testing.T) {
	t.Parallel()

	t.Run("square root takes one operand", func(t *testing.T) {
		t.Parallel()

		count, ok := domain.OperandCount(domain.OpSqrt)

		require.True(t, ok)
		require.Equal(t, 1, count)
	})

	t.Run("unknown operation is not supported", func(t *testing.T) {
		t.Parallel()

		_, ok := domain.OperandCount("modulo")

		require.False(t, ok)
	})
}

func TestOperations(t *testing.T) {
	t.Parallel()

	got := domain.Operations()

	require.Len(t, got, 7)
	require.Contains(t, got, domain.OpAdd)
	require.NotContains(t, got, domain.Operation("modulo"))
}
