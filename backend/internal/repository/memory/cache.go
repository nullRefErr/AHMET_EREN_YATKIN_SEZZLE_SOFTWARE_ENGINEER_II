// Package memory keeps completed calculations in the process that computed them.
//
// It implements the repository the service layer declares. Nothing above it knows that
// the answers live in a map, and replacing this package with a database-backed one is a
// change to a single constructor call in main.go.
package memory

import (
	"container/list"
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"
	"sync"

	"calculator/internal/domain"
)

// commutative marks the operations whose operand order does not change the answer, so
// that "2 + 3" and "3 + 2" share one entry instead of taking two.
var commutative = map[domain.Operation]bool{
	domain.OpAdd:      true,
	domain.OpMultiply: true,
}

// entry is what the recency list holds. It carries its own key so that evicting the
// oldest element can also remove it from the index.
type entry struct {
	key         string
	calculation domain.Calculation
}

// Cache is a bounded store of completed calculations that evicts the least recently used
// entry when it is full.
//
// The bound is not an optimisation. An unbounded map would let a client that sends
// endlessly varying operands grow the process until it is killed, so the capacity is a
// limit at a trust boundary.
type Cache struct {
	mu        sync.Mutex
	capacity  int
	index     map[string]*list.Element
	byRecency *list.List // front is the most recently used entry
}

// NewCache returns a Cache holding at most capacity entries.
func NewCache(capacity int) (*Cache, error) {
	if capacity <= 0 {
		return nil, errors.New("memory: cache capacity must be positive")
	}
	return &Cache{
		capacity:  capacity,
		index:     make(map[string]*list.Element, capacity),
		byRecency: list.New(),
	}, nil
}

// Find returns the stored calculation for req, if there is one, and marks it as recently
// used.
func (c *Cache) Find(_ context.Context, req domain.Request) (domain.Calculation, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	element, ok := c.index[key(req.Operation, req.Operands)]
	if !ok {
		return domain.Calculation{}, false, nil
	}
	c.byRecency.MoveToFront(element)
	return element.Value.(*entry).calculation, true, nil
}

// Save stores a completed calculation, evicting the least recently used entry if the
// cache is full.
func (c *Cache) Save(_ context.Context, calculation domain.Calculation) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// The caller is free to reuse its operand slice, so the entry keeps its own copy.
	calculation.Operands = slices.Clone(calculation.Operands)
	entryKey := key(calculation.Operation, calculation.Operands)

	if element, ok := c.index[entryKey]; ok {
		element.Value.(*entry).calculation = calculation
		c.byRecency.MoveToFront(element)
		return nil
	}

	c.index[entryKey] = c.byRecency.PushFront(&entry{key: entryKey, calculation: calculation})
	if c.byRecency.Len() > c.capacity {
		c.evictLeastRecentlyUsed()
	}
	return nil
}

// evictLeastRecentlyUsed drops the entry at the back of the recency list. The caller
// holds the lock.
func (c *Cache) evictLeastRecentlyUsed() {
	oldest := c.byRecency.Back()
	if oldest == nil {
		return
	}
	c.byRecency.Remove(oldest)
	delete(c.index, oldest.Value.(*entry).key)
}

// key builds the index key for an operation and its operands.
//
// Floats are written with strconv.FormatFloat(f, 'g', -1, 64), the shortest form that
// reads back as the same value, so 2 and 2.0 cannot produce two different keys.
func key(operation domain.Operation, operands []domain.Number) string {
	if commutative[operation] {
		operands = slices.Sorted(slices.Values(operands))
	}

	var b strings.Builder
	b.WriteString(string(operation))
	for _, operand := range operands {
		b.WriteByte(':')
		b.WriteString(strconv.FormatFloat(operand, 'g', -1, 64))
	}
	return b.String()
}
