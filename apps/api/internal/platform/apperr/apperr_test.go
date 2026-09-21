package apperr

import (
	"errors"
	"fmt"
	"testing"
)

func TestAs_FindsWrappedError(t *testing.T) {
	base := NotFound("user_not_found", "not found")
	wrapped := fmt.Errorf("loading profile: %w", base)

	got, ok := As(wrapped)
	if !ok {
		t.Fatal("As() did not find the wrapped *Error")
	}
	if got != base {
		t.Errorf("As() returned %v, want the original %v", got, base)
	}
}

func TestAs_RejectsPlainError(t *testing.T) {
	_, ok := As(errors.New("boom"))
	if ok {
		t.Error("As() reported true for a plain error")
	}
}

func TestValidation_CarriesAllFields(t *testing.T) {
	err := Validation(map[string]string{"email": "invalid", "display_name": "too long"})

	if err.Kind != KindInvalid {
		t.Errorf("Kind = %v, want KindInvalid", err.Kind)
	}
	if err.Code != "validation_failed" {
		t.Errorf("Code = %q, want validation_failed", err.Code)
	}
	if len(err.Fields) != 2 {
		t.Errorf("Fields = %v, want 2 entries", err.Fields)
	}
}

func TestInternal_DoesNotLeakUnderlyingMessageInError(t *testing.T) {
	// Error() is used for logs, not client responses, but it must still
	// carry the underlying cause so logs are useful — httpx is the layer
	// responsible for keeping it out of the response body.
	cause := errors.New("connection refused")
	err := Internal(cause)

	if err.Message != "Something went wrong. Please try again." {
		t.Errorf("Message leaked internal detail: %q", err.Message)
	}
	if !errors.Is(err, cause) {
		t.Error("Internal() did not wrap the cause for Unwrap/errors.Is")
	}
}
