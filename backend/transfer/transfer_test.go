package transfer

import (
	"errors"
	"io"
	"testing"
)

func TestIsRetryable(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{nil, false},
		{io.EOF, true},
		{errors.New("read tcp: connection reset by peer"), true},
		{errors.New("broken pipe"), true},
		{errors.New("authentication failed"), false},
		{errors.New("Access denied for user"), false},
	}
	for _, tc := range cases {
		if got := isRetryable(tc.err); got != tc.want {
			t.Fatalf("isRetryable(%v) = %v, want %v", tc.err, got, tc.want)
		}
	}
}
