package mock

import "testing"

func TestOpaqueKey(t *testing.T) {
	cases := []struct{ base, filename, want string }{
		{"", "", ""}, {"", ".", "."}, {"", "..", ".."},
		{"", "/a//b", "/a//b"}, {"", "x/../y", "x/../y"},
		{"tenant", "", "tenant/"}, {"tenant", "/a", "tenant//a"},
		{"tenant", "a//b", "tenant/a//b"}, {"tenant", ".", "tenant/."},
		{"tenant", "..", "tenant/.."}, {"tenant", "../outside", "tenant/../outside"},
		{"tenant/", "x/./y", "tenant//x/./y"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			client := NewClient(Config{BasePath: tc.base})
			if got := client.key(tc.filename); got != tc.want {
				t.Fatalf("key(%q, %q) = %q, want %q", tc.base, tc.filename, got, tc.want)
			}
		})
	}
}
