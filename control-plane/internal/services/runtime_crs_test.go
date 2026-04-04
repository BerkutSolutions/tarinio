package services

import "testing"

func TestRuntimeBaseURLFromHealthURL(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		input  string
		expect string
	}{
		{name: "default-empty", input: "", expect: "http://127.0.0.1:8081"},
		{name: "readyz", input: "http://runtime:8081/readyz", expect: "http://runtime:8081"},
		{name: "healthz", input: "https://runtime.internal:8443/healthz", expect: "https://runtime.internal:8443"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := RuntimeBaseURLFromHealthURL(tc.input); got != tc.expect {
				t.Fatalf("expected %s, got %s", tc.expect, got)
			}
		})
	}
}

