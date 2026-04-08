package visitor

import "testing"

func TestPrimaryLanguage(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"en-US,en;q=0.9,de;q=0.8", "en-US"},
		{"en", "en"},
		{"fr-FR;q=0.9", "fr-FR"},
		{"", ""},
		{" zh-CN , en;q=0.5", "zh-CN"},
	}
	for _, tt := range tests {
		if got := primaryLanguage(tt.input); got != tt.want {
			t.Errorf("primaryLanguage(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
