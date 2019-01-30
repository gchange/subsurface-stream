package socks5

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/gchange/subsurface-stream/dialer"
	"github.com/sirupsen/logrus"
	"io"
	"net"
	"strconv"
	"strings"
)

func Copy(dst, src net.Conn) {
	defer dst.Close()
	for {
		if n, err := io.Copy(dst, src); err != nil || n == 0 {
			logrus.WithError(err).Debug("copy socks5 failed")
			break
		}
	}
}

func Socks5Client(conn net.Conn, ip net.IP, port uint16) (net.IP, uint16, error) {
	_, err := conn.Write([]byte{5, 1, 0})
	if err != nil {
		return nil, 0, err
	}
	buf := make([]uint8, 2, 2)
	_, err = io.ReadFull(conn, buf)
	if err != nil {
		return nil, 0, err
	}
	if buf[0] != 5 || buf[1] != 0 {
		return nil, 0, errors.New("unsupported protocol")
	}
	buf = []byte{5, 1, 0, 0}
	if ip := ip.To4();ip != nil {
		buf[3] = 1
		buf = append(buf, ip...)
	} else if ip := ip.To16(); ip != nil {
		buf[3] = 4
		buf = append(buf, ip...)
	} else {
		buf[3] = 3
		buf = append(buf, uint8(len(ip)))
		buf = append(buf, ip...)
	}
	_, err = conn.Write(buf)
	if err != nil {
		return nil, 0, err
	}
	err = binary.Write(conn, binary.BigEndian, &port)
	if err != nil {
		return nil, 0, err
	}
	buf = make([]uint8, 4, 4)
	_, err = io.ReadFull(conn, buf)
	if err != nil {
		return nil, 0, err
	}
	if buf[0] != 5 || buf[1] != 0 || buf[2] != 0 {
		return nil, 0, errors.New("connect failed")
	}
	var bindIP net.IP
	switch buf[3] {
	case 1:
		bindIP = make(net.IP, net.IPv4len)
	case 3:
		var bindLen uint8
		err = binary.Read(conn, binary.BigEndian, &bindLen)
		bindIP = make(net.IP, bindLen)
	case 4:
		bindIP = make(net.IP, net.IPv6len)
	}
	_, err = io.ReadFull(conn, bindIP)
	if err != nil {
		return nil, 0, err
	}
	var bindPort uint16
	err = binary.Read(conn, binary.BigEndian, &bindPort)
	if err != nil {
		return nil, 0, err
	}
	return bindIP, bindPort, nil
}

func Decode(conn net.Conn) (net.IP, uint16, error) {
	buf := make([]uint8, 2, 2)
	_, err := io.ReadFull(conn, buf)
	if err != nil {
		return nil, 0, err
	}
	if buf[0] != 5 {
		return nil, 0, errors.New("unsupported protocol")
	}
	if buf[1] == 0 {
		return nil, 0, errors.New("missing verify method")
	}
	buf = make([]uint8, buf[1], buf[1])
	_, err = io.ReadFull(conn, buf)
	if err != nil {
		return nil, 0, err
	}
	flag := false
	for _, n := range buf {
		if n == 0 {
			flag = true
			break
		}
	}
	if !flag {
		return nil, 0, errors.New("unsupported protocol")
	}
	_, err = conn.Write([]byte{5, 0})
	if err != nil {
		return nil, 0, err
	}
	buf = make([]byte, 4, 4)
	_, err = io.ReadFull(conn, buf)
	if err != nil {return nil, 0, err}
	if buf[0] != 5 || buf[1] != 1 || buf[2] != 0 {
		return nil, 0, errors.New("unsupported protocol")
	}
	var ip net.IP
	switch buf[3] {
	case 1:
		ip = make(net.IP, net.IPv4len)
	case 3:
		var ipLen uint8
		err = binary.Read(conn, binary.BigEndian, &ipLen)
		if err != nil {
			return nil, 0, err
		}
		ip = make(net.IP, ipLen)
	case 4:
		ip = make(net.IP, net.IPv6len)
	default:
		return nil, 0, errors.New("unsupported protocol")
	}
	_, err = io.ReadFull(conn, ip)
	if err != nil {
		return nil, 0, err
	}
	var port uint16
	err = binary.Read(conn, binary.BigEndian, &port)
	if err != nil {
		return nil, 0, err
	}
	return ip, port, nil
}

func EncodeBindAddress(conn net.Conn, address string) error {
	var ip net.IP
	var port int
	var err error
	index := strings.LastIndex(address, ":")
	if index > 0 {
		ip = net.ParseIP(address[:index])
		port, err = strconv.Atoi(address[index+1:])
		if err != nil {
			return err
		}
	} else {
		ip = []byte(address)
		port = 0
	}
	return EncodeIPAndPort(conn, ip, uint16(port))
}

func EncodeIPAndPort(conn net.Conn, ip net.IP, port uint16) error {
	buf := []byte{5, 0, 0, 0}
	if ip := ip.To4(); ip != nil {
		buf[3] = 1
		buf = append(buf, ip...)
	} else if ip := ip.To16(); ip != nil {
		buf[3] = 4
		buf = append(buf, ip...)
	} else {
		buf[3] = 3
		buf = append(buf, uint8(len(ip)))
		buf = append(buf, ip...)
	}
	_, err := conn.Write(buf)
	if  err != nil {
		return err
	}
	err = binary.Write(conn, binary.BigEndian, &port)
	if err != nil {
		return err
	}
	return nil
}

func Socks5Proxy(conn net.Conn, dialer dialer.Dialer, network, address string) (net.Conn, error) {
	client, err := dialer.Dial(network, address)
	if err != nil {
		return nil, err
	}
	remoteIP, remotePort, err := Decode(conn)
	if err != nil {
		return nil, err
	}
	bindIP, bindPort, err := Socks5Client(client, remoteIP, remotePort)
	if err != nil {
		return nil, err
	}
	err = EncodeIPAndPort(conn, bindIP, bindPort)
	if err != nil {
		return nil, err
	}
	go Copy(client, conn)
	go Copy(conn, client)
	return conn, nil
}

func Socks5Server(conn net.Conn, dialer dialer.Dialer) (net.Conn, error) {
	remoteIP, remotePort, err := Decode(conn)
	if err != nil {
		return nil, err
	}

	address := fmt.Sprintf("%s:%d", remoteIP.String(), remotePort)
	remoteConn, err := dialer.Dial("tcp", address)
	if err != nil {
		conn.Write([]byte{5, 1, 0})
		return nil, err
	}
	err = EncodeBindAddress(conn, remoteConn.RemoteAddr().String())
	if err != nil {
		return nil, err
	}
	go Copy(conn, remoteConn)
	go Copy(remoteConn, conn)
	return conn, nil
}