package middleware

import "net/http"

func GetCurrentUserID(r *http.Request) (int, bool) {
	session, err := Store.Get(r, "halisi-session")
	if err != nil {
		return 0, false
	}

	userID, ok := session.Values["user_id"].(int)
	if !ok {
		return 0, false
	}

	return userID, true
}
