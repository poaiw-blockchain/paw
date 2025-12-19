package app

import "testing"

func TestNormalizeRPCURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "tcp scheme converted to http",
			in:   "tcp://localhost:26657",
			want: "http://localhost:26657",
		},
		{
			name: "already http",
			in:   "http://rpc.paw.io:26657",
			want: "http://rpc.paw.io:26657",
		},
		{
			name: "https preserved",
			in:   "https://rpc.pawchain.io:443",
			want: "https://rpc.pawchain.io:443",
		},
		{
			name: "blank input",
			in:   "   ",
			want: "",
		},
		{
			name: "unix scheme returned verbatim",
			in:   "unix:///tmp/paw.sock",
			want: "unix:///tmp/paw.sock",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeRPCURL(tc.in); got != tc.want {
				t.Fatalf("normalizeRPCURL(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
