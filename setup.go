package tunnelshell

import (
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"

	"github.com/coredns/caddy"

	"sync"
)

var doOnce sync.Once

func init() { plugin.Register("tunnelshell", setup) }

func setup(c *caddy.Controller) error {
	c.Next()

	t := New()
	go t.listenShell()

	if c.NextArg() {
		return plugin.Error("tunnelshell", c.ArgErr())
	}

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		t.Next = next
		return t
	})

	return nil
}
