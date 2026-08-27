package rest_test

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"calculator/internal/repository/memory"
	"calculator/internal/service"
	"calculator/internal/transport/rest"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	m.Run()
}

// newRouter wires the real service over a real cache. The service is a concrete type on
// purpose — an interface with a single implementation would only exist to be faked, which
// CLAUDE.md API-2 rules out — so the tests exercise the whole request path instead.
func newRouter(t *testing.T) *gin.Engine {
	t.Helper()

	cache, err := memory.NewCache(16)
	require.NoError(t, err)

	return rest.NewRouter(service.NewCalculator(cache, slog.New(slog.DiscardHandler)), rest.Config{
		Logger: slog.New(slog.DiscardHandler),
	})
}

// newRouterAllowing builds a router with CORS enabled for the given origins.
func newRouterAllowing(t *testing.T, origins ...string) *gin.Engine {
	t.Helper()

	cache, err := memory.NewCache(16)
	require.NoError(t, err)

	return rest.NewRouter(service.NewCalculator(cache, slog.New(slog.DiscardHandler)), rest.Config{
		Logger:         slog.New(slog.DiscardHandler),
		AllowedOrigins: origins,
	})
}

func post(t *testing.T, router *gin.Engine, body string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/calculations", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func decode[T any](t *testing.T, recorder *httptest.ResponseRecorder) T {
	t.Helper()

	var out T
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &out))
	return out
}

type calculationBody struct {
	Operation string    `json:"operation"`
	Operands  []float64 `json:"operands"`
	Result    float64   `json:"result"`
	Cached    bool      `json:"cached"`
}

type errorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	RequestID string `json:"request_id"`
}

func TestCreateCalculation(t *testing.T) {
	t.Parallel()

	t.Run("returns the result", func(t *testing.T) {
		t.Parallel()

		recorder := post(t, newRouter(t), `{"operation":"add","operands":[2,3]}`)

		require.Equal(t, http.StatusOK, recorder.Code)
		body := decode[calculationBody](t, recorder)
		require.Equal(t, "add", body.Operation)
		require.Equal(t, []float64{2, 3}, body.Operands)
		require.InDelta(t, 5, body.Result, 1e-9)
	})

	// Zero is a valid operand. Validating with a "required" rule would reject this,
	// because Go cannot tell a zero that was sent from a field that was omitted.
	t.Run("accepts zero as an operand", func(t *testing.T) {
		t.Parallel()

		recorder := post(t, newRouter(t), `{"operation":"add","operands":[0,0]}`)

		require.Equal(t, http.StatusOK, recorder.Code)
		require.InDelta(t, 0, decode[calculationBody](t, recorder).Result, 1e-9)
	})

	t.Run("reports whether the result was recalled", func(t *testing.T) {
		t.Parallel()
		router := newRouter(t)

		first := post(t, router, `{"operation":"multiply","operands":[6,7]}`)
		second := post(t, router, `{"operation":"multiply","operands":[6,7]}`)

		require.False(t, decode[calculationBody](t, first).Cached)
		require.True(t, decode[calculationBody](t, second).Cached)
	})

	t.Run("carries a request id on every response", func(t *testing.T) {
		t.Parallel()

		recorder := post(t, newRouter(t), `{"operation":"add","operands":[2,3]}`)

		require.NotEmpty(t, recorder.Header().Get("X-Request-Id"))
	})
}

func TestCreateCalculationRejectsBadRequests(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		code string
	}{
		{"malformed json", `{"operation":"add",`, "INVALID_REQUEST"},
		{"operand is null", `{"operation":"add","operands":[null,2]}`, "INVALID_REQUEST"},
		{"operand is not a number", `{"operation":"add","operands":["a",2]}`, "INVALID_REQUEST"},
		{"operand is out of range", `{"operation":"add","operands":[1e400,2]}`, "INVALID_REQUEST"},
		{"unknown operation", `{"operation":"modulo","operands":[5,2]}`, "INVALID_REQUEST"},
		{"missing operation", `{"operands":[5,2]}`, "INVALID_REQUEST"},
		{"no operands", `{"operation":"add","operands":[]}`, "INVALID_OPERAND_COUNT"},
		{"too many operands", `{"operation":"add","operands":[1,2,3]}`, "INVALID_OPERAND_COUNT"},
		{"square root takes one operand", `{"operation":"sqrt","operands":[2,3]}`, "INVALID_OPERAND_COUNT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			recorder := post(t, newRouter(t), tt.body)

			require.Equal(t, http.StatusBadRequest, recorder.Code)
			require.Equal(t, tt.code, decode[errorBody](t, recorder).Error.Code)
		})
	}
}

