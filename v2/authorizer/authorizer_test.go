package authorizer

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/ory/ladon"
)

// TestLadonWarden tests the loading of policies from json files
// The policies files are contained in the testdata/policies
// They are loaded leveraging the fs.FS interface, so you can use any valid implementation, for example embed.FS from Go 1.16
func TestLadonWarden(t *testing.T) {
	ctx := context.TODO()
	l := NewLadon()
	err := l.LoadPoliciesFromJSONS("_policies", os.DirFS("./testdata"))
	if err != nil {
		t.Error(err)
		return
	}

	err = l.IsAllowed(ctx, &ladon.Request{
		Action:   "pass",
		Resource: "test",
		Subject:  "this",
	})
	if err != nil {
		t.Error(err)
	}

	err = l.IsAllowed(ctx, &ladon.Request{
		Action:   "pass",
		Resource: "test",
		Subject:  "that",
	})
	if err != nil {
		t.Error(err)
	}
}

func TestNotInSetCondition(t *testing.T) {
	ctx := context.TODO()
	cases := []struct {
		Name     string
		Unwanted []string
		AskedFor interface{}
		Fulfills bool
	}{
		{
			Name:     "single unwanted thing, single matching input value",
			Unwanted: []string{"unwanted"},
			AskedFor: []string{"unwanted"},
			Fulfills: false,
		},
		{
			Name:     "single unwanted thing, single unmatching input value",
			Unwanted: []string{"unwanted"},
			AskedFor: []string{"thingy"},
			Fulfills: true,
		},
		{
			Name:     "multiple unwanted things, single matching input value",
			Unwanted: []string{"unwanted", "also unwanted"},
			AskedFor: []string{"unwanted"},
			Fulfills: false,
		},
		{
			Name:     "multiple unwanted things, single unmatching input value",
			Unwanted: []string{"unwanted", "also unwanted"},
			AskedFor: []string{"something else"},
			Fulfills: true,
		},
		{
			Name:     "multiple unwanted things, multiple matching input value",
			Unwanted: []string{"unwanted", "also unwanted", "nope"},
			AskedFor: []string{"unwanted", "nope"},
			Fulfills: false,
		},
		{
			Name:     "multiple unwanted things, one matching input value",
			Unwanted: []string{"unwanted", "also unwanted", "nope"},
			AskedFor: []string{"unwanted", "another thing"},
			Fulfills: false,
		},
		{
			Name:     "multiple unwanted things, multiple unmatching input value",
			Unwanted: []string{"unwanted", "also unwanted", "nope"},
			AskedFor: []string{"a thing", "another thing"},
			Fulfills: true,
		},
		{
			Name:     "empty askedfor",
			Unwanted: []string{"unwanted", "also unwanted", "nope"},
			AskedFor: nil,
			Fulfills: true,
		},
		{
			Name:     "empty unwanted things",
			Unwanted: nil,
			AskedFor: []string{"a thing", "another thing"},
			Fulfills: true,
		},
		{
			Name:     "empty everything",
			Unwanted: nil,
			AskedFor: nil,
			Fulfills: true,
		},
		{
			Name:     "multiple unwanted things, simple string as an input",
			Unwanted: []string{"unwanted", "also unwanted"},
			AskedFor: "something else",
			Fulfills: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(fmt.Sprintf("%s-fulfills:%v", tc.Name, tc.Fulfills), func(t *testing.T) {
			condition := NotInSetCondition{Denied: tc.Unwanted}
			if result := condition.Fulfills(ctx, tc.AskedFor, nil); result != tc.Fulfills {
				t.Fatalf("fulfill result should be %v, got %v", tc.Fulfills, result)
			}
		})
	}
}

func TestInSetCondition(t *testing.T) {
	ctx := context.TODO()
	cases := []struct {
		Name     string
		Valid    []string
		Input    interface{}
		Fulfills bool
	}{
		{
			Name:     "single valid thing, single matching input value",
			Valid:    []string{"valid"},
			Input:    []string{"valid"},
			Fulfills: true,
		},
		{
			Name:     "single valid thing, single unmatching input value",
			Valid:    []string{"valid"},
			Input:    []string{"thingy"},
			Fulfills: false,
		},
		{
			Name:     "multiple valid things, single matching input value",
			Valid:    []string{"valid", "also valid"},
			Input:    []string{"valid"},
			Fulfills: true,
		},
		{
			Name:     "multiple valid things, single unmatching input value",
			Valid:    []string{"valid", "also valid"},
			Input:    []string{"something else"},
			Fulfills: false,
		},
		{
			Name:     "multiple valid things, multiple matching input value",
			Valid:    []string{"valid", "also valid", "ye"},
			Input:    []string{"valid", "ye"},
			Fulfills: true,
		},
		{
			Name:     "multiple valid things, one matching input value",
			Valid:    []string{"valid", "also valid", "nope"},
			Input:    []string{"valid", "another thing"},
			Fulfills: true,
		},
		{
			Name:     "multiple valid things, multiple unmatching input value",
			Valid:    []string{"valid", "also valid", "nope"},
			Input:    []string{"a thing", "another thing"},
			Fulfills: false,
		},
		{
			Name:     "empty input",
			Valid:    []string{"valid", "also valid"},
			Input:    []string{},
			Fulfills: false,
		},
		{
			Name:     "nil input",
			Valid:    []string{"valid", "also valid"},
			Input:    nil,
			Fulfills: false,
		},
		{
			Name:     "empty valid things",
			Valid:    nil,
			Input:    []string{"a thing", "another thing"},
			Fulfills: false,
		},
		{
			Name:     "empty everything",
			Valid:    nil,
			Input:    []string{},
			Fulfills: false,
		},
		{
			Name:     "empty valid, nil input",
			Valid:    nil,
			Input:    nil,
			Fulfills: false,
		},
		{
			Name:     "multiple valid things, simple string as input value",
			Valid:    []string{"valid", "also valid"},
			Input:    "valid",
			Fulfills: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(fmt.Sprintf("%s-fulfills:%v", tc.Name, tc.Fulfills), func(t *testing.T) {
			condition := InSetCondition{Valid: tc.Valid}
			if result := condition.Fulfills(ctx, tc.Input, nil); result != tc.Fulfills {
				t.Fatalf("fulfill result should be %v, got %v", tc.Fulfills, result)
			}
		})
	}
}

func TestLadonWardenHighConcurrency(t *testing.T) {
	ctx := context.TODO()
	l := NewLadon()
	err := l.LoadPoliciesFromJSONS("_policies", os.DirFS("./testdata"))
	if err != nil {
		t.Fatal(err)
	}

	const numGoroutines = 100
	const numRequestsPerGoroutine = 100

	requests := []*ladon.Request{
		{Action: "pass", Resource: "test", Subject: "this"},
		{Action: "pass", Resource: "test", Subject: "that"},
	}

	var wg sync.WaitGroup
	errChan := make(chan error, numGoroutines*numRequestsPerGoroutine)

	for i := range numGoroutines {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := range numRequestsPerGoroutine {
				req := requests[(goroutineID+j)%len(requests)]
				if err := l.IsAllowed(ctx, req); err != nil {
					errChan <- fmt.Errorf("goroutine %d, request %d: %w", goroutineID, j, err)
				}
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		t.Errorf("got %d errors during concurrent access, first error: %v", len(errors), errors[0])
	}
}
