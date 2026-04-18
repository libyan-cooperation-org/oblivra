package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/vault"
)

var (
	socketPath = flag.String("socket", "/tmp/oblivra-vault.sock", "Path for the Unix domain socket")
	parentPID  = flag.Int("ppid", -1, "Parent PID to monitor for self-termination")
)

type rpcRequest struct {
	Op      string `json:"op"`
	Payload []byte `json:"payload"`
}

type rpcResponse struct {
	Result []byte `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

func main() {
	flag.Parse()

	// 5. Sovereign-Grade: Memory Hardening
	// Prevent the vault daemon's memory from being swapped to disk.
	if err := syscall.Mlockall(syscall.MCL_CURRENT | syscall.MCL_FUTURE); err != nil {
		fmt.Fprintf(os.Stderr, "WARNING: mlockall failed: %v — memory isolation weakened\n", err)
	}

	appLog := logger.NewStdoutLogger()
	v, err := vault.New(vault.Config{StorePath: "./.vault"}, appLog)
	if err != nil {
		log.Fatalf("failed to init vault: %v", err)
	}

	appLog.Info("Vault daemon initialized (Locked) and starting up...")

	if *parentPID > 0 {
		go monitorParent(*parentPID)
	}

	os.Remove(*socketPath)

	listener, err := net.Listen("unix", *socketPath)
	if err != nil {
		log.Fatalf("failed to listen on socket %s: %v", *socketPath, err)
	}
	defer listener.Close()

	// Adjust socket permissions to be private to this user
	os.Chmod(*socketPath, 0600)

	appLog.Info("Vault daemon listening on %s", *socketPath)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		appLog.Info("Shutting down vault daemon...")
		v.Lock()
		listener.Close()
		os.Exit(0)
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			break
		}
		go handleConnection(conn, v)
	}
}

func monitorParent(ppid int) {
	for {
		time.Sleep(2 * time.Second)
		process, err := os.FindProcess(ppid)
		if err != nil {
			os.Exit(0)
		}
		// Sending signal 0 checks if process exists (unix mainly, but Go tries its best)
		// More robust on Windows/Unix varies, but we'll use a basic check.
		err = process.Signal(syscall.Signal(0))
		if err != nil {
			os.Exit(0) // Parent died
		}
	}
}

func handleConnection(conn net.Conn, v *vault.Vault) {
	defer conn.Close()

	for {
		// Read length (4 bytes big-endian)
		lenBuf := make([]byte, 4)
		if _, err := io.ReadFull(conn, lenBuf); err != nil {
			return
		}

		length := int(lenBuf[0])<<24 | int(lenBuf[1])<<16 | int(lenBuf[2])<<8 | int(lenBuf[3])
		if length <= 0 || length > 10*1024*1024 {
			return
		}

		data := make([]byte, length)
		if _, err := io.ReadFull(conn, data); err != nil {
			return
		}

		var req rpcRequest
		if err := json.Unmarshal(data, &req); err != nil {
			sendError(conn, "invalid json")
			continue
		}

		var resp rpcResponse
		switch req.Op {
		case "ping":
			if v.IsUnlocked() {
				resp.Result = []byte("pong")
			} else {
				resp.Error = "locked"
			}
		case "unlock":
			if err := v.Unlock(string(req.Payload), nil, false); err != nil {
				resp.Error = err.Error()
			} else {
				resp.Result = []byte("ok")
			}
		case "lock":
			v.Lock()
			resp.Result = []byte("ok")
		case "master_key":
			err := v.AccessMasterKey(func(key []byte) error {
				resp.Result = make([]byte, len(key))
				copy(resp.Result, key)
				return nil
			})
			if err != nil {
				resp.Error = err.Error()
			}
		case "encrypt":
			res, err := v.Encrypt(req.Payload)
			if err != nil {
				resp.Error = err.Error()
			} else {
				resp.Result = res
			}
		case "decrypt":
			res, err := v.Decrypt(req.Payload)
			if err != nil {
				resp.Error = err.Error()
			} else {
				resp.Result = res
			}
		default:
			resp.Error = "unknown op"
		}

		sendResponse(conn, resp)
	}
}

func sendResponse(conn net.Conn, resp rpcResponse) {
	data, _ := json.Marshal(resp)
	frame := make([]byte, 4+len(data))
	l := len(data)
	frame[0] = byte(l >> 24)
	frame[1] = byte(l >> 16)
	frame[2] = byte(l >> 8)
	frame[3] = byte(l)
	copy(frame[4:], data)
	conn.Write(frame)
}

func sendError(conn net.Conn, msg string) {
	sendResponse(conn, rpcResponse{Error: msg})
}
