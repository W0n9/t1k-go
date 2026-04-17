package detection

import (
	"fmt"

	"github.com/chaitin/t1k-go/misc"
)

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

	return []byte(fmt.Sprintf(
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
	))
}

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

	return []byte(fmt.Sprintf(
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
	))
}

func PlaceholderRequestExtra(uuid string) []byte {
	return MakeRequestExtra("http", "go-sdk", "127.0.0.1", 30001, "127.0.0.1", 80, "", uuid, "n", "n", misc.Now(), misc.Now())
}

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

func GenResponseExtra(dc *DetectionContext) []byte {
	rspEndTime := dc.RspEndTime
	if rspEndTime == 0 {
		rspEndTime = misc.Now()
	}
	return MakeResponseExtra(dc.Scheme, dc.ProxyName, dc.RemoteAddr, dc.RemotePort, dc.LocalAddr, dc.LocalPort, dc.ServerName, dc.UUID, dc.RspBeginTime, rspEndTime)
}
