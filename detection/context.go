package detection

import (
	"errors"
	"net"
	"net/http"

	"github.com/chaitin/t1k-go/misc"
)

type DetectionContext struct {
	UUID         string
	Scheme       string
	ProxyName    string
	RemoteAddr   string
	Protocol     string
	RemotePort   uint16
	LocalAddr    string
	LocalPort    uint16
	ServerName   string
	ReqBeginTime int64
	ReqEndTime   int64
	RspBeginTime int64
	RspEndTime   int64

	T1KContext []byte

	Request  Request
	Response Response
}

func New() *DetectionContext {
	return &DetectionContext{
		UUID:       misc.GenUUID(),
		Scheme:     "http",
		ProxyName:  "go-sdk",
		RemoteAddr: "127.0.0.1",
		RemotePort: 30001,
		LocalAddr:  "127.0.0.1",
		LocalPort:  80,
		Protocol:   "HTTP/1.1",
	}
}

// The function returns an error if req is nil or if obtaining the upstream address or port fails.
func MakeContextWithRequest(req *http.Request) (*DetectionContext, error) {
	if req == nil {
		return nil, errors.New("nil http.request or response")
	}
	wrapReq := &HttpRequest{
		req: req,
	}

	// ignore GetRemoteIP error,not sure request record remote ip
	remoteIP, _ := wrapReq.GetRemoteIP()
	remotePort, _ := wrapReq.GetRemotePort()

	localAddr, err := wrapReq.GetUpstreamAddress()
	if err != nil {
		return nil, err
	}

	localPort, err := wrapReq.GetUpstreamPort()
	if err != nil {
		return nil, err
	}

	scheme := "http"
	if req.TLS != nil {
		scheme = "https"
	}

	// derive ServerName: prefer TLS SNI, fall back to Host header (without port)
	serverName := req.Host
	if req.TLS != nil && req.TLS.ServerName != "" {
		serverName = req.TLS.ServerName
	}
	if host, _, err := net.SplitHostPort(serverName); err == nil {
		serverName = host
	}

	context := &DetectionContext{
		UUID:         misc.GenUUID(),
		Scheme:       scheme,
		ProxyName:    "go-sdk",
		RemoteAddr:   remoteIP,
		RemotePort:   remotePort,
		LocalAddr:    localAddr,
		LocalPort:    localPort,
		ServerName:   serverName,
		ReqBeginTime: misc.Now(),
		Request:      wrapReq,
		Protocol:     req.Proto,
	}
	wrapReq.dc = context
	return context, nil
}

func (dc *DetectionContext) ProcessResult(r *Result) {
	if r.Objective == RO_REQUEST {
		dc.T1KContext = r.T1KContext
	}
}
