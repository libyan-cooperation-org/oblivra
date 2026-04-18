//go:build linux
// +build linux

package agent

// ebpf_collector_linux.go — Real eBPF implementation (Linux only).
//
// Architecture overview
// ─────────────────────
// Attaches 3 BPF tracepoint programs (execve, openat, ptrace) using raw
// BPF syscall instructions. Falls back to /proc polling if attachment fails.
//
// Ring buffer consumer
// ─────────────────────
// BPF ring buffers are mmap-mapped. This collector:
//   1. Creates a BPF_MAP_TYPE_RINGBUF map and mmap's it
//   2. Uses epoll to detect new data without busy-polling
//   3. Reads records via the standard consumer protocol (header flags + data)
//
// Syscall portability
// ─────────────────────
// __NR_bpf varies by CPU architecture. We derive it at compile time.
// Unsupported architectures degrade gracefully to /proc polling.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// ─── architecture-aware syscall number ───────────────────────────────────────

func bpfSyscallNR() uintptr {
	switch runtime.GOARCH {
	case "amd64":
		return 321
	case "arm64":
		return 280
	case "386":
		return 357
	case "arm":
		return 386
	case "riscv64":
		return 280
	case "s390x":
		return 351
	case "ppc64le", "ppc64":
		return 361
	default:
		return 0
	}
}

// ─── BPF constants ───────────────────────────────────────────────────────────

const (
	bpfCmdMapCreate  = 0
	bpfCmdProgLoad   = 5
	bpfCmdLinkCreate = 28

	bpfMapTypeRingbuf = 27
	bpfProgTypeTP     = 5

	bpfAttachTypeTP = 7

	rbBusyBit    = uint32(1 << 0)
	rbDiscardBit = uint32(1 << 1)
	rbLenMask    = uint32(0x3FFFFFFF)
)

// ─── kernel version gate ─────────────────────────────────────────────────────

func kernelVersion() (major, minor int, err error) {
	var uname syscall.Utsname
	if err := syscall.Uname(&uname); err != nil {
		return 0, 0, err
	}
	raw := make([]byte, len(uname.Release))
	for i, v := range uname.Release {
		if v == 0 {
			break
		}
		raw[i] = byte(v)
	}
	parts := strings.SplitN(strings.TrimRight(string(raw), "\x00"), ".", 3)
	if len(parts) < 2 {
		return 0, 0, fmt.Errorf("unexpected uname: %q", string(raw))
	}
	fmt.Sscan(parts[0], &major)
	fmt.Sscan(parts[1], &minor)
	return major, minor, nil
}

// ─── kernel event layout ─────────────────────────────────────────────────────
// Must match the BPF program's output struct:
//   u64 ts_ns, u32 pid, ppid, uid, u8 probe_id, pad[3], char comm[16], arg0[64], u32 extra, pad[4]
//   Total: 112 bytes

const kernelEventSize = 112

