package cli

import "testing"

func TestIsHardPaywallDomain(t *testing.T) {
	cases := []struct {
		url  string
		want bool
	}{
		{"https://www.wsj.com/articles/abc", true},
		{"https://wsj.com/", true},
		{"https://markets.wsj.com/quote/X", true},
		{"https://www.nytimes.com/2026/article.html", true},
		{"https://www.example.com/", false},
		{"https://notwsj.com/", false},
		{"https://www.ft.com/content/xyz", true},
		{"https://bloomberg.com/news", true},
		{"https://www.theatlantic.com/magazine/", true},
		{"not-a-url", false},
		{"", false},
	}
	for _, tc := range cases {
		got := isHardPaywallDomain(tc.url)
		if got != tc.want {
			t.Errorf("isHardPaywallDomain(%q) = %v, want %v", tc.url, got, tc.want)
		}
	}
}
