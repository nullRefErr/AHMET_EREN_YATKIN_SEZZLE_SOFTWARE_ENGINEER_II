package domain

// Calculation is a completed calculation: the request that produced it and the answer.
// It is the entity the repository stores, which is why the repository interface can be
// written without naming a single storage type.
type Calculation struct {
	Operation Operation
	Operands  []Number
	Result    Number
}
