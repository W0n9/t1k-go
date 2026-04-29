package t1k

import (
	"context"
	"errors"
	"net"
	"time"
)

// ConnectionFactory 连接工厂
type ConnectionFactory interface {
	//生成连接的方法
	Factory() (any, error)
	//关闭连接的方法
	Close(any) error
	//检查连接是否有效的方法
	Ping(any) error
}

// TcpFactory 结构体
type TcpFactory struct {
	Addr string
}

// Factory 方法生成 TCP 连接
func (t *TcpFactory) Factory() (any, error) {
	d := net.Dialer{Timeout: 3 * time.Second}
	conn, err := d.DialContext(context.Background(), "tcp", t.Addr)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// Close 方法关闭 TCP 连接
func (t *TcpFactory) Close(conn any) error {
	tcpConn, ok := conn.(net.Conn)
	if !ok {
		return errors.New("invalid connection type")
	}
	return tcpConn.Close()
}

// Ping 方法检查 TCP 连接是否有效
func (f *TcpFactory) Ping(conn any) error {
	tcpConn, ok := conn.(net.Conn)
	if !ok {
		return errors.New("invalid connection type")
	}
	// // 发送一个空的 TCP 数据包来检查连接是否有效
	// if err := tcpConn.SetDeadline(time.Now().Add(1 * time.Second)); err != nil {
	// 	return err
	// }
	// if _, err := tcpConn.Write([]byte{}); err != nil {
	// 	return err
	// }
	// return tcpConn.SetDeadline(time.Time{})
	err := DoHeartbeat(tcpConn)
	return err
}
