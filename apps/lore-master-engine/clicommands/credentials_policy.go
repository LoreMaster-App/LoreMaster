package clicommands

import (
	"errors"
	"strings"

	"lore-master/apps/lore-master-engine/rpcprotocol"
)

// The environment variables a Confluence credential is read from. Secrets never go in
// .lore-master.yaml; in CI they are the pipeline's secrets.
const (
	envEmail    = "LORE_MASTER_EMAIL"
	envToken    = "LORE_MASTER_TOKEN"
	envPAT      = "LORE_MASTER_PAT"
	envUser     = "LORE_MASTER_USER"
	envPassword = "LORE_MASTER_PASSWORD"
)

// errNoCredential tells the user which variables would have worked.
var errNoCredential = errors.New("no Confluence credential in the environment: set " + envEmail + " and " + envToken +
	" (Cloud API token), or " + envPAT + " (Data Center or Server personal access token), or " + envUser + " and " + envPassword + " (basic)")

// credentialFromEnvironment picks the credential the environment holds: an email with an API
// token (Confluence Cloud), else a personal access token (Data Center, Server 7.9+), else a
// user and password (older Server). The same credential is used for every Confluence output.
func credentialFromEnvironment(getenv func(string) string) (rpcprotocol.Credential, error) {
	value := func(name string) string { return strings.TrimSpace(getenv(name)) }

	switch {
	case value(envEmail) != "" && value(envToken) != "":
		return rpcprotocol.Credential{Kind: "apitoken", Email: value(envEmail), Token: value(envToken)}, nil
	case value(envPAT) != "":
		return rpcprotocol.Credential{Kind: "pat", Token: value(envPAT)}, nil
	case value(envUser) != "" && value(envPassword) != "":
		return rpcprotocol.Credential{Kind: "basic", User: value(envUser), Password: value(envPassword)}, nil
	}

	return rpcprotocol.Credential{}, errNoCredential
}
