package cmd

import (
	"errors"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWgDump_errors(t *testing.T) {
	_, err := NewWgDump([]string{"cat", "/dev/null"})
	require.ErrorContains(t, err, "with input from")

	devnull, err := os.Open("/dev/null")
	require.NoError(t, err)
	t.Cleanup(func() { devnull.Close() })

	stdin := os.Stdin
	t.Cleanup(func() { os.Stdin = stdin })
	os.Stdin = devnull

	_, err = NewWgDump([]string{})
	require.ErrorContains(t, err, "with input from stdin")
}

func TestWithWgCmd(t *testing.T) {
	err := withWgCmd([]string{}, func(r io.Reader) error {
		assert.Same(t, r, os.Stdin)
		return nil
	})
	require.NoError(t, err)

	var got string
	err = withWgCmd([]string{"echo", "foobar"}, func(r io.Reader) error {
		b, err := io.ReadAll(r)
		got = string(b)
		return err
	})
	require.NoError(t, err)
	assert.Equal(t, "foobar\n", got)

	wantErr := errors.New("test error")
	err = withWgCmd([]string{}, func(r io.Reader) error {
		return wantErr
	})
	require.ErrorIs(t, err, wantErr)

	err = withWgCmd([]string{""}, func(r io.Reader) error {
		return nil
	})
	require.ErrorContains(t, err, "exec: no command")

	err = withWgCmd([]string{"sh", "-c", "exit 1"}, func(r io.Reader) error {
		return nil
	})
	require.ErrorContains(t, err, "wait for")
}
