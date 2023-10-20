package util

import (
	"fmt"
	"net"
	"strings"

	"github.com/cakturk/go-netstat/netstat"
	"github.com/doki/common/model"
)

func GetIP(prefixs ...string) (string, error) {
	var prefix1, prefix2 string
	if len(prefixs) == 0 || prefixs[0] == "" {
		prefix1 = "10.59"
		prefix2 = "10.5"
	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	for _, addr := range addrs {
		ip := strings.Split(addr.String(), "/")[0]
		if strings.HasPrefix(ip, prefix1) {
			return ip, nil
		}
		if strings.HasPrefix(ip, prefix2) {
			return ip, nil
		}
	}
	return "", fmt.Errorf("IP NOT FOUND")
}

func GetLocalServices(services []string, prefix ...string) ([]*model.Process, error) {
	ip, err := GetIP(prefix...)
	if err != nil {
		return nil, err
	}
	filter := func(s *netstat.SockTabEntry) bool {
		if s.State.String() == "LISTEN" {
			return true
		}
		return false
	}
	socks, err := netstat.TCPSocks(filter)
	if err != nil {
		return nil, err
	}
	socks6, err := netstat.TCP6Socks(filter)
	if err == nil {
		socks = append(socks, socks6...)
	}
	var procs []*model.Process
	for _, v := range socks {
		if v.Process == nil {
			continue
		}
		name := v.Process.Name
		flag := Contains(services, name)
		if !flag {
			continue
		}
		proc := &model.Process{
			Port: int(v.LocalAddr.Port),
			Name: name,
			Ip:   ip,
			Pid:  v.Process.Pid,
		}
		procs = append(procs, proc)
	}
	if len(procs) == 0 {
		return nil, fmt.Errorf("not found any instance for services: %v", services)
	}
	return procs, nil
}

func Contains(ss []string, s string) bool {
	for _, v := range ss {
		if v != s {
			continue
		}
		return true
	}
	return false
}
