package dag

import (
	"context"
	"fmt"
	"sync"

	"github.com/kingknull/oblivrashell/internal/events"
)

// SovereignEvent is a type alias to the shared event type.
type Event = events.SovereignEvent

// Processor defines a single node in the processing DAG.
type Processor interface {
	Name() string
	// Process handles an event and returns a list of events to be passed to children.
	// Returning nil or empty slice stops processing for that branch.
	Process(ctx context.Context, evt *Event) ([]*Event, error)
}

// Node represents a vertex in the DAG.
type Node struct {
	Processor Processor
	Children  []*Node
}

// Engine orchestrates the execution of the DAG.
type Engine struct {
	root *Node
}

// NewEngine creates a new streaming engine with a root node.
func NewEngine(root *Node) *Engine {
	return &Engine{root: root}
}

// Execute runs the DAG for a single event.
func (e *Engine) Execute(ctx context.Context, evt *Event) error {
	if e.root == nil {
		return nil
	}
	return e.executeNode(ctx, e.root, evt)
}

func (e *Engine) executeNode(ctx context.Context, n *Node, evt *Event) error {
	outputs, err := n.Processor.Process(ctx, evt)
	if err != nil {
		return fmt.Errorf("node %s failed: %w", n.Processor.Name(), err)
	}

	if len(outputs) == 0 {
		return nil
	}

	// Fan-out to children
	var wg sync.WaitGroup
	errs := make(chan error, len(outputs)*len(n.Children))

	for _, outputEvt := range outputs {
		for _, child := range n.Children {
			wg.Add(1)
			go func(c *Node, ev *Event) {
				defer wg.Done()
				if err := e.executeNode(ctx, c, ev); err != nil {
					errs <- err
				}
			}(child, outputEvt)
		}
	}

	wg.Wait()
	close(errs)

	if len(errs) > 0 {
		return <-errs // Return the first error encountered
	}

	return nil
}
