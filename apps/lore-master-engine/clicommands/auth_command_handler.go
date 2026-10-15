package clicommands

import (
	"context"
	"flag"
	"fmt"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
)

// authReport is what --json prints.
type authReport struct {
	Remembered []authEntry `json:"remembered,omitempty"`
	Action     string      `json:"action,omitempty"`
	BaseURL    string      `json:"baseUrl,omitempty"`
}

type authEntry struct {
	BaseURL string `json:"baseUrl"`
	Edition string `json:"edition,omitempty"`
	User    string `json:"user,omitempty"`
	Kind    string `json:"kind"`
}

const authUsage = "usage: lore-master-engine auth login --base-url URL   (the credential comes from the LORE_MASTER_* variables)\n" +
	"       lore-master-engine auth logout --base-url URL\n" +
	"       lore-master-engine auth status\n"

// authCommand remembers a Confluence connection in the operating system's keychain, so the
// command line (and the extension, when it is set to use the keychain) signs in without the
// token in every shell's environment.
func authCommand(ctx context.Context, env Environment, methods rpcserver.Methods, args []string) int {
	if len(args) == 0 || (args[0] != "login" && args[0] != "logout" && args[0] != "status") {
		_, _ = fmt.Fprint(env.Stderr, authUsage)

		return ExitUsage
	}
	action := args[0]
	var baseURL string
	parsed, code, ok := parseFlags("auth "+action, env, args[1:], func(flags *flag.FlagSet) {
		flags.StringVar(&baseURL, "base-url", "", "the Confluence site's address")
	})
	if !ok {
		return code
	}
	if action != "status" && baseURL == "" {
		_, _ = fmt.Fprintf(env.Stderr, "auth %s needs --base-url\n", action)

		return ExitUsage
	}

	return withEngine(ctx, env, methods, func(engine *connection) int {
		report := &authReport{Action: action, BaseURL: baseURL}
		result := runAuth(ctx, env, engine, action, baseURL, report, parsed.json)
		if parsed.json {
			printJSON(env.Stdout, report)
		}

		return result
	})
}

func runAuth(ctx context.Context, env Environment, engine *connection, action string, baseURL string, report *authReport, asJSON bool) int {
	say := func(format string, args ...any) {
		if !asJSON {
			_, _ = fmt.Fprintf(env.Stdout, format+"\n", args...)
		}
	}
	fail := func(err error) int {
		_, _ = fmt.Fprintln(env.Stderr, "error:", err)

		return ExitFailed
	}

	switch action {
	case "login":
		credential, err := credentialFromEnvironment(env.Getenv)
		if err != nil {
			return fail(err)
		}
		var session rpcprotocol.SessionOpenResult
		if err := engine.call(ctx, rpcprotocol.MethodSessionOpen, rpcprotocol.SessionOpenParams{BaseURL: baseURL, Credential: credential}, &session); err != nil {
			return fail(fmt.Errorf("signing in to %s: %w", baseURL, err))
		}
		defer func() {
			_ = engine.call(ctx, rpcprotocol.MethodSessionClose, rpcprotocol.SessionCloseParams{SessionID: session.SessionID}, nil)
		}()
		user := session.User.Username
		if session.User.AccountID != "" {
			user = session.User.AccountID
		}
		entry := rpcprotocol.VaultEntry{BaseURL: session.BaseURL, Edition: session.Edition, DisplayName: session.User.DisplayName, User: user, Credential: credential}
		if err := engine.call(ctx, rpcprotocol.MethodCredentialStore, rpcprotocol.CredentialStoreParams{Entry: entry}, nil); err != nil {
			return fail(err)
		}
		report.BaseURL = session.BaseURL
		say("remembered the connection to %s as %s in this machine's keychain", session.BaseURL, session.User.DisplayName)
	case "logout":
		if err := engine.call(ctx, rpcprotocol.MethodCredentialDelete, rpcprotocol.CredentialDeleteParams{BaseURL: baseURL}, nil); err != nil {
			return fail(err)
		}
		say("forgot the connection to %s", baseURL)
	case "status":
		var list rpcprotocol.CredentialListResult
		if err := engine.call(ctx, rpcprotocol.MethodCredentialList, struct{}{}, &list); err != nil {
			return fail(err)
		}
		if len(list.Entries) == 0 {
			say("no connection is remembered in this machine's keychain")
		}
		for _, entry := range list.Entries {
			report.Remembered = append(report.Remembered, authEntry{BaseURL: entry.BaseURL, Edition: entry.Edition, User: entry.User, Kind: entry.Credential.Kind})
			say("%s  %s  signs in as %s (%s)", entry.BaseURL, entry.Edition, entry.DisplayName, entry.Credential.Kind)
		}
	}

	return ExitOK
}

// credentialFor is the credential to sign in to baseURL with: the environment's, else the one
// remembered in the keychain with `auth login`. The environment wins so a pipeline's secret is
// never shadowed by whatever a developer's machine remembered.
func credentialFor(ctx context.Context, env Environment, engine *connection, baseURL string) (rpcprotocol.Credential, error) {
	credential, err := credentialFromEnvironment(env.Getenv)
	if err == nil {
		return credential, nil
	}
	var found rpcprotocol.CredentialGetResult
	if callErr := engine.call(ctx, rpcprotocol.MethodCredentialGet, rpcprotocol.CredentialGetParams{BaseURL: baseURL}, &found); callErr == nil && found.Found && found.Entry != nil {
		return found.Entry.Credential, nil
	}

	return rpcprotocol.Credential{}, fmt.Errorf("%w; or remember one with `lore-master-engine auth login --base-url %s`", err, baseURL)
}
