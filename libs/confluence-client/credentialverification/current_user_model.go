package credentialverification

// User is the account a credential signs in as. Cloud identifies accounts by AccountID;
// Data Center and Server by Username (and UserKey).
type User struct {
	AccountID   string `json:"accountId"`
	Username    string `json:"username"`
	UserKey     string `json:"userKey"`
	DisplayName string `json:"displayName"`
	// Type is "known" for a signed-in user and "anonymous" when Cloud answered for a
	// request it did not authenticate.
	Type string `json:"type"`
}
