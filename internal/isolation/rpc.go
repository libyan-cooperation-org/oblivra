package isolation

import (
	"io"
	"net/rpc"
	"net/rpc/jsonrpc"
)

// IsolatedWorker represents the RPC server inside the child process.
type IsolatedWorker struct {
	WorkerType string
	Server     *rpc.Server
}

// NewIsolatedWorker creates a new JSON-RPC worker reading from stdin/stdout.
func NewIsolatedWorker(wType string) *IsolatedWorker {
	server := rpc.NewServer()
	return &IsolatedWorker{
		WorkerType: wType,
		Server:     server,
	}
}

// Register registers an RPC receiver object.
func (w *IsolatedWorker) Register(name string, rcvr interface{}) error {
	return w.Server.RegisterName(name, rcvr)
}

// ServeStdinStdout starts serving JSON-RPC requests on standard input/output.
// This blocks forever until the parent closes stdin.
func (w *IsolatedWorker) ServeStdinStdout(in io.Reader, out io.Writer) {
	// We use an empty interface struct to fulfill io.ReadWriteCloser
	codec := jsonrpc.NewServerCodec(&stdioPipe{
		r: in,
		w: out,
	})
	w.Server.ServeCodec(codec)
}

// stdioPipe wraps an io.Reader and io.Writer into an io.ReadWriteCloser.
// Closing is a no-op since it's stdin/stdout.
type stdioPipe struct {
	r io.Reader
	w io.Writer
}

func (p *stdioPipe) Read(b []byte) (n int, err error) {
	return p.r.Read(b)
}

func (p *stdioPipe) Write(b []byte) (n int, err error) {
	return p.w.Write(b)
}

func (p *stdioPipe) Close() error {
	return nil
}

// RPC types for Detection Engine (example)

type EvaluateEventArgs struct {
	RawEventJSON []byte
}

type EvaluateEventResponse struct {
	Matches []string // Rule IDs
}

// IsolationClient is used by the parent process to talk to children
type IsolationClient struct {
	client *rpc.Client
}

func NewIsolationClient(in io.Reader, out io.Writer) *IsolationClient {
	return &IsolationClient{
		client: jsonrpc.NewClient(&stdioPipe{r: in, w: out}),
	}
}

func (c *IsolationClient) Call(serviceMethod string, args interface{}, reply interface{}) error {
	return c.client.Call(serviceMethod, args, reply)
}

func (c *IsolationClient) Close() error {
	return c.client.Close()
}
