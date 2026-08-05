package middleware

import "github.com/gorilla/sessions"

var Store = sessions.NewCookieStore([]byte("halisi-secret-key"))
