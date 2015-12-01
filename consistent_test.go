package ssf

import (
	"testing"
	//"fmt"
)

func TestConsitent(t *testing.T) {
	c := NewConsistent(121)
	servers := []string{"127.0.0.1:7788", "127.0.0.1:7789", "127.0.0.1:7790", "127.0.0.1:7791"}
	for _, server := range servers {
		c.Add(server)
	}
	for _, server := range servers {
		ns := c.Get(server)
		t.Logf("%s -> (%d)%v", server, len(ns), ns)
	}
	t.Logf("##After add %d servers", len(servers))
	removed := servers[0]
	c.Remove(removed)
	servers = servers[1:]
	for _, server := range servers {
		ns := c.Get(server)
		t.Logf("%s -> (%d)%v", server, len(ns), ns)
	}
	t.Logf("##After remove 1 server")
	c.Add(removed)
	servers = append([]string{removed}, servers...)
	for _, server := range servers {
		ns := c.Get(server)
		t.Logf("%s -> (%d)%v", server, len(ns), ns)
	}
	t.Logf("##After resume 1 server")
	c.Remove(removed)
	c.Add(removed)
	c.Remove(removed)
	c.Add(removed)
	t.Logf("%v", c.Servers())

}
