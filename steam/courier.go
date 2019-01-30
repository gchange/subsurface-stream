package stream

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/gchange/subsurface-stream/dialer"
	"github.com/gchange/subsurface-stream/socks5"
	"net"
	"net/http"
	"os"
	"strconv"
)

type IPSegment struct {
	Start uint64
	End uint64
	ShortName string
	Name string
}

type IPList []IPSegment

type CourierConfig struct {
	Network string `subsurface:"network"`
	Address string `subsurface:"address"`
	IPv4 string `subsurface:"ipv4"`
	IPv6 string `subsurface:"ipv6"`
	Dialer map[string]interface{} `subsurface:"dialer"`
	ipList IPList
	localIP uint64
	localAddress string
	country string
	dialer dialer.Dialer
}

type Courier struct {
	*CourierConfig
}

func IPToUint64(ip net.IP) uint64 {
	if ipv4 := ip.To4(); ipv4 != nil {
		return  uint64(ipv4[0])<<24 + uint64(ipv4[1])<<16 + uint64(ipv4[2])<<8 + uint64(ipv4[3])
	} else if ipv6 := ip.To16(); ipv6 != nil {
		return uint64(ipv6[0])<<56+uint64(ipv6[1])<<48+uint64(ipv6[2])<<40+uint64(ipv6[3])<<32+uint64(ipv6[4])<<24+uint64(ipv6[5])<<16<<uint64(ipv6[6])<<8+uint64(ipv6[7])
	} else {
		return 0
	}
}

func (ipList IPList) find(n uint64) int {
	listLen := len(ipList)
	i, j, k := 0, listLen, listLen/2
	for i<j&&k<listLen {
		if ipList[k].Start > n {
			j = k
			k = (i+j)/2
		} else if ipList[k].End < n {
			i = k
			k = (i+j+1)/2
		} else {
			break
		}
	}
	return k
}

func (ipList IPList) Find(n uint64) IPSegment {
	index := ipList.find(n)
	if index == len(ipList) {
		return IPSegment{
			Start: n,
			End: n,
			ShortName: "-",
			Name:"-",
		}
	}
	return ipList[index]
}

func (ipList IPList) insert(start, end int, seg IPSegment) IPList {
	listLen := len(ipList)
		k := ipList.find(seg.Start)
		if k == listLen {
			return append(ipList, seg)
		}
		newList := append(ipList, ipList[listLen-1])
		for i:=listLen-1;i>k;i-- {
			ipList[i] = ipList[i-1]
		}
		newList[k] = seg
		return newList
}

func (ipList IPList) Insert(seg IPSegment) IPList {
	return ipList.insert(0, len(ipList), seg)
}

func (config *CourierConfig) IPUnmarshal(name string, list IPList) (IPList, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	reader := csv.NewReader(f)
	for {
		record, err := reader.Read()
		if err != nil {
			break
		}
		if len(record) != 4 {
			continue
		}
		start, err := strconv.ParseUint(record[0], 10, 64)
		if err != nil {
			continue
		}
		end, err := strconv.ParseUint(record[1], 10, 64)
		if err != nil {
			continue
		}
		seg := IPSegment{
			Start:start,
			End: end,
			ShortName: record[2],
			Name: record[3],
		}
		list = list.Insert(seg)
	}
	return list, nil
}

func (config *CourierConfig) Init() error {
	resp, err := http.Get("http://httpbin.org/ip")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	data := struct {
		Origin string `json:"origin"`
	}{}
	err = decoder.Decode(&data)
	if err != nil {
		return err
	}
	ip := net.ParseIP(data.Origin)
	config.localIP = IPToUint64(ip)
	config.localAddress = data.Origin


	if config.localAddress != "" {
		config.ipList = make(IPList, 0, 1024)
		if config.IPv4 != "" {
			config.ipList, err = config.IPUnmarshal(config.IPv4, config.ipList)
			if err != nil {
				return err
			}
		}
		if config.IPv6 != "" {
			config.ipList, err = config.IPUnmarshal(config.IPv6, config.ipList)
			if err != nil {
				return err
			}
		}
		if config.localIP != 0 {
			config.country = config.ipList.Find(uint64(config.localIP)).ShortName
		}
	}
	dialerConfig, err := dialer.GetDialerConfig(config.Dialer)
	if err != nil {
		return err
	}
	err = dialerConfig.Init()
	if err != nil {
		return err
	}
	config.dialer, err = dialerConfig.New()
	if err != nil {
		return err
	}
	return nil
}

func (config *CourierConfig) Clone() Config {
	return &CourierConfig{
		Network:config.Network,
		Address:config.Address,
		IPv4:config.IPv4,
		IPv6:config.IPv6,
		Dialer:config.Dialer,
		ipList: config.ipList,
		localIP: config.localIP,
		localAddress: config.localAddress,
		country : config.country,
		dialer : config.dialer,
	}
}

func (config *CourierConfig) Direct(conn net.Conn, remoteIP net.IP, remotePort uint16) (net.Conn, error) {
	address := fmt.Sprintf("%s:%d", remoteIP.String(), remotePort)
	remoteConn, err := config.dialer.Dial("tcp", address)
	if err != nil {
		return nil, err
	}
	err = socks5.EncodeBindAddress(conn, remoteConn.RemoteAddr().String())
	if err != nil {
		remoteConn.Close()
		conn.Close()
		return nil, err
	}
	go socks5.Copy(remoteConn, conn)
	go socks5.Copy(conn, remoteConn)
	return remoteConn, nil
}

func (config *CourierConfig) Proxy(conn net.Conn, remoteIP net.IP, remotePort uint16) (net.Conn, error) {
	proxyConn, err := config.dialer.Dial(config.Network, config.Address)
	if err != nil {
		return nil, err
	}
	bindIP, bindPort, err := socks5.Socks5Client(proxyConn, remoteIP, remotePort)
	if err != nil {
		return nil, err
	}
	err = socks5.EncodeIPAndPort(conn, bindIP, bindPort)
	if err != nil {
		proxyConn.Close()
		conn.Close()
		return nil, err
	}
	go socks5.Copy(proxyConn, conn)
	go socks5.Copy(conn, proxyConn)
	return proxyConn, nil
}

func (config *CourierConfig) New(conn net.Conn) (net.Conn, error) {
	remoteIP, remotePort, err := socks5.Decode(conn)
	if err != nil {
		return nil, err
	}
	if config.Address == "" {
		return config.Direct(conn, remoteIP, remotePort)
	}
	remoteUIP := IPToUint64(remoteIP)
	if remoteUIP == 0 {
		return config.Direct(conn, remoteIP, remotePort)
	}
	seg := config.ipList.Find(remoteUIP)
	if seg.ShortName == config.country {
		return config.Direct(conn, remoteIP, remotePort)
	}
	return config.Proxy(conn, remoteIP, remotePort)
}

func init() {
	config := &CourierConfig{
		Network: "tcp",
	}
	Register("courier", config)
}
