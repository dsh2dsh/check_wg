package wg

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

const (
	dumpNone = "(none)"
	dumpOff  = "off"
)

func NewDump(r io.Reader) (Dump, error) {
	var dump Dump
	return dump, dump.Parse(r)
}

type Dump struct {
	PrivateKey string
	PublicKey  string
	ListenPort uint16
	FwMark     uint32

	Peers []DumpPeer
}

func (self *Dump) Parse(r io.Reader) error {
	lines := csv.NewReader(r)
	lines.Comma = '\t'
	lines.ReuseRecord = true

	lines.FieldsPerRecord = 4
	rec, err := lines.Read()
	if err != nil {
		return fmt.Errorf("csv parse first line %v: %w", rec, err)
	} else if err := self.parseInterface(rec); err != nil {
		return fmt.Errorf("parse interface record %v: %w", rec, err)
	}

	lines.FieldsPerRecord = 8
	for {
		rec, err := lines.Read()
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return fmt.Errorf("csv parse peer line %v: %w", rec, err)
		}
		peer, err := NewDumpPeer(rec)
		if err != nil {
			return fmt.Errorf("parse peer record %v: %w", rec, err)
		}
		self.Peers = append(self.Peers, peer)
	}
	return nil
}

func (self *Dump) parseInterface(rec []string) error {
	if rec[0] != dumpNone {
		self.PrivateKey = rec[0]
	}
	self.PublicKey = rec[1]
	if err := self.parseListenPort(rec[2]); err != nil {
		return err
	} else if err := self.parseFwMark(rec[3]); err != nil {
		return err
	}
	return nil
}

func (self *Dump) parseListenPort(s string) error {
	port, err := strconv.ParseUint(s, 10, 16)
	if err != nil {
		return fmt.Errorf("failed parse port number %q: %w", s, err)
	}
	self.ListenPort = uint16(port)
	return nil
}

func (self *Dump) parseFwMark(s string) error {
	if s == dumpOff {
		return nil
	}
	fwMark, err := strconv.ParseUint(s, 0, 32)
	if err != nil {
		return fmt.Errorf("failed parse fwmark %q: %w", s, err)
	}
	self.FwMark = uint32(fwMark)
	return nil
}

func (self *Dump) OldestHandshake() *DumpPeer {
	var oldestPeer *DumpPeer
	for i := range self.Peers {
		p := &self.Peers[i]
		if oldestPeer == nil || p.HandshakeBefore(oldestPeer) {
			oldestPeer = p
		}
	}
	return oldestPeer
}

func (self *Dump) Peer(name string) *DumpPeer {
	for i := range self.Peers {
		p := &self.Peers[i]
		if p.Name() == name {
			return p
		}
	}
	return nil
}

// --------------------------------------------------

func NewDumpPeer(rec []string) (DumpPeer, error) {
	var peer DumpPeer
	if err := peer.Parse(rec); err != nil {
		return peer, err
	}
	peer.valid = true
	return peer, nil
}

type DumpPeer struct {
	PublicKey       string
	PresharedKey    string
	Endpoint        string
	AllowedIPs      []string
	LatestHandshake time.Time
	Rx              uint64
	Tx              uint64
	Keepalive       time.Duration

	valid bool
}

func (self *DumpPeer) Parse(rec []string) error {
	self.PublicKey = rec[0]
	if rec[1] != dumpNone {
		self.PresharedKey = rec[1]
	}
	self.Endpoint = rec[2]
	self.AllowedIPs = strings.Split(rec[3], ",")
	if err := self.parseLatestHanshake(rec[4]); err != nil {
		return err
	} else if err := self.parseRxTx(rec[5], rec[6]); err != nil {
		return err
	}
	return self.parseKeepalive(rec[7])
}

func (self *DumpPeer) parseLatestHanshake(s string) error {
	secs, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return fmt.Errorf("failed parse latest-handshake %q: %w", s, err)
	} else if secs > 0 {
		self.LatestHandshake = time.Unix(int64(secs), 0)
	}
	return nil
}

func (self *DumpPeer) parseRxTx(rx string, tx string) error {
	rxBytes, err := strconv.ParseUint(rx, 10, 64)
	if err != nil {
		return fmt.Errorf("failed parse transfer-rx %q: %w", rx, err)
	}
	self.Rx = rxBytes

	txBytes, err := strconv.ParseUint(tx, 10, 64)
	if err != nil {
		return fmt.Errorf("failed parse transfer-tx %q: %w", tx, err)
	}
	self.Tx = txBytes
	return nil
}

func (self *DumpPeer) parseKeepalive(s string) error {
	if s == dumpOff {
		return nil
	}
	secs, err := strconv.ParseUint(s, 10, 16)
	if err != nil {
		return fmt.Errorf("failed parse persistent-keepalive %q: %w", s, err)
	}
	self.Keepalive = time.Duration(secs) * time.Second
	return nil
}

func (self *DumpPeer) Valid() bool {
	return self.valid
}

func (self *DumpPeer) HandshakeBefore(p *DumpPeer) bool {
	return self.LatestHandshake.Before(p.LatestHandshake)
}

func (self *DumpPeer) Name() string {
	return self.AllowedIPs[0]
}

func (self *DumpPeer) ResolvedName() (string, error) {
	cidr := self.AllowedIPs[0]
	ip, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", fmt.Errorf("parse %q: %w", cidr, err)
	}

	hostname, err := lookupAddr(ip.String())
	if err != nil {
		return "", fmt.Errorf("resolving %q from %q: %w", ip, cidr, err)
	} else if hostname == ip.String() {
		return cidr, nil
	}
	// ip/mask (hostname)
	return cidr + " (" + hostname + ")", nil
}

func lookupAddr(name string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	names, err := net.DefaultResolver.LookupAddr(ctx, name)
	if err != nil {
		var dnsError *net.DNSError
		if errors.As(err, &dnsError) && dnsError.IsNotFound {
			return name, nil
		}
		return "", fmt.Errorf("resolving %q: %w", name, err)
	}
	hostname, _ := strings.CutSuffix(names[0], ".")
	return hostname, nil
}

func (self *DumpPeer) EndpointName() (string, error) {
	ip, _, found := strings.Cut(self.Endpoint, ":")
	if !found {
		return self.Endpoint, nil
	}

	hostname, err := lookupAddr(ip)
	if err != nil {
		return "", fmt.Errorf("resolving %q from %q: %w", ip, self.Endpoint, err)
	} else if hostname == ip {
		return self.Endpoint, nil
	}
	// ip:port (hostname)
	return self.Endpoint + " (" + hostname + ")", nil
}