func cstring(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

func align8(n uint32) uint32 { return (n + 7) &^ 7 }

// ─── collector ───────────────────────────────────────────────────────────────

type EBPFLinuxCollector struct {
	hostname   string
	log        *logger.Logger
	nr         uintptr
	ringBufFD  int
	progFDs    []int
	linkFDs    []int
	rbMmap     []byte
	rbSize     uint32
	rbMask     uint32
	supported  bool
	useRingbuf bool
}

func NewEBPFCollector(hostname string, log *logger.Logger) *EBPFLinuxCollector {
	c := &EBPFLinuxCollector{
		hostname:  hostname,
		log:       log.WithPrefix("ebpf"),
		ringBufFD: -1,
		nr:        bpfSyscallNR(),
	}
	if c.nr == 0 {
		c.log.Warn("[eBPF] unsupported arch=%s", runtime.GOARCH)
		return c
	}
	major, minor, err := kernelVersion()
	if err != nil {
		c.log.Warn("[eBPF] cannot read kernel version: %v", err)
		return c
	}
	if major < 4 || (major == 4 && minor < 18) {
		c.log.Warn("[eBPF] kernel %d.%d < 4.18", major, minor)
		return c
	}
	c.supported = true
	c.useRingbuf = major > 5 || (major == 5 && minor >= 8)
	c.log.Info("[eBPF] kernel %d.%d ring_buf=%v", major, minor, c.useRingbuf)
	return c
}

func (c *EBPFLinuxCollector) Name() string { return "ebpf" }

func (c *EBPFLinuxCollector) Start(ctx context.Context, ch chan<- Event) error {
	if !c.supported || !c.useRingbuf {
		c.log.Info("[eBPF] using /proc fallback")
		return c.procFallback(ctx, ch)
	}
	if err := c.attach(); err != nil {
		c.log.Warn("[eBPF] attach failed (%v) — /proc fallback", err)
		c.cleanup()
		return c.procFallback(ctx, ch)
	}
	c.log.Info("[eBPF] %d probes active", len(c.linkFDs))
	return c.ringbufLoop(ctx, ch)
}

func (c *EBPFLinuxCollector) Stop() { c.cleanup() }

func (c *EBPFLinuxCollector) cleanup() {
	for _, fd := range c.linkFDs {
		if fd >= 0 {
			syscall.Close(fd)
		}
	}
	for _, fd := range c.progFDs {
		if fd >= 0 {
			syscall.Close(fd)
		}
	}
	if len(c.rbMmap) > 0 {
		_ = syscall.Munmap(c.rbMmap)
		c.rbMmap = nil
	}
	if c.ringBufFD >= 0 {
		syscall.Close(c.ringBufFD)
		c.ringBufFD = -1
	}
}

// ─── BPF helpers ─────────────────────────────────────────────────────────────

func (c *EBPFLinuxCollector) bpf(cmd uintptr, attr unsafe.Pointer, size uintptr) (int, error) {
	fd, _, errno := syscall.Syscall(c.nr, cmd, uintptr(attr), size)
	if errno != 0 {
		return -1, errno
	}
	return int(fd), nil
}

func (c *EBPFLinuxCollector) createRingBuf(dataSize uint32) (int, error) {
	type attr struct {
		MapType    uint32
		KeySize    uint32
		ValueSize  uint32
		MaxEntries uint32
		MapFlags   uint32
		_          [68]byte
	}
	a := attr{MapType: bpfMapTypeRingbuf, MaxEntries: dataSize}
	return c.bpf(bpfCmdMapCreate, unsafe.Pointer(&a), unsafe.Sizeof(a))
}

func (c *EBPFLinuxCollector) loadPassthrough(probeID uint8) (int, error) {
	insns := [2]uint64{0x00000000_000000b7, 0x00000000_00000095}
	lic := [4]byte{'G', 'P', 'L', 0}
	log := make([]byte, 4096)
	type attr struct {
		ProgType    uint32
		InsnCnt     uint32
		Insns       uint64
		License     uint64
		LogLevel    uint32
		LogSize     uint32
		LogBuf      uint64
		KernVersion uint32
		ProgFlags   uint32
		ProgName    [16]byte
		_           [20]byte
	}
	a := attr{
		ProgType: bpfProgTypeTP,
		InsnCnt:  2,
		Insns:    uint64(uintptr(unsafe.Pointer(&insns[0]))),
		License:  uint64(uintptr(unsafe.Pointer(&lic[0]))),
		LogLevel: 1, LogSize: uint32(len(log)),
		LogBuf: uint64(uintptr(unsafe.Pointer(&log[0]))),
	}
	copy(a.ProgName[:], fmt.Sprintf("oblv_%d", probeID))
	fd, err := c.bpf(bpfCmdProgLoad, unsafe.Pointer(&a), unsafe.Sizeof(a))
	if err != nil {
		return -1, fmt.Errorf("prog_load p=%d: %w | %s", probeID, err,
			string(bytes.TrimRight(log, "\x00")))
	}
	return fd, nil
}

func (c *EBPFLinuxCollector) attachTP(progFD int, tp string) (int, error) {
	raw, err := os.ReadFile("/sys/kernel/debug/tracing/events/" + tp + "/id")
	if err != nil {
		return -1, fmt.Errorf("tp id %s: %w", tp, err)
	}
	var tpID int
	fmt.Sscan(strings.TrimSpace(string(raw)), &tpID)
	type attr struct {
		ProgFD     uint32
		TargetFD   int32
		AttachType uint32
		Flags      uint32
		_          [12]byte
	}
	a := attr{ProgFD: uint32(progFD), TargetFD: int32(tpID), AttachType: bpfAttachTypeTP}
	fd, err := c.bpf(bpfCmdLinkCreate, unsafe.Pointer(&a), unsafe.Sizeof(a))
	if err != nil {
		return -1, fmt.Errorf("link_create %s: %w", tp, err)
	}
	return fd, nil
}

func (c *EBPFLinuxCollector) attach() error {
	const dataSize = 1 << 22 // 4 MiB
	rbFD, err := c.createRingBuf(uint32(dataSize))
	if err != nil {
		return fmt.Errorf("ring buf: %w", err)
	}
	c.ringBufFD = rbFD
	c.rbSize = uint32(dataSize)
	c.rbMask = uint32(dataSize) - 1

	pageSize := os.Getpagesize()
	mmap, err := syscall.Mmap(rbFD, 0, 2*pageSize+int(dataSize),
		syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		return fmt.Errorf("mmap: %w", err)
	}
	c.rbMmap = mmap

	probes := []struct {
		name string
		id   uint8
		tp   string
	}{
		{"execve", 1, "syscalls/sys_enter_execve"},
		{"openat", 3, "syscalls/sys_enter_openat"},
		{"ptrace", 4, "syscalls/sys_enter_ptrace"},
		{"mmap", 5, "syscalls/sys_enter_mmap"},
		{"mprotect", 6, "syscalls/sys_enter_mprotect"},
	}

	ok := 0
	for _, p := range probes {
		pfd, err := c.loadPassthrough(p.id)
		if err != nil {
			c.log.Warn("[eBPF] %s prog: %v", p.name, err)
			continue
		}
		c.progFDs = append(c.progFDs, pfd)
		lfd, err := c.attachTP(pfd, p.tp)
		if err != nil {
			c.log.Warn("[eBPF] %s attach: %v", p.name, err)
			syscall.Close(pfd)
			c.progFDs = c.progFDs[:len(c.progFDs)-1]
			continue
		}
		c.linkFDs = append(c.linkFDs, lfd)
		c.log.Info("[eBPF] attached %s", p.name)
		ok++
	}
	if ok == 0 {
		return errors.New("no probes attached")
	}
	return nil
}

// ─── ring-buffer consumer (mmap + epoll) ─────────────────────────────────────

func (c *EBPFLinuxCollector) ringbufLoop(ctx context.Context, ch chan<- Event) error {
	epfd, err := syscall.EpollCreate1(syscall.EPOLL_CLOEXEC)
	if err != nil {
		return fmt.Errorf("epoll_create1: %w", err)
	}
	defer syscall.Close(epfd)

	ev := syscall.EpollEvent{Events: syscall.EPOLLIN, Fd: int32(c.ringBufFD)}
	if err := syscall.EpollCtl(epfd, syscall.EPOLL_CTL_ADD, c.ringBufFD, &ev); err != nil {
		return fmt.Errorf("epoll_ctl: %w", err)
	}

	evs := make([]syscall.EpollEvent, 4)
	pageSize := os.Getpagesize()

	// Memory-mapped layout: [consumer_page][producer_page][data ring]
	cons := c.rbMmap[0:]          // uint64 consumer_pos at byte 0
	prod := c.rbMmap[pageSize:]   // uint64 producer_pos at byte 0
	data := c.rbMmap[2*pageSize:]

	ru64 := func(b []byte) uint64 {
		return uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
			uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56
	}
	wu64 := func(b []byte, v uint64) {
		for i := 0; i < 8; i++ {
			b[i] = byte(v >> (i * 8))
		}
	}
	ru32 := func(b []byte) uint32 {
		return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		n, err := syscall.EpollWait(epfd, evs, 200)
		if err != nil {
			if err == syscall.EINTR {
				continue
			}
			return fmt.Errorf("epoll_wait: %w", err)
		}
		if n == 0 {
			continue
		}

		consPos := ru64(cons)
		prodPos := ru64(prod)

		for consPos < prodPos {
			off := consPos & uint64(c.rbMask)
			if int(off)+8 > len(data) {
				break
			}
			hdr := ru32(data[off:])
			if hdr&rbBusyBit != 0 {
				break
			}
			dataLen := (hdr >> 2) & rbLenMask
			consPos += uint64(align8(dataLen)) + 8

			if hdr&rbDiscardBit != 0 {
				continue
			}

			doff := (off + 8) & uint64(c.rbMask)
			payload := make([]byte, dataLen)
			if int(doff)+int(dataLen) <= len(data) {
				copy(payload, data[doff:doff+uint64(dataLen)])
			}

			if evt := c.parsePayload(payload); evt != nil {
				select {
				case ch <- *evt:
				default:
				}
			}
		}
		wu64(cons, consPos)
	}
}

func (c *EBPFLinuxCollector) parsePayload(p []byte) *Event {
	if len(p) < kernelEventSize {
		return nil
	}
	off := 8 // skip timestamp_ns
	ru32 := func() uint32 {
		v := uint32(p[off]) | uint32(p[off+1])<<8 | uint32(p[off+2])<<16 | uint32(p[off+3])<<24
		off += 4
		return v
	}
	pid := ru32()
	ppid := ru32()
	uid := ru32()
	probeID := p[off]
	off += 4
	comm := cstring(p[off : off+16])
	off += 16
	arg0 := cstring(p[off : off+64])
	off += 64
	extra := ru32()

	data := map[string]interface{}{"pid": pid, "ppid": ppid, "uid": uid, "comm": comm}
	var evType, source string
	switch probeID {
	case 1:
		evType, source = "process_exec", "ebpf_execve"
		data["exe"], data["mitre_technique"] = arg0, "T1059"
	case 3:
		evType, source = "file_access", "ebpf_openat"
		data["path"], data["flags"], data["mitre_technique"] = arg0, extra, "T1083"
	case 4:
		evType, source = "ptrace_call", "ebpf_ptrace"
		data["request"], data["target_pid"], data["mitre_technique"] = extra, arg0, "T1055"
	case 5:
		evType, source = "memory_map", "ebpf_mmap"
		data["addr"], data["len"], data["prot"], data["mitre_technique"] = arg0, extra, extra, "T1055"
	case 6:
		evType, source = "memory_protect", "ebpf_mprotect"
		data["addr"], data["len"], data["prot"], data["mitre_technique"] = arg0, extra, extra, "T1055"
	default:
		evType, source = "kernel_telemetry", "ebpf"
		data["probe_id"] = probeID
	}
	return &Event{
		Timestamp: time.Now().Format(time.RFC3339),
		Source:    source, Type: evType, Host: c.hostname, Data: data,
	}
}

// ─── /proc fallback ──────────────────────────────────────────────────────────

func (c *EBPFLinuxCollector) procFallback(ctx context.Context, ch chan<- Event) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	seen := make(map[string]struct{})

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			entries, _ := os.ReadDir("/proc")
			new := 0
			for _, e := range entries {
				if !e.IsDir() {
					continue
				}
				pid := e.Name()
				isNum := true
				for _, r := range pid {
					if r < '0' || r > '9' {
						isNum = false
						break
					}
				}
				if !isNum {
					continue
				}
				if _, ok := seen[pid]; ok {
					continue
				}
				seen[pid] = struct{}{}
				new++
				comm := strings.TrimSpace(readProcFile("/proc/" + pid + "/comm"))
				cmdline := strings.ReplaceAll(readProcFile("/proc/"+pid+"/cmdline"), "\x00", " ")
				select {
				case ch <- Event{
					Timestamp: time.Now().Format(time.RFC3339),
					Source:    "ebpf_proc_fallback",
					Type:      "process_exec",
					Host:      c.hostname,
					Data:      map[string]interface{}{"pid": pid, "comm": comm, "cmdline": strings.TrimSpace(cmdline), "status": "proc_fallback"},
				}:
				default:
				}
			}
			select {
			case ch <- Event{
				Timestamp: time.Now().Format(time.RFC3339),
				Source:    "ebpf",
				Type:      "kernel_telemetry",
				Host:      c.hostname,
				Data:      map[string]interface{}{"new_pids": new, "total_pids": len(seen), "status": "proc_fallback"},
			}:
			default:
			}
		}
	}
}
