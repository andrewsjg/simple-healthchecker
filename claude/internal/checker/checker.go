package checker

import (
	"context"
	"fmt"

	"github.com/andrewsjg/simple-healthchecker/claude/pkg/models"
)

// Checker is the interface that all health checkers must implement
type Checker interface {
	Check(ctx context.Context, host models.Host, check models.Check) models.CheckResult
	Type() models.CheckType
}

// Registry holds all registered checkers
type Registry struct {
	checkers map[models.CheckType]Checker
}

// NewRegistry creates a new checker registry
func NewRegistry() *Registry {
	return &Registry{
		checkers: make(map[models.CheckType]Checker),
	}
}

// Register registers a checker
func (r *Registry) Register(checker Checker) {
	r.checkers[checker.Type()] = checker
}

// Get retrieves a checker by type
func (r *Registry) Get(checkType models.CheckType) (Checker, error) {
	checker, ok := r.checkers[checkType]
	if !ok {
		return nil, fmt.Errorf("no checker registered for type: %s", checkType)
	}
	return checker, nil
}

// GetAll returns all registered checkers
func (r *Registry) GetAll() map[models.CheckType]Checker {
	return r.checkers
}
