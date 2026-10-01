package api_test

import (
	"strings"
	"testing"
)

func TestCreateProductRequestRequestDeviceFingerprint(t *testing.T) {
	cases := []struct {
		name        string
		fingerprint string
		wantErr     bool
	}{
		{"absent", "", false},
		{"valid digest", strings.Repeat("0f", 32), false},
		{"too short", strings.Repeat("0f", 31), true},
		{"upper case", strings.Repeat("0F", 32), true},
		{"not hex", strings.Repeat("xy", 32), true},
		{"trailing newline", strings.Repeat("0f", 32) + "\n", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := validRequest()
			r.DeviceFingerprint = tc.fingerprint
			err := r.Validate()
			if tc.wantErr && !isValidationError(err) {
				t.Fatalf("expected a validation error, got %v", err)
			} else if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
