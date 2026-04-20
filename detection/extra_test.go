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

func TestGenRequestExtraHasRspIfBlockNoResponse(t *testing.T) {
	dc := New()
	// dc.Response is nil — HasRspIfOK should be "u"
	extra := string(GenRequestExtra(dc))
	if !strings.Contains(extra, "HasRspIfOK:u\n") {
		t.Errorf("expected HasRspIfOK:u, got:\n%s", extra)
	}
	if !strings.Contains(extra, "HasRspIfBlock:n\n") {
		t.Errorf("expected HasRspIfBlock:n even without response, got:\n%s", extra)
	}
}

func TestMakeRequestExtraServerName(t *testing.T) {
	extra := string(MakeRequestExtra(
		"https", "go-sdk",
		"1.2.3.4", 54321,
		"127.0.0.1", 443,
		"example.com",
		"test-uuid", "n", "n",
		1000, 2000,
	))
	if !strings.Contains(extra, "ServerName:example.com\n") {
		t.Errorf("ServerName missing or wrong:\n%s", extra)
	}
	if !strings.Contains(extra, "ReqBeginTime:1000\n") {
		t.Errorf("ReqBeginTime missing:\n%s", extra)
	}
	if !strings.Contains(extra, "ReqEndTime:2000\n") {
		t.Errorf("ReqEndTime missing:\n%s", extra)
	}
}

func TestMakeResponseExtraServerName(t *testing.T) {
	extra := string(MakeResponseExtra(
		"https", "go-sdk",
		"1.2.3.4", 54321,
		"127.0.0.1", 443,
		"example.com",
		"test-uuid",
		3000, 4000,
	))
	if !strings.Contains(extra, "ServerName:example.com\n") {
		t.Errorf("ServerName missing or wrong:\n%s", extra)
	}
	if !strings.Contains(extra, "RspBeginTime:3000\n") {
		t.Errorf("RspBeginTime missing:\n%s", extra)
	}
	if !strings.Contains(extra, "RspEndTime:4000\n") {
		t.Errorf("RspEndTime missing:\n%s", extra)
	}
}

func TestGenRequestExtraServerName(t *testing.T) {
	dc := New()
	dc.ServerName = "my-sni.example.com"
	extra := string(GenRequestExtra(dc))
	if !strings.Contains(extra, "ServerName:my-sni.example.com\n") {
		t.Errorf("ServerName wrong:\n%s", extra)
	}
}
