package apiflow

import (
	"context"
	"fmt"
	"testing"
	"time"
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

func TestApiFlow_Do(t *testing.T) {
	ctx := context.Background()

	flow := NewApiFlow(10 * time.Second)

	dataA := &NodeAData{}
	dataB := &NodeBData{}
	dataC := &NodeCData{}
	dataD := &NodeDData{}

	nodeA := NewNode("A", func(ctx context.Context, node *Node, inputs map[string]interface{}) (interface{}, error) {
		fmt.Printf("Executing node %s\n", node.ID)
		time.Sleep(1 * time.Second)
		data := &NodeAData{
			Prop1: "propa1",
			Prop2: "propa2",
		}
		return data, nil
	})

	nodeB := NewNode("B", func(ctx context.Context, node *Node, inputs map[string]interface{}) (interface{}, error) {
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

	nodeC := NewNode("C", func(ctx context.Context, node *Node, inputs map[string]interface{}) (interface{}, error) {
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

	nodeD := NewNode("D", func(ctx context.Context, node *Node, inputs map[string]interface{}) (interface{}, error) {
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
	flow.AddNode(nodeB, []*Node{nodeA})
	flow.AddNode(nodeC, []*Node{nodeA, nodeB})
	flow.AddNode(nodeD, []*Node{nodeA})

	// Run the API flow

	flow.Run(ctx)

	if err := flow.Receive(nodeA, dataA); err != nil {
		t.Error("Receive Data A Error:", err)
	}
	if err := flow.Receive(nodeB, dataB); err != nil {
		t.Error("Receive Data B Error:", err)
	}
	if err := flow.Receive(nodeC, dataC); err != nil {
		t.Error("Receive Data C Error:", err)
	}
	if err := flow.Receive(nodeD, dataD); err != nil {
		t.Error("Receive Data D Error:", err)
	}

	if fmt.Sprint(dataA) == "&{propa1 propa2}" {
		t.Log("Receive Data A Success")
	} else {
		t.Error("Receive Data A Error")
	}
	if fmt.Sprint(dataB) == "&{propa1:propb1 propa2:propb2}" {
		t.Log("Receive Data B Success")
	} else {
		t.Error("Receive Data B Error")
	}

	if fmt.Sprint(dataC) == "&{propa1:propa1:propb1:propc1 propa2:propa2:propb2:propc2}" {
		t.Log("Receive Data C Success")
	} else {
		t.Error("Receive Data C Error")
	}

	if fmt.Sprint(dataD) == "&{propa1:propd1 propa2:propd2}" {
		t.Log("Receive Data D Success")
	} else {
		t.Error("Receive Data D Error")
	}
}
