package cli

import (
	"reflect"
	"strings"
	"testing"
)

func TestWaybackURLVariants(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{
			name: "bare domain — no duplicates",
			in:   "example.com",
			want: []string{"example.com"},
		},
		{
			name: "scheme and trailing slash both stripped progressively",
			in:   "https://simonwillison.net/",
			want: []string{
				"https://simonwillison.net/",
				"https://simonwillison.net",
				"simonwillison.net",
			},
		},
		{
			name: "www. stripped last",
			in:   "https://www.bbc.com/news",
			want: []string{
				"https://www.bbc.com/news",
				"www.bbc.com/news",
				"bbc.com/news",
			},
		},
		{
			name: "http scheme handled the same way",
			in:   "http://example.org/path/",
			want: []string{
				"http://example.org/path/",
				"http://example.org/path",
				"example.org/path",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := waybackURLVariants(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("waybackURLVariants(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestParseCDXRows(t *testing.T) {
	cases := []struct {
		name       string
		rows       [][]string
		wantErr    string // substring match; empty means no error expected
		wantMemURL string
	}{
		{
			name:    "empty array — no snapshot",
			rows:    [][]string{},
			wantErr: "no wayback snapshot available",
		},
		{
			name:    "header row only — no snapshot",
			rows:    [][]string{{"timestamp", "original"}},
			wantErr: "no wayback snapshot available",
		},
		{
			name: "short row — no snapshot",
			rows: [][]string{
				{"timestamp", "original"},
				{"20260411000000"},
			},
			wantErr: "no wayback snapshot available",
		},
		{
			name: "empty timestamp — no snapshot",
			rows: [][]string{
				{"timestamp", "original"},
				{"", "https://example.com/"},
			},
			wantErr: "no wayback snapshot available",
		},
		{
			name: "picks last row as newest",
			rows: [][]string{
				{"timestamp", "original"},
				{"20200101000000", "https://example.com/"},
				{"20230615120000", "https://example.com/"},
				{"20260411000000", "https://example.com/"},
			},
			wantMemURL: "https://web.archive.org/web/20260411000000/https://example.com/",
		},
		{
			name: "single data row works",
			rows: [][]string{
				{"timestamp", "original"},
				{"20250101000000", "https://simonwillison.net"},
			},
			wantMemURL: "https://web.archive.org/web/20250101000000/https://simonwillison.net",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m, err := parseCDXRows(tc.rows)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("error %q does not contain %q", err.Error(), tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if m.MementoURL != tc.wantMemURL {
				t.Errorf("MementoURL = %q, want %q", m.MementoURL, tc.wantMemURL)
			}
			if m.Backend != "wayback" {
				t.Errorf("Backend = %q, want wayback", m.Backend)
			}
			if m.Mirror != "web.archive.org" {
				t.Errorf("Mirror = %q, want web.archive.org", m.Mirror)
			}
		})
	}
}
