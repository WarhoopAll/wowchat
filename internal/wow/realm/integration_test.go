package realm

import (
	"os"
	"testing"

	"github.com/WarhoopAll/wowchat/internal/config"
)

func TestRealmLoginIntegration(t *testing.T) {
	if os.Getenv("WOWCHAT_INTEGRATION") != "1" {
		t.Skip("set WOWCHAT_INTEGRATION=1 to run the authserver integration test")
	}
	cfg, err := config.Load(".env")
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	res, err := New(cfg, "enUS").Login()
	if err != nil {
		t.Fatalf("realm login: %v", err)
	}
	t.Logf("selected realm %s @ %s:%d (session key %d bytes)",
		res.RealmName, res.Host, res.Port, len(res.SessionKey))
	if len(res.SessionKey) != 40 {
		t.Errorf("session key length = %d, want 40", len(res.SessionKey))
	}
}
