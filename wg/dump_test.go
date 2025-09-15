package wg

import (
	"bytes"
	_ "embed"
	"encoding/csv"
	"io"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/wg_show_dump.txt
var showDumpOutput []byte

//go:embed testdata/latest_handshake_zero.txt
var showDumpZeroHandshake []byte

var testDump = Dump{
	PrivateKey: "",
	PublicKey:  "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
	ListenPort: 12345,
	FwMark:     0,
	Peers: []DumpPeer{
		{
			PublicKey:       "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB",
			PresharedKey:    "",
			Endpoint:        "10.0.0.1:54321",
			AllowedIPs:      []string{"10.0.0.2/32"},
			LatestHandshake: time.Unix(1709565849, 0),
			Rx:              293787123,
			Tx:              2098018008,
			Keepalive:       15 * time.Second,
			valid:           true,
		},
		{
			PublicKey:       "CCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC",
			PresharedKey:    "",
			Endpoint:        "10.0.0.1:54322",
			AllowedIPs:      []string{"10.0.0.3/32"},
			LatestHandshake: time.Unix(1709565798, 0),
			Rx:              984267560,
			Tx:              3834155220,
			Keepalive:       0,
			valid:           true,
		},
		{
			PublicKey:       "DDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDD",
			PresharedKey:    "",
			Endpoint:        "10.0.0.1:54323",
			AllowedIPs:      []string{"10.0.0.4/32"},
			LatestHandshake: time.Unix(1709565713, 0),
			Rx:              10672758695,
			Tx:              338641384756,
			Keepalive:       0,
			valid:           true,
		},
		{
			PublicKey:       "EEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEE",
			PresharedKey:    "",
			Endpoint:        "10.0.0.1:54324",
			AllowedIPs:      []string{"10.0.0.5/32"},
			LatestHandshake: time.Unix(1709565894, 0),
			Rx:              3803572656,
			Tx:              61671294044,
			Keepalive:       0,
			valid:           true,
		},
	},
}

func TestShowDumpOutput(t *testing.T) {
	assert.NotEmpty(t, showDumpOutput)
}

func TestDump_Parse(t *testing.T) {
	b := bytes.NewBuffer(showDumpOutput)
	dump, err := NewDump(b)
	require.NoError(t, err)
	assert.Equal(t, testDump, dump)

	for i := range dump.Peers {
		peer := &dump.Peers[i]
		assert.True(t, peer.Valid())
		assert.Equal(t, peer.AllowedIPs[0], peer.Name())
		assert.Same(t, peer, dump.Peer(peer.Name()))
	}

	peer := dump.OldestHandshake()
	require.NotNil(t, peer)
	t.Log("1st oldest peer:", peer.Name(), peer.LatestHandshake)
	assert.Same(t, &dump.Peers[2], peer)

	peer2 := dump.OldestHandshake("10.0.0.4/32")
	require.NotNil(t, peer2)
	t.Log("2nd oldest peer:", peer2.Name(), peer2.LatestHandshake)
	assert.NotSame(t, peer, peer2)
	assert.Same(t, &dump.Peers[1], peer2)
}

func TestDump_Parse_readEOF(t *testing.T) {
	b := bytes.NewBufferString("")
	_, err := NewDump(b)
	require.ErrorIs(t, err, io.EOF)
	require.ErrorContains(t, err, "csv parse first line")
}

func TestDump_Parse_parseInterface_Err(t *testing.T) {
	b := bytes.NewBufferString("A\tB\tC\tD")
	_, err := NewDump(b)
	require.ErrorIs(t, err, strconv.ErrSyntax)
	require.ErrorContains(t, err, "parse interface record")
}

func TestDump_Parse_parsePeerLineErr(t *testing.T) {
	b := bytes.NewBufferString("A\tB\t0\toff\nE")
	_, err := NewDump(b)
	require.ErrorIs(t, err, csv.ErrFieldCount)
}

func TestDump_Parse_newDumpPeer_Err(t *testing.T) {
	b := bytes.NewBufferString("A\tB\t0\toff\n1\t2\t3\t4\tX\t6\t7\t8")
	_, err := NewDump(b)
	require.ErrorIs(t, err, strconv.ErrSyntax)
	require.ErrorContains(t, err, "parse peer record")
	require.ErrorContains(t, err, "failed parse latest-handshake")
}

func TestDump_parseInterface_parseFwMark_Err(t *testing.T) {
	var dump Dump
	err := dump.parseInterface([]string{"A", "B", "0", "C"})
	require.ErrorIs(t, err, strconv.ErrSyntax)
}

func TestDump_parseFwMark_hex(t *testing.T) {
	var dump Dump
	require.NoError(t, dump.parseFwMark("0xffffffff"))
}

func TestDump_Peer_notFound(t *testing.T) {
	var dump Dump
	assert.Nil(t, dump.Peer("foobar"))
}

func TestDump_Parse_zeroHandshake(t *testing.T) {
	require.NotEmpty(t, showDumpZeroHandshake)
	b := bytes.NewBuffer(showDumpZeroHandshake)
	dump, err := NewDump(b)
	require.NoError(t, err)

	peer := dump.OldestHandshake()
	require.NotNil(t, peer)
	t.Log(peer.LatestHandshake)
	assert.True(t, peer.LatestHandshake.IsZero())
}

// --------------------------------------------------

func TestDumpPeer_emptyNotValid(t *testing.T) {
	var peer DumpPeer
	assert.False(t, peer.Valid())
}

func TestDumpPeer_Parse_parseRxTx_Err(t *testing.T) {
	var peer DumpPeer
	err := peer.Parse([]string{"", "", "", "", "0", "RX", "TX"})
	require.ErrorIs(t, err, strconv.ErrSyntax)
	require.ErrorContains(t, err, "failed parse transfer-rx")

	err = peer.Parse([]string{"", "", "", "", "0", "0", "TX"})
	require.ErrorIs(t, err, strconv.ErrSyntax)
	require.ErrorContains(t, err, "failed parse transfer-tx")
}

func TestDumpPeer_parseKeepalive_Err(t *testing.T) {
	var peer DumpPeer
	err := peer.parseKeepalive("XXX")
	require.ErrorIs(t, err, strconv.ErrSyntax)
	require.ErrorContains(t, err, "failed parse persistent-keepalive")
}

func TestDumpPeer_parseLatestHanshake_zero(t *testing.T) {
	var peer DumpPeer
	require.NoError(t, peer.parseLatestHanshake("0"))
	assert.True(t, peer.LatestHandshake.IsZero())
}

func TestLookupAddr(t *testing.T) {
	tests := []struct {
		name     string
		peer     string
		errAs    any
		expected string
	}{
		{
			name: "localhost",
			peer: "127.0.0.1",
		},
		{
			name:     "no such host",
			peer:     "255.255.255.255",
			expected: "255.255.255.255",
		},
		{
			name:  "unrecognized address",
			peer:  "127.0.0.1/8",
			errAs: &net.DNSError{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hostname, err := lookupAddr(tt.peer)
			if tt.errAs != nil {
				require.ErrorAs(t, err, &tt.errAs)
			} else {
				require.NoError(t, err)
				t.Log(hostname)
				if tt.expected != "" {
					assert.Equal(t, tt.expected, hostname)
				}

			}
		})
	}
}

func TestDumpPeer_ResolvedName(t *testing.T) {
	tests := []struct {
		name  string
		peer  string
		errAs any
	}{
		{
			name: "localhost",
			peer: "127.0.0.1/32",
		},
		{
			name:  "invalid CIDR",
			peer:  "127.0.0.1",
			errAs: &net.ParseError{},
		},
		{
			name: "no such host",
			peer: "127.0.0.2/32",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			peer := DumpPeer{AllowedIPs: []string{tt.peer}}
			hostname, err := peer.ResolvedName()
			if tt.errAs != nil {
				require.ErrorAs(t, err, &tt.errAs)
			} else {
				require.NoError(t, err)
				t.Log(hostname)
			}
		})
	}
}

func TestDumpPeer_EndpointName(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		errAs    any
	}{
		{
			name:     "localhost",
			endpoint: "127.0.0.1:12345",
		},
		{
			name:     "localhost without port",
			endpoint: "127.0.0.1",
		},
		{
			name:     "no such host",
			endpoint: "127.0.0.2:12345",
		},
		{
			name:     "unrecognized address",
			endpoint: "127.0.0:12345",
			errAs:    &net.DNSError{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			peer := DumpPeer{Endpoint: tt.endpoint}
			hostname, err := peer.EndpointName()
			if tt.errAs != nil {
				require.ErrorAs(t, err, &tt.errAs)
			} else {
				require.NoError(t, err)
				t.Log(hostname)
			}
		})
	}
}
