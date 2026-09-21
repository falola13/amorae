package middleware

import (
	"crypto/subtle"
	"net"
	"net/http"
	"net/netip"

	"github.com/falola13/amorae/apps/api/internal/platform/httpx"
)

// Headers the web BFF sends on every API call. Browsers never reach the API
// directly, so without them every web user would share the BFF server's IP.
const (
	HeaderClientIP  = "X-Client-IP"
	HeaderBFFSecret = "X-BFF-Secret"
)

// ClientIP resolves who is really calling and stores it for httpx.ClientIP.
//
// X-Client-IP is believed only when the request also carries the shared
// BFF secret, so proving it came from our own web server. Anyone else
// (including a mobile app, or an attacker setting headers) is identified by
// the TCP peer address, which can't be spoofed. X-Forwarded-For is never
// trusted: any client can write whatever it likes there.
func ClientIP(bffSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := peerIP(r.RemoteAddr)
			if forwarded, ok := fromBFF(r, bffSecret); ok {
				ip = forwarded
			}
			next.ServeHTTP(w, r.WithContext(httpx.WithClientIP(r.Context(), ip)))
		})
	}
}

func fromBFF(r *http.Request, secret string) (string, bool) {
	if secret == "" {
		return "", false
	}
	// Constant time, so response timing can't be used to guess the secret.
	if subtle.ConstantTimeCompare([]byte(r.Header.Get(HeaderBFFSecret)), []byte(secret)) != 1 {
		return "", false
	}
	addr, err := netip.ParseAddr(r.Header.Get(HeaderClientIP))
	if err != nil {
		return "", false
	}
	return addr.Unmap().String(), true
}

func peerIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return host
}
