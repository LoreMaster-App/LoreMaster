package authentication

import (
	"encoding/base64"
	"errors"
	"fmt"
)

// AuthorizationHeader is the value of the Authorization header for credential: Basic
// for an API token (email:token) or a password, Bearer for a personal access token.
func AuthorizationHeader(credential Credential) (string, error) {
	switch c := credential.(type) {
	case APIToken:
		if c.Email == "" || c.Token == "" {
			return "", errors.New("an API token credential needs both the account email and the token")
		}

		return basic(c.Email, c.Token), nil
	case PAT:
		if c.Token == "" {
			return "", errors.New("the personal access token is empty")
		}

		return "Bearer " + c.Token, nil
	case Basic:
		if c.User == "" || c.Password == "" {
			return "", errors.New("a user name and password credential needs both")
		}

		return basic(c.User, c.Password), nil
	case nil:
		return "", errors.New("no credential was given")
	}

	return "", fmt.Errorf("unsupported credential kind %q", credential.Kind())
}

func basic(user string, secret string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+secret))
}
