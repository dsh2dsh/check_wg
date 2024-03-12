package cmd

import (
	"errors"
	"io"
	"os"
	"testing"

	"github.com/inexio/go-monitoringplugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dsh2dsh/check_wg/wg"
)

func TestMonitoringResponse(t *testing.T) {
	var callCount int
	resp := monitoringResponse("test OK",
		[]string{"cat", "../wg/testdata/wg_show_dump.txt"},
		func(dump *wg.Dump, resp *monitoringplugin.Response) error {
			callCount++
			return nil
		})
	require.NotNil(t, resp)
	assert.Equal(t, 1, callCount)
	assert.Equal(t, monitoringplugin.OK, resp.GetStatusCode())

	resp = monitoringResponse("test OK", []string{"cat", "/dev/null"},
		func(dump *wg.Dump, resp *monitoringplugin.Response) error {
			callCount++
			return nil
		})
	require.NotNil(t, resp)
	assert.Equal(t, 1, callCount)
	assert.Equal(t, monitoringplugin.UNKNOWN, resp.GetStatusCode())
	assert.Contains(t, resp.GetInfo().RawOutput,
		"with input from [cat /dev/null]")

	wantErr := errors.New("test error")
	resp = monitoringResponse("test OK",
		[]string{"cat", "../wg/testdata/wg_show_dump.txt"},
		func(dump *wg.Dump, resp *monitoringplugin.Response) error {
			return wantErr
		})
	require.NotNil(t, resp)
	assert.Equal(t, monitoringplugin.UNKNOWN, resp.GetStatusCode())
	assert.Contains(t, resp.GetInfo().RawOutput, wantErr.Error())
}

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
