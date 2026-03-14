package execx

import (
	"errors"
	"testing"
)

func TestErrMessage(t *testing.T) {
	tests := []struct {
		name string
		res  CmdResult
		want string
	}{
		{name: "prefers stderr", res: CmdResult{Stderr: "bad", Err: errors.New("fail")}, want: "bad"},
		{name: "falls back to err", res: CmdResult{Stderr: "", Err: errors.New("fail")}, want: "fail"},
		{name: "empty when no error", res: CmdResult{}, want: ""},
		{name: "whitespace stderr falls back", res: CmdResult{Stderr: "  \n", Err: errors.New("fail")}, want: "fail"},
	}
	for _, tt := range tests {
		got := ErrMessage(tt.res)
		if got != tt.want {
			t.Errorf("%s: ErrMessage = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestStdoutOrStderr(t *testing.T) {
	tests := []struct {
		name string
		res  CmdResult
		want string
	}{
		{name: "prefers stdout", res: CmdResult{Stdout: "out", Stderr: "err"}, want: "out"},
		{name: "falls back to stderr", res: CmdResult{Stdout: "", Stderr: "err"}, want: "err"},
		{name: "whitespace stdout falls back", res: CmdResult{Stdout: "  \n", Stderr: "err"}, want: "err"},
		{name: "both empty", res: CmdResult{}, want: ""},
	}
	for _, tt := range tests {
		got := StdoutOrStderr(tt.res)
		if got != tt.want {
			t.Errorf("%s: StdoutOrStderr = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestFirstNonEmpty(t *testing.T) {
	tests := []struct {
		name string
		vals []string
		want string
	}{
		{name: "first non-empty", vals: []string{"", "  ", "hello"}, want: "hello"},
		{name: "all empty", vals: []string{"", "  "}, want: ""},
		{name: "first wins", vals: []string{"a", "b"}, want: "a"},
		{name: "no vals", vals: nil, want: ""},
	}
	for _, tt := range tests {
		got := FirstNonEmpty(tt.vals...)
		if got != tt.want {
			t.Errorf("%s: FirstNonEmpty = %q, want %q", tt.name, got, tt.want)
		}
	}
}
