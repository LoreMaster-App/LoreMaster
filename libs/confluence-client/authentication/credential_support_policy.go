package authentication

import (
	"fmt"

	"lore-master/libs/confluence-client/connection"
)

// personalAccessTokensSince is the first Data Center and Server release with PATs.
var personalAccessTokensSince = connection.Version{Major: 7, Minor: 9}

// Supports reports, as an error naming the remedy, why the edition at that version
// cannot accept the credential; nil when it can. Cloud takes only an email and API
// token. Data Center and Server take a personal access token from 7.9 on, a user name
// and password before it, and an OAuth access token (from 7.17's OAuth 2.0 provider).
// When the version is unknown, either of PAT and password is allowed: the server's
// answer is the judge. Cloud stays API-token only, so it refuses OAuth here too (see
// docs/architecture/oauth.md).
func Supports(edition connection.Edition, version connection.Version, credential Credential) error {
	switch edition {
	case connection.Cloud:
		if _, ok := credential.(APIToken); ok {
			return nil
		}

		return fmt.Errorf("a %s credential does not work on Confluence Cloud, which accepts only an Atlassian account email with an API token (create one at https://id.atlassian.com/manage-profile/security/api-tokens)", credential.Kind())
	case connection.DataCenter, connection.Server:
		switch credential.(type) {
		case PAT:
			if version.Known() && !version.AtLeast(personalAccessTokensSince.Major, personalAccessTokensSince.Minor) {
				return fmt.Errorf("personal access tokens need Confluence 7.9 or later and this site runs %s; sign in with a user name and password instead", version)
			}

			return nil
		case Basic:
			if version.Known() && version.AtLeast(personalAccessTokensSince.Major, personalAccessTokensSince.Minor) {
				return fmt.Errorf("this site runs Confluence %s, which supports personal access tokens; create one under Profile → Personal Access Tokens and use it instead of a password", version)
			}

			return nil
		case OAuth:
			return nil
		case APIToken:
			return fmt.Errorf("API tokens belong to Atlassian Cloud accounts; on Confluence %s use a personal access token", editionName(edition))
		}
	}

	return fmt.Errorf("unknown Confluence edition %q", edition)
}

func editionName(edition connection.Edition) string {
	if edition == connection.DataCenter {
		return "Data Center"
	}

	return "Server"
}
