package detection

import (
	"fmt"

	"github.com/chaitin/t1k-go/misc"
)

// MakeRequestExtra builds a newline-delimited byte payload containing request metadata.
// The payload contains labeled fields: Scheme, ProxyName, RemoteAddr, RemotePort, LocalAddr, LocalPort, ServerName, UUID, HasRspIfOK, HasRspIfBlock, ReqBeginTime, and ReqEndTime populated from the provided parameters.
func MakeRequestExtra(
	scheme string,
	proxyName string,
	remoteAddr string,
	remotePort uint16,
	localAddr string,
	localPort uint16,
	serverName string,
	uuid string,
	hasRspIfOK string,
	hasRspIfBlock string,
	reqBeginTime int64,
	reqEndTime int64,
) []byte {
	format := "Scheme:%s\n" +
		"ProxyName:%s\n" +
		"RemoteAddr:%s\n" +
		"RemotePort:%d\n" +
		"LocalAddr:%s\n" +
		"LocalPort:%d\n" +
		"ServerName:%s\n" +
		"UUID:%s\n" +
		"HasRspIfOK:%s\n" +
		"HasRspIfBlock:%s\n" +
		"ReqBeginTime:%d\n" +
		"ReqEndTime:%d\n"

	return fmt.Appendf(nil,
		format,
		scheme,
		proxyName,
		remoteAddr,
		remotePort,
		localAddr,
		localPort,
		serverName,
		uuid,
		hasRspIfOK,
		hasRspIfBlock,
		reqBeginTime,
		reqEndTime,
	)
}

// MakeResponseExtra builds a newline-delimited byte payload containing labeled response metadata.
// The payload includes Scheme, ProxyName, RemoteAddr, RemotePort, LocalAddr, LocalPort, ServerName,
// UUID, RspBeginTime, and RspEndTime in that order.
// It returns the formatted metadata as a []byte.
func MakeResponseExtra(
	scheme string,
	proxyName string,
	remoteAddr string,
	remotePort uint16,
	localAddr string,
	localPort uint16,
	serverName string,
	uuid string,
	rspBeginTime int64,
	rspEndTime int64,
) []byte {
	format := "Scheme:%s\n" +
		"ProxyName:%s\n" +
		"RemoteAddr:%s\n" +
		"RemotePort:%d\n" +
		"LocalAddr:%s\n" +
		"LocalPort:%d\n" +
		"ServerName:%s\n" +
		"UUID:%s\n" +
		"RspBeginTime:%d\n" +
		"RspEndTime:%d\n"

	return fmt.Appendf(nil,
		format,
		scheme,
		proxyName,
		remoteAddr,
		remotePort,
		localAddr,
		localPort,
		serverName,
		uuid,
		rspBeginTime,
		rspEndTime,
	)
}

// PlaceholderRequestExtra builds a request extra payload using hardcoded placeholder values suitable for testing or default cases.
// The provided uuid is inserted as the request UUID; the payload contains newline-delimited labeled metadata fields (scheme "http", proxy "go-sdk", loopback addresses/ports, empty ServerName, flags "n" for response indicators, and begin/end timestamps set to the current time) and is returned as a []byte.
func PlaceholderRequestExtra(uuid string) []byte {
	return MakeRequestExtra("http", "go-sdk", "127.0.0.1", 30001, "127.0.0.1", 80, "", uuid, "n", "n", misc.Now(), misc.Now())
}

// GenRequestExtra builds the request "extra" payload from the supplied DetectionContext.
// It derives the `HasRspIfOK` flag as "y" when dc.Response is present and "u" otherwise,
// and uses the current time when dc.ReqEndTime is zero. The `HasRspIfBlock` flag is set to "n".
// The returned []byte is a newline-delimited, labeled string containing scheme, proxy name,
// remote/local addresses and ports, server name, UUID, flags, and request begin/end times.
func GenRequestExtra(dc *DetectionContext) []byte {
	hasRsp := "u"
	if dc.Response != nil {
		hasRsp = "y"
	}
	reqEndTime := dc.ReqEndTime
	if reqEndTime == 0 {
		reqEndTime = misc.Now()
	}
	return MakeRequestExtra(dc.Scheme, dc.ProxyName, dc.RemoteAddr, dc.RemotePort, dc.LocalAddr, dc.LocalPort, dc.ServerName, dc.UUID, hasRsp, "n", dc.ReqBeginTime, reqEndTime)
}

// GenResponseExtra builds a response "extra" payload from the provided DetectionContext and returns it as a byte slice.
// It uses dc's scheme, proxy name, remote/local addresses and ports, server name, UUID, and response begin/end times.
// If dc.RspEndTime is zero, the current time from misc.Now() is used as the response end time.
// The returned slice contains newline-delimited, labeled response metadata suitable for downstream consumers.
func GenResponseExtra(dc *DetectionContext) []byte {
	rspEndTime := dc.RspEndTime
	if rspEndTime == 0 {
		rspEndTime = misc.Now()
	}
	return MakeResponseExtra(dc.Scheme, dc.ProxyName, dc.RemoteAddr, dc.RemotePort, dc.LocalAddr, dc.LocalPort, dc.ServerName, dc.UUID, dc.RspBeginTime, rspEndTime)
}
