package contract

import "testing"

func TestVisibilitySchedulingAndTimeoutPolicy(t *testing.T) {
	seen := map[string]bool{}
	for _, method := range Methods {
		if seen[method.Name] || method.Params == nil || method.Result == nil || method.Group == "" || method.Error == "" {
			t.Fatal("incomplete or duplicate method", method.Name)
		}
		seen[method.Name] = true
	}
	for _, name := range []string{"shutdown", "save_views", "media.resolve", "attachments.import", "attachments.read"} {
		m, ok := Lookup(name)
		if !ok || m.Renderer {
			t.Fatal("internal method exposed", name)
		}
	}
	for name, timeout := range map[string]int{"tools.run": 0, "chat.send": 0, "mcp.test": 35000, "network.state": 35000, "web.image": 20000, "web.icon": 10000, "local.state": 60000, "deepseek.refresh": 60000} {
		m, ok := Lookup(name)
		if !ok || !m.Renderer || m.Timeout != timeout {
			t.Fatal("timeout changed", name)
		}
	}
	for _, name := range []string{"tools.run", "network.scan", "generation.state", "mcp.save", "local.import"} {
		m, _ := Lookup(name)
		if !m.Async {
			t.Fatal("blocking dispatch changed", name)
		}
	}
	if _, ok := Lookup("unexpected.method"); ok {
		t.Fatal("unknown method registered")
	}
}
