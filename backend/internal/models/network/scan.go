package network

import (
	"context"
	"net"
	"sort"
	"strings"
	"sync"
	"time"
)

func (m *Manager) Scan(ctx context.Context) ([]Server, error) {
	m.scanMutex.Lock()
	if m.cancelScan != nil {
		m.cancelScan()
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	m.cancelScan = cancel
	m.scanMutex.Unlock()
	defer cancel()
	addresses, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}
	hosts := map[string]bool{}
	for _, address := range addresses {
		ip, subnet, err := net.ParseCIDR(address.String())
		if err != nil || ip.To4() == nil || !ip.IsPrivate() {
			continue
		}
		ip = ip.To4()
		for last := 1; last < 255; last++ {
			target := net.IPv4(ip[0], ip[1], ip[2], byte(last))
			if subnet.Contains(target) && !target.Equal(ip) {
				hosts[target.String()] = true
			}
		}
	}
	queue := make(chan string)
	found := []Server{}
	var mutex sync.Mutex
	var workers sync.WaitGroup
	for i := 0; i < 24; i++ {
		workers.Go(func() {
			for host := range queue {
				if ctx.Err() != nil {
					continue
				}
				connection, err := (&net.Dialer{Timeout: 400 * time.Millisecond}).DialContext(ctx, "tcp", net.JoinHostPort(host, "1234"))
				if err != nil {
					continue
				}
				connection.Close()
				server, err := discover(ctx, "http://"+net.JoinHostPort(host, "1234"), "")
				if err == nil {
					mutex.Lock()
					found = append(found, server)
					mutex.Unlock()
				}
			}
		})
	}
	for host := range hosts {
		select {
		case queue <- host:
		case <-ctx.Done():
		}
	}
	close(queue)
	workers.Wait()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	sort.Slice(found, func(i, j int) bool { return strings.Compare(found[i].URL, found[j].URL) < 0 })
	return found, nil
}
func (m *Manager) CancelScan() {
	m.scanMutex.Lock()
	defer m.scanMutex.Unlock()
	if m.cancelScan != nil {
		m.cancelScan()
	}
}
