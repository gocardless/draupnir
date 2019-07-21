package middleware

import (
	"net"
	"net/http"
	"strings"
	"testing"

	"github.com/prometheus/common/log"
	"github.com/stretchr/testify/assert"
)

var (
	_, allAddressesNet, _     = net.ParseCIDR("0.0.0.0/0")
	_, only100AddressesNet, _ = net.ParseCIDR("100.0.0.0/8")
)

func TestExtractRequestIPAddress(t *testing.T) {
	testCases := []struct {
		name               string
		remoteAddr         string
		forwarderForHeader string
		useXForwardedFor   bool
		trustedProxies     []*net.IPNet
		result             string
		expectedError      string
	}{
		{
			"remote addr only",
			"1.2.3.4:4567",
			"",
			true,
			[]*net.IPNet{},
			"1.2.3.4",
			"",
		},
		{
			"invalid remote addr",
			"1.2.3.4",
			"",
			true,
			[]*net.IPNet{},
			"",
			"unknown remote addr format: 1.2.3.4",
		},
		{
			"x-forwarded-for header set with single value",
			"1.2.3.4",
			"88.0.0.1",
			true,
			[]*net.IPNet{},
			"88.0.0.1",
			"",
		},
		{
			"x-forwarded-for header set with multiple values",
			"1.2.3.4",
			"88.0.0.1, 100.1.1.1",
			true,
			[]*net.IPNet{},
			"100.1.1.1",
			"",
		},
		{
			"x-forwarded-for header set with an invalid format, falls back to remote addr",
			"1.2.3.4:4567",
			"88.0.0.1 100.1.1.1", // comma is missing
			true,
			[]*net.IPNet{},
			"1.2.3.4",
			"",
		},
		{
			"x-forwarded-for header set and no elements are trusted",
			"1.2.3.4:4567",
			"88.0.0.1, 100.1.1.1",
			true,
			[]*net.IPNet{},
			"100.1.1.1",
			"",
		},
		{
			"x-forwarded-for header set and all elements are trusted, falls back to remote addr",
			"1.2.3.4:4567",
			"88.0.0.1, 100.1.1.1",
			true,
			[]*net.IPNet{
				allAddressesNet,
			},
			"1.2.3.4",
			"",
		},
		{
			"x-forwarded-for header set, but not used",
			"1.2.3.4:4567",
			"100.1.1.1, 88.0.0.1",
			false,
			[]*net.IPNet{
				only100AddressesNet,
			},
			"1.2.3.4",
			"",
		},
		{
			"x-forwarded-for header set and some elements are trusted",
			"1.2.3.4:4567",
			"100.1.1.1, 88.0.0.1",
			true,
			[]*net.IPNet{
				only100AddressesNet,
			},
			"88.0.0.1",
			"",
		},
		{
			"x-forwarded-for header set with no spaces between elements",
			"1.2.3.4:4567",
			"88.0.0.1,100.1.1.1,200.0.0.1, 100.200.100.200 ",
			true,
			[]*net.IPNet{
				only100AddressesNet,
			},
			"200.0.0.1",
			"",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			request, _ := http.NewRequest("GET", "/some/path", strings.NewReader(""))
			request.RemoteAddr = tc.remoteAddr
			request.Header.Set("x-forwarded-for", tc.forwarderForHeader)

			logger := log.NewNopLogger()
			result, err := extractRequestIPAddress(logger, request, tc.trustedProxies, tc.useXForwardedFor)

			if tc.expectedError != "" {
				assert.EqualError(t, err, tc.expectedError)
			} else {
				assert.Equal(t, tc.result, result)
			}

		})
	}
}
