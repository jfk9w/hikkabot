package httpf

import "net/http"

type basicAuth [2]string

func (a basicAuth) SetAuth(req *http.Request) {
	req.SetBasicAuth(a[0], a[1])
}

// Basic returns Basic Authorization header.
func Basic(username, password string) Authorization {
	return basicAuth{username, password}
}

type bearerAuth string

func (a bearerAuth) SetAuth(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+string(a))
}

// Bearer returns Bearer Authorization header.
func Bearer(token string) Authorization {
	return bearerAuth(token)
}
