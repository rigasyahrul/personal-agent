package auth

import "crypto/subtle"

func ValidCSRF(cookie, header string) bool {
	return cookie != "" && len(cookie) == len(header) && subtle.ConstantTimeCompare([]byte(cookie), []byte(header)) == 1
}
