package middleware

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/prometheus/common/log"

	"github.com/gocardless/draupnir/pkg/server/api/chain"
)

const UserIPAddressKey key = 4

var forwardedForSplitRegexp = regexp.MustCompile(`,\s*`)

func RecordUserIPAddress(logger log.Logger, trustedProxies []*net.IPNet, useXForwardedFor bool) chain.Middleware {
	return func(next chain.Handler) chain.Handler {
		return func(w http.ResponseWriter, r *http.Request) error {
			ipAddress, err := extractRequestIPAddress(logger, r, trustedProxies, useXForwardedFor)
			if err != nil {
				return errors.Wrap(err, "failed to determine IP address")
			}

			r = r.WithContext(context.WithValue(r.Context(), UserIPAddressKey, ipAddress))

			return next(w, r)
		}
	}
}

func extractRequestIPAddress(logger log.Logger, r *http.Request, trustedProxies []*net.IPNet, useXForwardedFor bool) (string, error) {
	var ipAddress string
	xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))

	// If X-Forwarded-For is set, and we want to use it, then inspect that first
	if useXForwardedFor && xff != "" {
		// Using a regexp to split the string into a list allows us to ignore
		// missing or extra whitespace.
		proxies := forwardedForSplitRegexp.Split(xff, -1)
		var untrustedProxies []net.IP

		// Remove any proxies from our list which we consider trusted, as per the
		// `trusted_proxy_cidrs` config option.
		for _, p := range proxies {
			ip := net.ParseIP(p)
			if ip == nil {
				// An invalid IP address was presented, so ignore it
				logger.Warnf("Invalid IP address '%s' in X-Forwarded-For header '%s'", p, xff)
				continue
			}

			trusted := false
			for _, cidr := range trustedProxies {
				if cidr.Contains(ip) {
					trusted = true
				}
			}

			if !trusted {
				untrustedProxies = append(untrustedProxies, ip)
			}
		}

		// If we still have anything left in the list of proxies then take the last
		// element in this list, as this element is the one that our load balancers
		// will have added.
		if len(untrustedProxies) != 0 {
			userIP := untrustedProxies[len(untrustedProxies)-1]
			ipAddress = userIP.String()
		}
	}

	// If there is no X-Forwarded-For header, or we haven't been able to identify
	// any valid IP addresses from it, then revert to using the IP address that
	// the request was made from.
	if ipAddress == "" {
		parts := strings.Split(r.RemoteAddr, ":")
		if len(parts) != 2 {
			return ipAddress, fmt.Errorf("unknown remote addr format: %s", r.RemoteAddr)
		}
		ipAddress = parts[0]
	}

	return ipAddress, nil
}

func GetUserIPAddress(r *http.Request) (string, error) {
	addr, ok := r.Context().Value(UserIPAddressKey).(string)
	if !ok {
		return "", errors.New("Could not determine user's IP address")
	}
	return addr, nil
}
