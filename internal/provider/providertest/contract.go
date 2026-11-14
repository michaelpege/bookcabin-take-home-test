package providertest

import (
	"context"
	"testing"
	"time"

	"github.com/michaelpege/bookcabin-take-home-test/internal/domain"
)

// VerifyContract runs a shared behavioral contract suite against any domain.FlightProvider:
// 1. Name() is non-empty
// 2. Search() returns validated flights with non-nil Amenities
// 3. Search() respects context cancellation promptly
func VerifyContract(t *testing.T, p domain.FlightProvider, criteria domain.SearchCriteria) {
	t.Helper()

	if p.Name() == "" {
		t.Fatal("provider Name() must not be empty")
	}

	t.Run("valid_search", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		flights, err := p.Search(ctx, criteria)
		if err != nil {
			t.Fatalf("expected successful search, got error: %v", err)
		}
		if flights == nil {
			t.Fatal("expected non-nil flights slice")
		}
		for i, f := range flights {
			if err := f.Validate(); err != nil {
				t.Fatalf("flight[%d] (%s) failed domain validation: %v", i, f.ID, err)
			}
			if f.Amenities == nil {
				t.Fatalf("flight[%d] (%s) has nil Amenities (must be []string{} for JSON [])", i, f.ID)
			}
		}
	})

	t.Run("context_cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		done := make(chan error, 1)
		go func() {
			_, err := p.Search(ctx, criteria)
			done <- err
		}()

		select {
		case err := <-done:
			if err == nil {
				t.Fatal("expected error on cancelled context, got nil")
			}
		case <-time.After(500 * time.Millisecond):
			t.Fatal("provider did not abort promptly on cancelled context")
		}
	})
}
