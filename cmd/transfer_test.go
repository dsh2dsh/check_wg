package cmd

import (
	"fmt"
	"testing"

	"github.com/inexio/go-monitoringplugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransferResponse(t *testing.T) {
	dump, err := NewWgDump([]string{"cat", "../wg/testdata/wg_show_dump.txt"})
	require.NoError(t, err)
	require.NotNil(t, dump)

	for i := range dump.Peers {
		peer := &dump.Peers[i]
		resp := monitoringplugin.NewResponse("test OK")
		require.NoError(t, transferResponse(&dump, peer.Name(), resp))
		assert.Equal(t, monitoringplugin.OK, resp.GetStatusCode())
		assert.Contains(t, resp.GetInfo().RawOutput, "peer="+peer.Name())
		assert.Contains(t, resp.GetInfo().RawOutput,
			fmt.Sprintf("'rx'=%vb", peer.Rx))
		assert.Contains(t, resp.GetInfo().RawOutput,
			fmt.Sprintf("'tx'=%vb", peer.Tx))
	}

	resp := monitoringplugin.NewResponse("test OK")
	require.ErrorContains(t, transferResponse(&dump, "foobar", resp),
		"peer not found: foobar")
}
