package audit

import (
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-aurora/internal/platform/logging"
)

func TestFromRequest_PopulatesIPAndCorrelation(t *testing.T) {
	corr := uuid.NewString()
	req := httptest.NewRequest("PATCH", "/api/v1/admin/feature-flags/x", nil)
	req.RemoteAddr = "203.0.113.7:54812"
	req = req.WithContext(logging.WithCorrelationID(req.Context(), corr))

	e := FromRequest(req)

	if e.IPAddress != "203.0.113.7:54812" {
		t.Errorf("IPAddress = %q, want the raw RemoteAddr (sanitizeIP strips the port at Record time)", e.IPAddress)
	}
	if e.CorrelationID == nil || e.CorrelationID.String() != corr {
		t.Errorf("CorrelationID = %v, want %s", e.CorrelationID, corr)
	}
}

func TestFromRequest_NonUUIDCorrelationIsDropped(t *testing.T) {
	req := httptest.NewRequest("PATCH", "/x", nil)
	req = req.WithContext(logging.WithCorrelationID(req.Context(), "client-supplied-not-a-uuid"))

	e := FromRequest(req)

	if e.CorrelationID != nil {
		t.Errorf("CorrelationID = %v, want nil for a non-UUID X-Request-ID", e.CorrelationID)
	}
}

func TestFromRequest_NoCorrelationInContext(t *testing.T) {
	req := httptest.NewRequest("PATCH", "/x", nil)
	req.RemoteAddr = "198.51.100.4:1010"

	e := FromRequest(req)

	if e.CorrelationID != nil {
		t.Errorf("CorrelationID = %v, want nil", e.CorrelationID)
	}
	if e.IPAddress != "198.51.100.4:1010" {
		t.Errorf("IPAddress = %q", e.IPAddress)
	}
}
