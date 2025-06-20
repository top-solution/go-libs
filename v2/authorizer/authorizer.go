package authorizer

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/ory/ladon"
	manager "github.com/ory/ladon/manager/memory"
)

//nolint:gochecknoinits
func init() {
	ladon.ConditionFactories[new(NotInSetCondition).GetName()] = func() ladon.Condition {
		return new(NotInSetCondition)
	}
	ladon.ConditionFactories[new(InSetCondition).GetName()] = func() ladon.Condition {
		return new(InSetCondition)
	}
}

// NotInSetCondition is a Ladon condition that blocks requests containing one or more unwanted context values
type NotInSetCondition struct {
	Denied []string `json:"values"`
}

// GetName returns the name of the NotInSetCondition
func (c *NotInSetCondition) GetName() string {
	return "NotInSetCondition"
}

// Fulfills determines if the NotInSetCondition is fulfilled.
// The NotInSetCondition is fulfilled if the provided strings are not matched in a set
func (c *NotInSetCondition) Fulfills(ctx context.Context, value interface{}, _ *ladon.Request) bool {
	if value == nil {
		return true
	}

	val, ok := value.([]string)
	if !ok {
		if singleVal, ok := value.(string); ok {
			val = []string{singleVal}
		} else {
			return false
		}
	}

	unwanted := make(map[string]struct{}, len(c.Denied))
	for _, el := range c.Denied {
		unwanted[el] = struct{}{}
	}

	for _, el := range val {
		if _, found := unwanted[el]; found {
			return false
		}
	}

	return true
}

// InSetCondition is a Ladon condition that allows requests containing one or more valid context values
type InSetCondition struct {
	Valid []string `json:"values"`
}

// GetName returns the name of the InSetCondition
func (c *InSetCondition) GetName() string {
	return "InSetCondition"
}

// Fulfills determines if the InSetCondition is fulfilled.
// The InSetCondition is fulfilled if at least one of the provided strings are matched in a set
func (c *InSetCondition) Fulfills(ctx context.Context, value interface{}, _ *ladon.Request) bool {
	if value == nil {
		return false
	}

	val, ok := value.([]string)
	if !ok {
		if singleVal, ok := value.(string); ok {
			val = []string{singleVal}
		} else {
			return false
		}
	}

	wanted := make(map[string]struct{}, len(c.Valid))
	for _, el := range c.Valid {
		wanted[el] = struct{}{}
	}

	for _, el := range val {
		if _, found := wanted[el]; found {
			return true
		}
	}

	return false
}

// LadonAuthorizer is a Ladon-backed Authorizer
type LadonAuthorizer struct {
	*ladon.Ladon
	ladon.Policies
}

// NewLadon returns a new Ladon-backed authorizer
func NewLadon() *LadonAuthorizer {
	return &LadonAuthorizer{
		Ladon: &ladon.Ladon{
			Manager: manager.NewMemoryManager(),
		},
	}
}

func (l *LadonAuthorizer) IsUserAllowed(ctx context.Context, r *ladon.Request) error {
	err := l.Ladon.IsAllowed(ctx, r)
	if err != nil {
		return fmt.Errorf("you have no permission for action %s on %s (%v)", r.Action, r.Resource, r.Context)
	}
	return nil
}

func (l *LadonAuthorizer) LoadPoliciesFromJSONS(root string, fsys fs.FS) error {
	return fs.WalkDir(fsys, root, func(path string, info fs.DirEntry, err error) error {
		if filepath.Ext(path) != ".json" {
			return nil
		}

		if err != nil {
			return err
		}

		file, err := fs.ReadFile(fsys, path)
		if err != nil {
			return fmt.Errorf("readfile %s: %w", path, err)
		}

		// Unmarshal policy
		policy := ladon.DefaultPolicy{}
		err = policy.UnmarshalJSON(file)
		if err != nil {
			return fmt.Errorf("unmarshal %s: %w", path, err)
		}
		l.Policies = append(l.Policies, &policy)

		// Create policy
		err = l.Manager.Create(context.TODO(), &policy)
		if err != nil {
			return fmt.Errorf("create policy %s, file %s: %w", policy.ID, path, err)
		}
		return nil
	})
}
