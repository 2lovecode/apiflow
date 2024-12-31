package apiflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sourcegraph/conc"
	"time"
)

// ApiFlow wraps a dependency tree to provide API execution logic
type ApiFlow struct {
	tree        *DependencyTree
	resultsChan chan *Node
	timeout     time.Duration
	cancel      context.CancelFunc
	wg          *conc.WaitGroup
}

// NewApiFlow creates a new ApiFlow instance
func NewApiFlow(timeout time.Duration) *ApiFlow {
	return &ApiFlow{
		tree:        NewDependencyTree(),
		resultsChan: make(chan *Node, 100), // Buffered to avoid blocking
		timeout:     timeout,
	}
}

// AddNode adds a node with dependencies to the flow
func (af *ApiFlow) AddNode(node *Node, upstreamIDs []string) {
	af.tree.AddNode(node, upstreamIDs)
}

func (af *ApiFlow) Receive(node *Node, data interface{}) error {
	var err error
	if n, ok := af.tree.nodes[node.ID]; ok {
		node.Mutex.Lock()
		if data != nil && n.Data != nil && n.Data.Ptr != nil {
			var bd []byte
			if n.Data.LazyByte != nil && len(n.Data.LazyByte) > 0 {
				bd = n.Data.LazyByte
			} else {
				bd, err = json.Marshal(n.Data.Ptr)
			}
			if err == nil {
				err = json.Unmarshal(bd, data)
			}
		}
		node.Mutex.Unlock()
		return err
	}
	return errors.New("node not exists")
}

// Run executes the API flow
func (af *ApiFlow) Run(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, af.timeout)
	af.cancel = cancel
	defer cancel()

	af.wg = conc.NewWaitGroup()

	// Start worker goroutines to process nodes
	af.wg.Go(func() {
		af.processResults(ctx)
	})

	// Traverse the tree and start execution of root nodes
	for nodeK, _ := range af.tree.nodes {
		node := af.tree.nodes[nodeK]
		if node.Predecessors == nil || len(node.Predecessors) <= 0 {
			af.wg.Go(func() {
				af.executeNode(ctx, node)
			})
		}
	}

	rec := af.wg.WaitAndRecover()
	if rec.AsError() != nil {
		fmt.Println("error: ", rec.AsError())
	}
	close(af.resultsChan)
}

// executeNode runs the handler for a single node and sends the result to the results channel
func (af *ApiFlow) executeNode(ctx context.Context, node *Node) {
	node.Mutex.Lock()
	if node.State != StatePending {
		node.Mutex.Unlock()
		return
	}
	node.Mutex.Unlock()

	// Collect input data from predecessors
	inputs := make(map[string]interface{})
	for id, predecessor := range node.Predecessors {
		predecessor.Mutex.Lock()
		inputs[id] = predecessor.Data
		predecessor.Mutex.Unlock()
	}

	// Execute the handler with a timeout
	nodeCtx, cancel := context.WithTimeout(ctx, af.timeout)
	defer cancel()

	data, err := node.Handler(nodeCtx, node, inputs)
	node.Mutex.Lock()
	if err != nil || errors.Is(nodeCtx.Err(), context.DeadlineExceeded) {
		if errors.Is(nodeCtx.Err(), context.DeadlineExceeded) {
			node.Failure = FailureTimeout
		} else {
			node.Failure = FailureExecute
		}
		node.State = StateFailure
	} else {
		node.State = StateSuccess
		node.Data = &NodeData{
			Ptr: data,
		} // Store the result of the handler
	}
	node.Mutex.Unlock()

	af.resultsChan <- node
}

// processResults processes completed nodes and triggers successors
func (af *ApiFlow) processResults(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case node, ok := <-af.resultsChan:
			if !ok {
				return
			}
			noPending := true
			for _, n := range af.tree.nodes {
				n.Mutex.Lock()
				if n.State == StatePending {
					noPending = false
				}
				n.Mutex.Unlock()
			}
			if noPending {
				af.cancel()
				return
			}

			if node.State == StateSuccess {
				for sk, _ := range node.Successors {
					successor := node.Successors[sk]
					successorReady := true
					successor.Mutex.Lock()
					for _, predecessor := range successor.Predecessors {
						if predecessor.State != StateSuccess {
							successorReady = false
							break
						}
					}
					successor.Mutex.Unlock()
					if successorReady {
						af.wg.Go(func() {
							af.executeNode(ctx, successor)
						})
					}
				}
			} else if node.State == StateFailure {
				for sk, _ := range node.Successors {
					successor := node.Successors[sk]
					successor.Mutex.Lock()
					if successor.State == StatePending {
						successor.State = StateFailure
						successor.Failure = FailurePreError
					}
					successor.Mutex.Unlock()
					af.resultsChan <- successor
				}
			}
		}
	}
}
