package detection

import (
	"io"
	"strings"
	"testing"
)

// stubResponse satisfies the Response interface with zero values.
type stubResponse struct{}

func (s *stubResponse) RequestHeader() ([]byte, error)       { return nil, nil }
func (s *stubResponse) Header() ([]byte, error)              { return nil, nil }
func (s *stubResponse) Body() (uint32, io.ReadCloser, error) { return 0, io.NopCloser(strings.NewReader("")), nil }
func (s *stubResponse) Extra() ([]byte, error)               { return nil, nil }
func (s *stubResponse) T1KContext() ([]byte, error)          { return nil, nil }

func TestGenRequestExtraHasRspIfBlock(t *testing.T) {
	dc := New()
	dc.Response = &stubResponse{} // non-nil so HasRspIfOK becomes 'y'
	extra := string(GenRequestExtra(dc))
	if !strings.Contains(extra, "HasRspIfOK:y\n") {
		t.Errorf("expected HasRspIfOK:y, got:\n%s", extra)
	}
	if !strings.Contains(extra, "HasRspIfBlock:n\n") {
		t.Errorf("expected HasRspIfBlock:n, got:\n%s", extra)
	}
}
