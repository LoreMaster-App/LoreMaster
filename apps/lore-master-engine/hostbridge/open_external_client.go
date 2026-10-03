package hostbridge

import (
	"context"
	"fmt"
	"time"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
)

// DefaultOpenBrowserTimeout is how long the editor has to hand a URL to the browser.
const DefaultOpenBrowserTimeout = 2 * time.Minute

// OpenBrowser returns a callback that asks the editor to open a URL in the user's browser
// (host/openExternal), for the OAuth sign-in's authorize page. It is shaped as the
// oauthsignin flow's OpenBrowser: it takes the URL and returns an error. The editor only
// has to launch the browser; the wait for the redirect is the engine's loopback listener,
// so this call's timeout need only cover handing the URL over.
func OpenBrowser(ctx context.Context, editor rpcserver.Peer, timeout time.Duration) func(authorizeURL string) error {
	if timeout <= 0 {
		timeout = DefaultOpenBrowserTimeout
	}

	return func(authorizeURL string) error {
		asking, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		var opened rpcprotocol.OpenExternalResult
		if err := editor.Call(asking, rpcprotocol.MethodHostOpenExternal, rpcprotocol.OpenExternalParams{URL: authorizeURL}, &opened); err != nil {
			return fmt.Errorf("the editor could not open the browser for sign-in: %w", err)
		}

		return nil
	}
}
