package dag

import (
	"context"
	"testing"
	"sync/atomic"

	"github.com/kingknull/oblivrashell/internal/events"
)

type MockProcessor struct {
	BaseNode
	count *int32
}

func (m *MockProcessor) Process(ctx context.Context, evt *Event) ([]*Event, error) {
	atomic.AddInt32(m.count, 1)
	return []*Event{evt}, nil
}

func TestEngine_Execute(t *testing.T) {
	var count1, count2, count3 int32
	
	node3 := &Node{Processor: &MockProcessor{BaseNode: BaseNode{nodeName: "node3"}, count: &count3}}
	node2 := &Node{Processor: &MockProcessor{BaseNode: BaseNode{nodeName: "node2"}, count: &count2}, Children: []*Node{node3}}
	node1 := &Node{Processor: &MockProcessor{BaseNode: BaseNode{nodeName: "node1"}, count: &count1}, Children: []*Node{node2}}
	
	engine := NewEngine(node1)
	ctx := context.Background()
	evt := &events.SovereignEvent{Id: "test-event"}
	
	if err := engine.Execute(ctx, evt); err != nil {
		t.Fatalf("Engine execute failed: %v", err)
	}
	
	if atomic.LoadInt32(&count1) != 1 {
		t.Errorf("node1 should have executed once, got %d", count1)
	}
	if atomic.LoadInt32(&count2) != 1 {
		t.Errorf("node2 should have executed once, got %d", count2)
	}
	if atomic.LoadInt32(&count3) != 1 {
		t.Errorf("node3 should have executed once, got %d", count3)
	}
}

func TestEngine_FanOut(t *testing.T) {
	var rootCount, branch1Count, branch2Count int32
	
	branch1 := &Node{Processor: &MockProcessor{BaseNode: BaseNode{nodeName: "branch1"}, count: &branch1Count}}
	branch2 := &Node{Processor: &MockProcessor{BaseNode: BaseNode{nodeName: "branch2"}, count: &branch2Count}}
	root := &Node{Processor: &MockProcessor{BaseNode: BaseNode{nodeName: "root"}, count: &rootCount}, Children: []*Node{branch1, branch2}}
	
	engine := NewEngine(root)
	ctx := context.Background()
	evt := &events.SovereignEvent{Id: "test-event"}
	
	if err := engine.Execute(ctx, evt); err != nil {
		t.Fatalf("Engine execute failed: %v", err)
	}
	
	if atomic.LoadInt32(&rootCount) != 1 {
		t.Errorf("root should have executed once, got %d", rootCount)
	}
	if atomic.LoadInt32(&branch1Count) != 1 {
		t.Errorf("branch1 should have executed once, got %d", branch1Count)
	}
	if atomic.LoadInt32(&branch2Count) != 1 {
		t.Errorf("branch2 should have executed once, got %d", branch2Count)
	}
}
