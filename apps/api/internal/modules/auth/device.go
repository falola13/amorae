package auth

import "strings"

// deviceLabel turns a user agent into something a person can recognise in
// their own session list — "Safari on iPhone" — and nothing finer. It is
// deliberately coarse: the point is "is that me?", not device fingerprinting,
// and the raw string never leaves the API.
func deviceLabel(userAgent string) string {
	if strings.TrimSpace(userAgent) == "" {
		return "Unknown device"
	}

	browser := ""
	switch {
	// Order matters: Edge's agent also says Chrome and Safari, and Chrome's
	// also says Safari.
	case strings.Contains(userAgent, "Edg/"):
		browser = "Edge"
	case strings.Contains(userAgent, "OPR/"):
		browser = "Opera"
	case strings.Contains(userAgent, "SamsungBrowser"):
		browser = "Samsung Internet"
	case strings.Contains(userAgent, "Firefox/"):
		browser = "Firefox"
	case strings.Contains(userAgent, "Chrome/"):
		browser = "Chrome"
	case strings.Contains(userAgent, "Safari/"):
		browser = "Safari"
	}

	platform := ""
	switch {
	case strings.Contains(userAgent, "iPhone"):
		platform = "iPhone"
	case strings.Contains(userAgent, "iPad"):
		platform = "iPad"
	case strings.Contains(userAgent, "Android"):
		platform = "Android"
	case strings.Contains(userAgent, "Macintosh"), strings.Contains(userAgent, "Mac OS X"):
		platform = "Mac"
	case strings.Contains(userAgent, "Windows"):
		platform = "Windows"
	case strings.Contains(userAgent, "Linux"):
		platform = "Linux"
	}

	switch {
	case browser != "" && platform != "":
		return browser + " on " + platform
	case browser != "":
		return browser
	case platform != "":
		return platform
	default:
		return "Unknown device"
	}
}
