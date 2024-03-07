package cmd

import (
	"os"
	"testing"
	"time"

	"github.com/inexio/go-monitoringplugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dsh2dsh/check_wg/wg"
)

func TestOldestHandshake_errors(t *testing.T) {
	peer, err := OldestHandshake([]string{"cat", "/dev/null"})
	require.ErrorContains(t, err, "with input from")
	assert.False(t, peer.Valid())

	devnull, err := os.Open("/dev/null")
	require.NoError(t, err)
	t.Cleanup(func() { devnull.Close() })

	stdin := os.Stdin
	t.Cleanup(func() { os.Stdin = stdin })
	os.Stdin = devnull

	peer, err = OldestHandshake([]string{})
	require.ErrorContains(t, err, "with input from stdin")
	assert.False(t, peer.Valid())
}

func TestOldestHandshake_handshakeResponse(t *testing.T) {
	peer, err := OldestHandshake(
		[]string{"cat", "../wg/testdata/wg_show_dump.txt"})
	require.NoError(t, err)
	require.True(t, peer.Valid())
	assert.Equal(t, "10.0.0.4/32", peer.Name())

	tests := []struct {
		latestHandshake time.Duration
		statusCode      int
	}{
		{
			latestHandshake: handshakeWarn - time.Minute,
			statusCode:      monitoringplugin.OK,
		},
		{
			latestHandshake: handshakeWarn + time.Minute,
			statusCode:      monitoringplugin.WARNING,
		},
		{
			latestHandshake: handshakeCrit + time.Minute,
			statusCode:      monitoringplugin.CRITICAL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.latestHandshake.String(), func(t *testing.T) {
			peer.LatestHandshake = time.Now().Add(-tt.latestHandshake)
			resp := monitoringplugin.NewResponse("test OK")
			handshakeResponse(&peer, resp)
			assert.Equal(t, tt.statusCode, resp.GetStatusCode())
			assert.Contains(t, resp.GetInfo().RawOutput, "peer="+peer.Name())
		})
	}
}

func TestHandshakeResponse_peerInvalid(t *testing.T) {
	resp := monitoringplugin.NewResponse("test OK")
	var peer wg.DumpPeer
	handshakeResponse(&peer, resp)
	assert.Equal(t, monitoringplugin.WARNING, resp.GetStatusCode())
	assert.Equal(t, "WARNING: no valid peer found", resp.GetInfo().RawOutput)
}
