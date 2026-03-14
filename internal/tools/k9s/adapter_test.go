package k9s

import "testing"

func TestParseK9sInfoConfig(t *testing.T) {
	tests := []struct {
		name   string
		stdout string
		want   string
	}{
		{
			name:   "normal output",
			stdout: " ____  __.________\n|    |/ _/   __   \\\nConfig:  /home/user/.config/k9s/config.yaml\n",
			want:   "/home/user/.config/k9s/config.yaml",
		},
		{
			name:   "with ANSI escape codes",
			stdout: "\x1b[36mConfig:\x1b[0m /tmp/k9s.yaml\n",
			want:   "/tmp/k9s.yaml",
		},
		{
			name:   "no config line",
			stdout: "some random output\nother line\n",
			want:   "",
		},
		{
			name:   "empty input",
			stdout: "",
			want:   "",
		},
		{
			name:   "config with spaces",
			stdout: "Config:    /path/with spaces/config.yaml\n",
			want:   "/path/with spaces/config.yaml",
		},
	}
	for _, tt := range tests {
		got := parseK9sInfoConfig(tt.stdout)
		if got != tt.want {
			t.Errorf("%s: parseK9sInfoConfig = %q, want %q", tt.name, got, tt.want)
		}
	}
}
