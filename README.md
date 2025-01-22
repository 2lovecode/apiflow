# apiflow
适用于有依赖关系的并行任务执行

### 示例
```go
package main

import (
	"context"
	"fmt"
	"time"
	"github.com/2lovecode/apiflow"
)

type NodeAData struct {
	Prop1 string
	Prop2 string
}

type NodeBData struct {
	Prop1 string
	Prop2 string
}

type NodeCData struct {
	Prop1 string
	Prop2 string
}

type NodeDData struct {
	Prop1 string
	Prop2 string
}

func main() {
	ctx := context.Background()

	flow := apiflow.NewApiFlow(10 * time.Second)

	dataA := &NodeAData{}
	dataB := &NodeBData{}
	dataC := &NodeCData{}
	dataD := &NodeDData{}

	nodeA := apiflow.NewNode("A", func(ctx context.Context, node *apiflow.Node, inputs map[string]interface{}) (interface{}, error) {
		fmt.Printf("Executing node %s\n", node.ID)
		time.Sleep(1 * time.Second)
		data := &NodeAData{
			Prop1: "propa1",
			Prop2: "propa2",
		}
		return data, nil
	})

	nodeB := apiflow.NewNode("B", func(ctx context.Context, node *apiflow.Node, inputs map[string]interface{}) (interface{}, error) {
		fmt.Printf("Executing node %s\n", node.ID)

		err := flow.Receive(nodeA, dataA)
		if err != nil {
			fmt.Println("Receive Data A Error:", err)
		}
		time.Sleep(2 * time.Second)
		data := &NodeBData{
			Prop1: fmt.Sprintf("%s:propb1", dataA.Prop1),
			Prop2: fmt.Sprintf("%s:propb2", dataA.Prop2),
		}
		return data, nil
	})

	nodeC := apiflow.NewNode("C", func(ctx context.Context, node *apiflow.Node, inputs map[string]interface{}) (interface{}, error) {
		fmt.Printf("Executing node %s\n", node.ID)
		err := flow.Receive(nodeA, dataA)
		if err != nil {
			fmt.Println("Receive Data A Error:", err)
		}
		err = flow.Receive(nodeB, dataB)
		if err != nil {
			fmt.Println("Receive Data B Error:", err)
		}
		time.Sleep(1 * time.Second)

		data := &NodeCData{
			Prop1: fmt.Sprintf("%s:%s:propc1", dataA.Prop1, dataB.Prop1),
			Prop2: fmt.Sprintf("%s:%s:propc2", dataA.Prop2, dataB.Prop2),
		}
		return data, nil
	})

	nodeD := apiflow.NewNode("D", func(ctx context.Context, node *apiflow.Node, inputs map[string]interface{}) (interface{}, error) {
		fmt.Printf("Executing node %s\n", node.ID)
		err := flow.Receive(nodeA, dataA)
		if err != nil {
			fmt.Println("Receive Data A Error:", err)
		}
		time.Sleep(3 * time.Second)
		data := &NodeDData{
			Prop1: fmt.Sprintf("%s:propd1", dataA.Prop1),
			Prop2: fmt.Sprintf("%s:propd2", dataA.Prop2),
		}
		return data, nil
	})

	// Adding nodes with dependencies
	flow.AddNode(nodeA, nil)
	flow.AddNode(nodeB, []*apiflow.Node{nodeA})
	flow.AddNode(nodeC, []*apiflow.Node{nodeA, nodeB})
	flow.AddNode(nodeD, []*apiflow.Node{nodeA})

	// Run the API flow

	flow.Run(ctx)

    
    fmt.Println("Receive Data A Error:", flow.Receive(nodeA, dataA))
    fmt.Println("Receive Data B Error:", flow.Receive(nodeB, dataB))
    fmt.Println("Receive Data C Error:", flow.Receive(nodeC, dataC))
    fmt.Println("Receive Data D Error:", flow.Receive(nodeD, dataD))
    
    fmt.Printf("Receive Data A: %v\n", dataA)
    fmt.Printf("Receive Data B: %v\n", dataB)
    fmt.Printf("Receive Data C: %v\n", dataC)
    fmt.Printf("Receive Data D: %v\n", dataD)
}
```
