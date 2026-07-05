package services

import (
	"net/netip"
	"testing"

	"github.com/moby/moby/api/types/network"
)

// TestPreferredContainerIP covers the v29 container-IP selection: v29 removed the
// top-level NetworkSettings.IPAddress, so we read it per network endpoint. The
// default bridge must win (matching v28 behaviour), and with no bridge the choice
// must be deterministic — not whatever Go's random map order yields.
func TestPreferredContainerIP(t *testing.T) {
	addr := netip.MustParseAddr
	tests := []struct {
		name     string
		networks map[string]*network.EndpointSettings
		want     string
	}{
		{"nil map", nil, ""},
		{"empty map", map[string]*network.EndpointSettings{}, ""},
		{
			"bridge preferred over an alphabetically-earlier network",
			map[string]*network.EndpointSettings{
				"aaa-frontend": {IPAddress: addr("10.0.0.5")},
				"bridge":       {IPAddress: addr("172.17.0.2")},
			},
			"172.17.0.2",
		},
		{
			"no bridge: lowest-named network wins deterministically",
			map[string]*network.EndpointSettings{
				"zzz": {IPAddress: addr("10.0.0.9")},
				"aaa": {IPAddress: addr("10.0.0.1")},
				"mmm": {IPAddress: addr("10.0.0.5")},
			},
			"10.0.0.1",
		},
		{
			"skips endpoints with invalid addresses",
			map[string]*network.EndpointSettings{
				"aaa": {IPAddress: netip.Addr{}},
				"bbb": {IPAddress: addr("10.0.0.7")},
			},
			"10.0.0.7",
		},
		{
			"bridge with an invalid address falls through to the named search",
			map[string]*network.EndpointSettings{
				"bridge": {IPAddress: netip.Addr{}},
				"other":  {IPAddress: addr("10.0.0.3")},
			},
			"10.0.0.3",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := preferredContainerIP(tc.networks); got != tc.want {
				t.Errorf("preferredContainerIP() = %q, want %q", got, tc.want)
			}
		})
	}
}
