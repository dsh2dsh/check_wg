package cmd

import (
	"testing"
	"time"

	"github.com/inexio/go-monitoringplugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandshakeResponse(t *testing.T) {
	dump, err := NewWgDump([]string{"cat", "../wg/testdata/wg_show_dump.txt"})
	require.NoError(t, err)
	require.NotNil(t, dump)

	peer := dump.OldestHandshake()
	require.NotNil(t, peer)
	assert.Equal(t, "10.0.0.4/32", peer.Name())
	for i := range dump.Peers {
		p := &dump.Peers[i]
		p.LatestHandshake = time.Now()
	}

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
			require.NoError(t, handshakeResponse(&dump, resp))
			assert.Equal(t, tt.statusCode, resp.GetStatusCode())
			assert.Contains(t, resp.GetInfo().RawOutput, "peer="+peer.Name())
		})
	}
}

func TestHandshakeResponse_errors(t *testing.T) {
	dump, err := NewWgDump(
		[]string{"head", "-1", "../wg/testdata/wg_show_dump.txt"})
	require.NoError(t, err)
	require.NotNil(t, dump)

	resp := monitoringplugin.NewResponse("test OK")
	require.ErrorContains(t, handshakeResponse(&dump, resp),
		"no valid peer found")
}