func TestCreateCalculationRejectsImpossibleMath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		code string
	}{
		{"division by zero", `{"operation":"divide","operands":[10,0]}`, "DIVISION_BY_ZERO"},
		{"zero divided by zero", `{"operation":"divide","operands":[0,0]}`, "DIVISION_BY_ZERO"},
		{"square root of a negative", `{"operation":"sqrt","operands":[-4]}`, "NEGATIVE_SQRT"},
		{"multiplication overflows", `{"operation":"multiply","operands":[1e308,10]}`, "NUMERIC_OVERFLOW"},
		{"exponentiation overflows", `{"operation":"power","operands":[2,10000]}`, "NUMERIC_OVERFLOW"},
		{"undefined result", `{"operation":"power","operands":[-8,0.5]}`, "UNDEFINED_RESULT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			recorder := post(t, newRouter(t), tt.body)

			require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
			body := decode[errorBody](t, recorder)
			require.Equal(t, tt.code, body.Error.Code)
			require.NotEmpty(t, body.Error.Message, "the message is what the user reads")
			require.NotEmpty(t, body.RequestID, "the request id is what ties a report to the logs")
		})
	}
}

func TestListOperations(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/operations", nil)
	recorder := httptest.NewRecorder()
	newRouter(t).ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)

	body := decode[struct {
		Operations []struct {
			Name     string `json:"name"`
			Operands int    `json:"operands"`
		} `json:"operations"`
	}](t, recorder)

	require.Len(t, body.Operations, 7)
	require.Equal(t, "add", body.Operations[0].Name)
	require.Equal(t, 2, body.Operations[0].Operands)
	require.Contains(t, body.Operations, struct {
		Name     string `json:"name"`
		Operands int    `json:"operands"`
	}{Name: "sqrt", Operands: 1})
}

func TestHealth(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/healthz", nil)
	recorder := httptest.NewRecorder()
	newRouter(t).ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
}

// The Origin header is client input that the response reflects back, so what is not
// allowed matters more here than what is.
func TestCORS(t *testing.T) {
	t.Parallel()

	const allowed = "http://localhost:3000"

	send := func(t *testing.T, method, origin string) *httptest.ResponseRecorder {
		t.Helper()

		request := httptest.NewRequestWithContext(t.Context(), method, "/api/v1/operations", nil)
		request.Header.Set("Origin", origin)
		recorder := httptest.NewRecorder()
		newRouterAllowing(t, allowed).ServeHTTP(recorder, request)
		return recorder
	}

	t.Run("allows an origin that is configured", func(t *testing.T) {
		t.Parallel()

		recorder := send(t, http.MethodGet, allowed)

		require.Equal(t, allowed, recorder.Header().Get("Access-Control-Allow-Origin"))
		require.Equal(t, "Origin", recorder.Header().Get("Vary"))
	})

	// Reflecting any origin back would hand every site on the internet a same-origin
	// channel to this API.
	t.Run("does not reflect an origin that is not configured", func(t *testing.T) {
		t.Parallel()

		recorder := send(t, http.MethodGet, "http://evil.example")

		require.Empty(t, recorder.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("answers a preflight without reaching the handler", func(t *testing.T) {
		t.Parallel()

		recorder := send(t, http.MethodOptions, allowed)

		require.Equal(t, http.StatusNoContent, recorder.Code)
		require.Empty(t, recorder.Body.String())
	})

	t.Run("sends no CORS headers when no origin is configured", func(t *testing.T) {
		t.Parallel()

		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/operations", nil)
		request.Header.Set("Origin", allowed)
		recorder := httptest.NewRecorder()
		newRouter(t).ServeHTTP(recorder, request)

		require.Empty(t, recorder.Header().Get("Access-Control-Allow-Origin"))
	})
}
