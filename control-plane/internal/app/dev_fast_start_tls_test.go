package app

import "testing"

func TestShouldUseSelfSignedForDevFastStartHost(t *testing.T) {
	tests := []struct {
		name string
		host string
		want bool
	}{
		{name: "localhost", host: "localhost", want: true},
		{name: "subdomain localhost", host: "waf.localhost", want: true},
		{name: "localhost with port", host: "localhost:443", want: true},
		{name: "ipv4 loopback", host: "127.0.0.1", want: true},
		{name: "ipv4 loopback with port", host: "127.0.0.1:443", want: true},
		{name: "ipv6 loopback", host: "::1", want: true},
		{name: "ipv6 loopback with port", host: "[::1]:443", want: true},
		{name: "public host", host: "waf.example.com", want: false},
		{name: "public ip", host: "178.154.225.93", want: false},
		{name: "empty", host: "", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldUseSelfSignedForDevFastStartHost(tc.host); got != tc.want {
				t.Fatalf("host=%q got=%v want=%v", tc.host, got, tc.want)
			}
		})
	}
}
