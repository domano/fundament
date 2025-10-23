package fundament

import (
	"errors"
	"testing"

	"github.com/domano/fundament/internal/native"
)

func withAvailabilityHook(fn func() (native.Availability, error)) func() {
	prev := nativeCheckAvailability
	if fn != nil {
		nativeCheckAvailability = fn
	}
	return func() {
		nativeCheckAvailability = prev
	}
}

func TestAvailabilityString(t *testing.T) {
	ready := Availability{State: AvailabilityReady}
	if got := ready.String(); got != "available" {
		t.Fatalf("expected available, got %q", got)
	}
	unavailable := Availability{State: AvailabilityUnavailable, Reason: AvailabilityReasonModelNotReady}
	if got := unavailable.String(); got != "unavailable(3)" {
		t.Fatalf("unexpected string %q", got)
	}
	unknown := Availability{}
	if got := unknown.String(); got != "unknown" {
		t.Fatalf("unexpected string %q", got)
	}
}

func TestCheckAvailabilityMapping(t *testing.T) {
	tests := []struct {
		name    string
		stub    func() (native.Availability, error)
		want    Availability
		wantErr bool
	}{
		{
			name: "ready",
			stub: func() (native.Availability, error) {
				return native.Availability{State: 1, Reason: 0}, nil
			},
			want: Availability{State: AvailabilityReady, Reason: AvailabilityReasonNone},
		},
		{
			name: "unavailable reason mapping",
			stub: func() (native.Availability, error) {
				return native.Availability{State: 0, Reason: 2}, nil
			},
			want: Availability{State: AvailabilityUnavailable, Reason: AvailabilityReasonAppleIntelligenceDisabled},
		},
		{
			name: "error propagation",
			stub: func() (native.Availability, error) {
				return native.Availability{}, errors.New("boom")
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			restore := withAvailabilityHook(tc.stub)
			t.Cleanup(restore)

			got, err := CheckAvailability()
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("expected %+v, got %+v", tc.want, got)
			}
		})
	}
}
