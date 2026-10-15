package credentialcommands

import (
	"context"
	"errors"

	"lore-master/apps/lore-master-engine/credentialvault"
	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
)

// vaultError turns a vault failure into the protocol's: no keychain on this machine is
// "unsupported", which a caller can tell apart from a real failure and quietly skip.
func vaultError(err error) error {
	if errors.Is(err, credentialvault.ErrUnavailable) {
		return rpcprotocol.Errorf(rpcprotocol.CodeUnsupported, "%v", err)
	}

	return err
}

// StoreCredential handles credential/store.
func StoreCredential(vault credentialvault.Vault) rpcserver.Method {
	return func(_ context.Context, call rpcserver.Call) (any, error) {
		var params rpcprotocol.CredentialStoreParams
		if err := call.Decode(&params); err != nil {
			return nil, err
		}
		if params.Entry.BaseURL == "" {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodeInvalidParams, "a connection needs the site's address")
		}

		return nil, vaultError(vault.Store(params.Entry))
	}
}

// GetCredential handles credential/get.
func GetCredential(vault credentialvault.Vault) rpcserver.Method {
	return func(_ context.Context, call rpcserver.Call) (any, error) {
		var params rpcprotocol.CredentialGetParams
		if err := call.Decode(&params); err != nil {
			return nil, err
		}
		entry, found, err := vault.Get(params.BaseURL)
		if err != nil {
			return nil, vaultError(err)
		}
		if !found {
			return rpcprotocol.CredentialGetResult{}, nil
		}

		return rpcprotocol.CredentialGetResult{Found: true, Entry: &entry}, nil
	}
}

// DeleteCredential handles credential/delete.
func DeleteCredential(vault credentialvault.Vault) rpcserver.Method {
	return func(_ context.Context, call rpcserver.Call) (any, error) {
		var params rpcprotocol.CredentialDeleteParams
		if err := call.Decode(&params); err != nil {
			return nil, err
		}

		return nil, vaultError(vault.Delete(params.BaseURL))
	}
}

// ListCredentials handles credential/list.
func ListCredentials(vault credentialvault.Vault) rpcserver.Method {
	return func(_ context.Context, _ rpcserver.Call) (any, error) {
		entries, err := vault.List()
		if err != nil {
			return nil, vaultError(err)
		}
		if entries == nil {
			entries = []rpcprotocol.VaultEntry{}
		}

		return rpcprotocol.CredentialListResult{Entries: entries}, nil
	}
}
