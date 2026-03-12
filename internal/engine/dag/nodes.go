package dag

import (
	"context"
)

// BaseNode provides common functionality for processors.
type BaseNode struct {
	nodeName string
}

func (b *BaseNode) Name() string { return b.nodeName }

// PassthroughNode simply passes the event to children.
type PassthroughNode struct {
	BaseNode
}

func NewPassthroughNode(name string) *PassthroughNode {
	return &PassthroughNode{BaseNode{nodeName: name}}
}

func (n *PassthroughNode) Process(ctx context.Context, evt *Event) ([]*Event, error) {
	return []*Event{evt}, nil
}

// MultiDestinationNode splits processing into different paths (e.g. for branching logic).
type MultiDestinationNode struct {
	BaseNode
}

func NewMultiDestinationNode(name string) *MultiDestinationNode {
	return &MultiDestinationNode{BaseNode{nodeName: name}}
}

func (n *MultiDestinationNode) Process(ctx context.Context, evt *Event) ([]*Event, error) {
	// Root of a fan-out: return the same event so all children receive it.
	return []*Event{evt}, nil
}

// ConditionNode evaluates a function and routes events accordingly.
type ConditionNode struct {
	BaseNode
	predicate func(evt *Event) bool
}

func NewConditionNode(name string, predicate func(evt *Event) bool) *ConditionNode {
	return &ConditionNode{
		BaseNode:  BaseNode{nodeName: name},
		predicate: predicate,
	}
}

func (n *ConditionNode) Process(ctx context.Context, evt *Event) ([]*Event, error) {
	if n.predicate(evt) {
		return []*Event{evt}, nil
	}
	return nil, nil // Condition not met, stop this branch
}
