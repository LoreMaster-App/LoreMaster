package credentialvault

import (
	"testing"

	"lore-master/apps/lore-master-engine/rpcprotocol"
)

func TestKeyIgnoresCaseAndTrailingSlashes(t *testing.T) {
	if Key(" https://Acme.Atlassian.net/wiki// ") != "https://acme.atlassian.net/wiki" {
		t.Fatalf("key = %q", Key(" https://Acme.Atlassian.net/wiki// "))
	}
}

func TestInMemoryVaultStoresFindsListsAndForgets(t *testing.T) {
	vault := NewInMemory()
	entry := Entry{BaseURL: "https://acme.atlassian.net/wiki", Edition: "cloud", Credential: rpcprotocol.Credential{Kind: "pat", Token: "t"}}

	if err := vault.Store(entry); err != nil {
		t.Fatal(err)
	}
	found, ok, err := vault.Get("https://ACME.atlassian.net/wiki/")
	if err != nil || !ok || found.Credential.Token != "t" {
		t.Fatalf("get: %+v %v %v", found, ok, err)
	}
	listed, _ := vault.List()
	if len(listed) != 1 {
		t.Fatalf("listed %+v", listed)
	}
	if err := vault.Delete(entry.BaseURL); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := vault.Get(entry.BaseURL); ok {
		t.Fatal("a forgotten connection is still there")
	}
	if err := vault.Delete("https://never.example"); err != nil {
		t.Fatalf("forgetting an unknown site is not an error: %v", err)
	}
}
