package fundament

import (
	"fmt"

	"github.com/domano/fundament/internal/native"
)

// AvailabilityState indicates whether the on-device model is ready to serve.
type AvailabilityState int

const (
	AvailabilityUnknown AvailabilityState = iota
	AvailabilityReady
	AvailabilityUnavailable
)

// AvailabilityReason encodes why the model is unavailable, mirroring SystemLanguageModel.Availability.Reason values.
type AvailabilityReason int

const (
	AvailabilityReasonNone AvailabilityReason = iota
	AvailabilityReasonDeviceNotEligible
	AvailabilityReasonAppleIntelligenceDisabled
	AvailabilityReasonModelNotReady
	AvailabilityReasonUnknown
)

// Availability reports the readiness of the language model.
type Availability struct {
	State  AvailabilityState
	Reason AvailabilityReason
}

func (a Availability) String() string {
	switch a.State {
	case AvailabilityReady:
		return "available"
	case AvailabilityUnavailable:
		return fmt.Sprintf("unavailable(%v)", a.Reason)
	default:
		return "unknown"
	}
}

// CheckAvailability queries the Swift shim for the current availability status.
func CheckAvailability() (Availability, error) {
	meta, err := native.CheckAvailability()
	if err != nil {
		return Availability{}, err
	}
	state := AvailabilityUnknown
	if meta.State == 1 {
		state = AvailabilityReady
	} else if meta.State == 0 {
		state = AvailabilityUnavailable
	}

	reason := AvailabilityReasonUnknown
	switch meta.Reason {
	case 0:
		reason = AvailabilityReasonNone
	case 1:
		reason = AvailabilityReasonDeviceNotEligible
	case 2:
		reason = AvailabilityReasonAppleIntelligenceDisabled
	case 3:
		reason = AvailabilityReasonModelNotReady
	case -1:
		reason = AvailabilityReasonUnknown
	}

	return Availability{
		State:  state,
		Reason: reason,
	}, nil
}
