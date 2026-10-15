package clicommands

import (
	"strings"
	"testing"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
)

func vaultDouble(remembered *rpcprotocol.VaultEntry) *engineDouble {
	answers := map[string]func(rpcserver.Call) (any, error){
		rpcprotocol.MethodSessionOpen: func(rpcserver.Call) (any, error) {
			return rpcprotocol.SessionOpenResult{
				SessionID: "s1", BaseURL: "https://acme.atlassian.net/wiki", Edition: "cloud",
				User: rpcprotocol.SessionUser{DisplayName: "Ada Lovelace", AccountID: "acc-1"},
			}, nil
		},
		rpcprotocol.MethodSessionClose:     func(rpcserver.Call) (any, error) { return nil, nil },
		rpcprotocol.MethodCredentialStore:  func(rpcserver.Call) (any, error) { return nil, nil },
		rpcprotocol.MethodCredentialDelete: func(rpcserver.Call) (any, error) { return nil, nil },
		rpcprotocol.MethodCredentialGet: func(rpcserver.Call) (any, error) {
			if remembered == nil {
				return rpcprotocol.CredentialGetResult{}, nil
			}

			return rpcprotocol.CredentialGetResult{Found: true, Entry: remembered}, nil
		},
		rpcprotocol.MethodCredentialList: func(rpcserver.Call) (any, error) {
			if remembered == nil {
				return rpcprotocol.CredentialListResult{}, nil
			}

			return rpcprotocol.CredentialListResult{Entries: []rpcprotocol.VaultEntry{*remembered}}, nil
		},
		rpcprotocol.MethodSettingsRead: settingsAnswer(confluenceOutput),
		rpcprotocol.MethodSyncPlan: func(rpcserver.Call) (any, error) {
			return rpcprotocol.SyncPlanResult{PlanID: "p", Counts: map[string]int{"unchanged": 1}}, nil
		},
		rpcprotocol.MethodSyncExecute:   func(rpcserver.Call) (any, error) { return rpcprotocol.SyncExecuteResult{}, nil },
		rpcprotocol.MethodGeneratorsRun: func(rpcserver.Call) (any, error) { return rpcprotocol.GeneratorsRunResult{}, nil },
	}

	return newEngineDouble(answers)
}

func TestAuthLoginChecksTheCredentialThenRemembersIt(t *testing.T) {
	double := vaultDouble(nil)

	stdout, _, code := double.run(t, cloudCredentials, "auth", "login", "--base-url", "https://acme.atlassian.net/wiki")

	if code != ExitOK || !strings.Contains(stdout, "remembered the connection to https://acme.atlassian.net/wiki as Ada Lovelace") {
		t.Fatalf("exit %d: %s", code, stdout)
	}
	var stored rpcprotocol.CredentialStoreParams
	double.paramsOf(t, rpcprotocol.MethodCredentialStore, &stored)
	if stored.Entry.Credential.Email != "me@acme.com" || stored.Entry.Credential.Token != "secret" || stored.Entry.User != "acc-1" || stored.Entry.Edition != "cloud" {
		t.Fatalf("stored %+v", stored.Entry)
	}
	if !double.called(rpcprotocol.MethodSessionClose) {
		t.Fatal("the session was left open")
	}
}

func TestAuthLoginWithoutACredentialRemembersNothing(t *testing.T) {
	double := vaultDouble(nil)

	_, stderr, code := double.run(t, nil, "auth", "login", "--base-url", "https://acme.atlassian.net/wiki")

	if code != ExitFailed || !strings.Contains(stderr, "no Confluence credential") || double.called(rpcprotocol.MethodCredentialStore) {
		t.Fatalf("exit %d: %s", code, stderr)
	}
}

func TestAuthStatusListsWhatIsRemembered(t *testing.T) {
	double := vaultDouble(&rpcprotocol.VaultEntry{BaseURL: "https://acme.atlassian.net/wiki", Edition: "cloud", DisplayName: "Ada", Credential: rpcprotocol.Credential{Kind: "apitoken", Token: "secret"}})

	stdout, _, code := double.run(t, nil, "auth", "status")

	if code != ExitOK || !strings.Contains(stdout, "https://acme.atlassian.net/wiki  cloud  signs in as Ada (apitoken)") {
		t.Fatalf("exit %d: %s", code, stdout)
	}
	if strings.Contains(stdout, "secret") {
		t.Fatal("status printed the token")
	}
}

func TestAuthLogoutForgetsTheSite(t *testing.T) {
	double := vaultDouble(nil)

	_, _, code := double.run(t, nil, "auth", "logout", "--base-url", "https://acme.atlassian.net/wiki")

	if code != ExitOK || !double.called(rpcprotocol.MethodCredentialDelete) {
		t.Fatalf("exit %d", code)
	}
}

func TestSyncSignsInWithTheRememberedCredentialWhenTheEnvironmentHasNone(t *testing.T) {
	double := vaultDouble(&rpcprotocol.VaultEntry{BaseURL: "https://acme.atlassian.net/wiki", Credential: rpcprotocol.Credential{Kind: "pat", Token: "from-keychain"}})

	_, _, code := double.run(t, nil, "sync")

	if code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	var open rpcprotocol.SessionOpenParams
	double.paramsOf(t, rpcprotocol.MethodSessionOpen, &open)
	if open.Credential.Token != "from-keychain" {
		t.Fatalf("signed in with %+v", open.Credential)
	}
}

func TestTheEnvironmentWinsOverTheKeychain(t *testing.T) {
	double := vaultDouble(&rpcprotocol.VaultEntry{BaseURL: "https://acme.atlassian.net/wiki", Credential: rpcprotocol.Credential{Kind: "pat", Token: "from-keychain"}})

	_, _, code := double.run(t, cloudCredentials, "sync")

	if code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	var open rpcprotocol.SessionOpenParams
	double.paramsOf(t, rpcprotocol.MethodSessionOpen, &open)
	if open.Credential.Token != "secret" || double.called(rpcprotocol.MethodCredentialGet) {
		t.Fatalf("signed in with %+v", open.Credential)
	}
}

func TestSyncWithNeitherSaysHowToRememberOne(t *testing.T) {
	double := vaultDouble(nil)

	_, stderr, code := double.run(t, nil, "sync")

	if code != ExitFailed || !strings.Contains(stderr, "auth login --base-url") {
		t.Fatalf("exit %d: %s", code, stderr)
	}
}
