package credentialverification

import (
	"context"
	"errors"
	"net/http"

	"lore-master/libs/confluence-client/httptransport"
)

// VerifyCredentials asks GET /rest/api/user/current, the same v1 endpoint on every
// edition, who the client's credential belongs to. A 401 or 403, or an anonymous
// answer, is an *UnauthorizedError; any other failure is returned as it is.
func VerifyCredentials(ctx context.Context, client *httptransport.Client) (User, error) {
	var user User
	err := client.GetJSON(ctx, "/rest/api/user/current", nil, &user)
	var apiError *httptransport.APIError
	if errors.As(err, &apiError) && (apiError.Status == http.StatusUnauthorized || apiError.Status == http.StatusForbidden) {
		return User{}, &UnauthorizedError{Status: apiError.Status}
	}
	if err != nil {
		return User{}, err
	}
	if user.Type == "anonymous" || (user.AccountID == "" && user.Username == "" && user.UserKey == "") {
		return User{}, &UnauthorizedError{Status: http.StatusOK}
	}

	return user, nil
}
