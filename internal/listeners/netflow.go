package listeners

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync/atomic"
	"time"

	"github.com/kingknull/oblivra/internal/services"
)

// NetFlow v5 packet layout:
//   24-byte header + N × 48-byte records, max 30 records/packet.
//
// Header: version(2) count(2) sysUptime(4) unixSecs(4) unixNsecs(4)
//         flowSequence(4) engineType(1) engineID(1) sampling(2)
// Record: srcIP(4) dstIP(4) nextHop(4) input(2) output(2) packets(4) bytes(4)
//         first(4) last(4) srcPort(2) dstPort(2) pad(1) tcpFlags(1) proto(1)
//         tos(1) srcAS(2) dstAS(2) srcMask(1) dstMask(1) pad(2)
const (
	netflow5Version = 5
	headerSize      = 24
	recordSize      = 48
)

type NetFlowV5 struct {
	log   *slog.Logger
	ndr   *services.NdrService
	addr  string
	conn  *net.UDPConn
	count atomic.Int64
}

type NetFlowOptions struct {
	Addr string // e.g. ":2055"
}

func NewNetFlowV5(log *slog.Logger, ndr *services.NdrService, opts NetFlowOptions) *NetFlowV5 {
	if opts.Addr == "" {
		opts.Addr = ":2055"
	}
	return &NetFlowV5{log: log, ndr: ndr, addr: opts.Addr}
}

func (n *NetFlowV5) Start(ctx context.Context) error {
	udpAddr, err := net.ResolveUDPAddr("udp", n.addr)
	if err != nil {
		return fmt.Errorf("netflow resolve: %w", err)
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("netflow listen: %w", err)
	}
	n.conn = conn
	n.log.Info("NetFlow v5 listening", "addr", conn.LocalAddr().String())

	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()

	buf := make([]byte, 64*1024)
	for {
		nb, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			n.log.Warn("netflow read", "err", err)
			continue
		}
		flows, perr := parseNetFlowV5(buf[:nb])
		if perr != nil {
			n.log.Warn("netflow parse", "err", perr)
			continue
		}
		for _, f := range flows {
			n.ndr.Record(f)
			n.count.Add(1)
		}
	}
}

func (n *NetFlowV5) Count() int64 { return n.count.Load() }

func parseNetFlowV5(b []byte) ([]services.NetFlowRecord, error) {
	if len(b) < headerSize {
		return nil, errors.New("netflow: short header")
	}
	if v := binary.BigEndian.Uint16(b[0:2]); v != netflow5Version {
		return nil, fmt.Errorf("netflow: unsupported version %d", v)
	}
	count := int(binary.BigEndian.Uint16(b[2:4]))
	if count == 0 {
		return nil, nil
	}
	expected := headerSize + count*recordSize
	if len(b) < expected {
		return nil, fmt.Errorf("netflow: short body (got %d expected %d)", len(b), expected)
	}
	unixSecs := int64(binary.BigEndian.Uint32(b[8:12]))
	unixNs := int64(binary.BigEndian.Uint32(b[12:16]))
	exportTime := time.Unix(unixSecs, unixNs).UTC()

	out := make([]services.NetFlowRecord, 0, count)
	for i := 0; i < count; i++ {
		off := headerSize + i*recordSize
		rec := b[off : off+recordSize]
		first := time.Duration(binary.BigEndian.Uint32(rec[24:28])) * time.Millisecond
		last := time.Duration(binary.BigEndian.Uint32(rec[28:32])) * time.Millisecond
		out = append(out, services.NetFlowRecord{
			StartTime: exportTime.Add(-first),
			EndTime:   exportTime.Add(-last),
			SrcIP:     net.IPv4(rec[0], rec[1], rec[2], rec[3]).String(),
			DstIP:     net.IPv4(rec[4], rec[5], rec[6], rec[7]).String(),
			SrcPort:   int(binary.BigEndian.Uint16(rec[32:34])),
			DstPort:   int(binary.BigEndian.Uint16(rec[34:36])),
			Protocol:  protoName(rec[38]),
			Bytes:     int64(binary.BigEndian.Uint32(rec[20:24])),
			Packets:   int64(binary.BigEndian.Uint32(rec[16:20])),
		})
	}
	return out, nil
}

func protoName(p byte) string {
	switch p {
	case 1:
		return "ICMP"
	case 6:
		return "TCP"
	case 17:
		return "UDP"
	case 47:
		return "GRE"
	case 50:
		return "ESP"
	default:
		return fmt.Sprintf("PROTO_%d", p)
	}
}
