package util

import (
	"fmt"
	"net"
	"os"
	"strconv"
)

func FormatHost(hostPort string, lis net.Listener) (string, error) {
	addr, port, err := net.SplitHostPort(hostPort)
	if err != nil && lis == nil {
		return "", err
	}
	if lis != nil {
		if addr, ok := lis.Addr().(*net.TCPAddr); ok {
			port = strconv.Itoa(addr.Port)
		} else {
			return "", fmt.Errorf("failed to extract port: %v", lis.Addr())
		}
	}
	if len(addr) > 0 && (addr != "0.0.0.0" && addr != "[::]" && addr != "::") {
		return net.JoinHostPort(addr, port), nil
	}
	addr, err = getAddress()
	if err != nil {
		return "", err
	}
	return net.JoinHostPort(addr, port), nil
}

func getAddress() (string, error) {
	if os.Getenv("interface") != "" {
		if i, err := net.InterfaceByName(os.Getenv("interface")); err == nil {
			if ip, err := parseInterface(*i); err == nil && ip != "" {
				return ip, nil
			}
		}
	}
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		ip, err := parseInterface(iface)
		if err != nil || ip == "" {
			continue
		}
		return ip, nil
	}
	return "", nil
}

func parseInterface(i net.Interface) (string, error) {
	addrs, err := i.Addrs()
	if err != nil {
		return "", err
	}
	for _, rawAddr := range addrs {
		var ip net.IP
		switch addr := rawAddr.(type) {
		case *net.IPAddr:
			ip = addr.IP
		case *net.IPNet:
			ip = addr.IP
		default:
			continue
		}
		return ip.String(), nil
	}
	return "", nil
}
