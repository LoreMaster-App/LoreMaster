package clicommands

import (
	"errors"
	"strings"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/libs/documentation-sync/configcatalog"
)

// The environment variables a credential is read from are the ones the platform's description names: each field of each sign-in
// method says its own. Secrets never go in the settings; in CI they are the pipeline's secrets.

// signIn is the sign-in the command line uses: the first platform that signs in (Confluence today), and what it is called.
func signIn() (string, configcatalog.Connection) {
	for _, platform := range configcatalog.Platforms() {
		if platform.Connection != nil {
			return platform.Label, *platform.Connection
		}
	}

	return "", configcatalog.Connection{}
}

// errNoCredential tells the user which variables would have worked.
func errNoCredential() error {
	label, connection := signIn()
	var ways []string
	for _, kind := range connection.Kinds {
		var names []string
		for _, field := range kind.Fields {
			names = append(names, field.Env)
		}
		ways = append(ways, strings.Join(names, " and ")+" ("+kind.Label+")")
	}

	return errors.New("no " + label + " credential in the environment: set " + strings.Join(ways, ", or "))
}

// credentialFromEnvironment picks the credential the environment holds: the first sign-in method, in the order the description
// lists them, whose variables are all set (an email with an API token, else a personal access token, else a user and password).
// The same credential is used for every output that signs in.
func credentialFromEnvironment(getenv func(string) string) (rpcprotocol.Credential, error) {
	_, connection := signIn()
	for _, kind := range connection.Kinds {
		values := map[string]string{}
		complete := true
		for _, field := range kind.Fields {
			values[field.Key] = strings.TrimSpace(getenv(field.Env))
			complete = complete && values[field.Key] != ""
		}
		if complete {
			return rpcprotocol.Credential{Kind: kind.ID, Email: values["email"], Token: values["token"], User: values["user"], Password: values["password"]}, nil
		}
	}

	return rpcprotocol.Credential{}, errNoCredential()
}
