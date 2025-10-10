package main

import (
	"fmt"

	"github.com/ComingCL/go-workflow/workflow"
)

// Node types and phases - define locally since they might be missing from the package
const (
	NodeTypeStart   workflow.NodeType = "start"
	NodeTypeEnd     workflow.NodeType = "end"
	NodeTypeDeploy  workflow.NodeType = "deploy"
	NodeTypeApiCall workflow.NodeType = "api-call"
	NodeTypeBuild   workflow.NodeType = "build"
)

const (
	NodePending   workflow.NodePhase = "Pending"
	NodeRunning   workflow.NodePhase = "Running"
	NodeSucceeded workflow.NodePhase = "Succeeded"
	NodeSkipped   workflow.NodePhase = "Skipped"
	NodeFailed    workflow.NodePhase = "Failed"
	NodeError     workflow.NodePhase = "Error"
	NodeOmitted   workflow.NodePhase = "Omitted"
)

func main() {
	// Create DAG
	dag := workflow.NewWorkflowDAG()

	// Add nodes
	node1 := workflow.NewWorkflowNode("node1", "Start Node", NodePending, NodeTypeStart)
	node2 := workflow.NewWorkflowNode("node2", "Build Node", NodePending, NodeTypeBuild)
	node3 := workflow.NewWorkflowNode("node3", "Test Node", NodePending, NodeTypeBuild)
	node4 := workflow.NewWorkflowNode("node4", "Deploy Node", NodePending, NodeTypeDeploy)
	node5 := workflow.NewWorkflowNode("node5", "End Node", NodePending, NodeTypeEnd)

	// Add nodes to DAG
	dag.AddNode(node1)
	dag.AddNode(node2)
	dag.AddNode(node3)
	dag.AddNode(node4)
	dag.AddNode(node5)

	// Add dependencies
	// Start -> Build & Test (parallel)
	dag.AddDependency("node1", "node2") // Start -> Build
	dag.AddDependency("node1", "node3") // Start -> Test

	// Build & Test -> Deploy
	dag.AddDependency("node2", "node4") // Build -> Deploy
	dag.AddDependency("node3", "node4") // Test -> Deploy

	// Deploy -> End
	dag.AddDependency("node4", "node5") // Deploy -> End

	// Validate DAG
	err := dag.ValidateDAG()
	if err != nil {
		fmt.Printf("DAG validation failed: %v\n", err)
		return
	}
	fmt.Println("✓ DAG validation passed")

	// Get execution order
	order, err := dag.GetExecutionOrder()
	if err != nil {
		fmt.Printf("Failed to get execution order: %v\n", err)
		return
	}
	fmt.Println("Execution order:", order)

	// Get root nodes (nodes with no dependencies)
	roots := dag.GetRootNodes()
	fmt.Println("Root nodes:")
	for _, root := range roots {
		fmt.Printf("  - %s (%s)\n", root.Name, root.ID)
	}

	// Get leaf nodes (nodes with no dependents)
	leaves := dag.GetLeafNodes()
	fmt.Println("Leaf nodes:")
	for _, leaf := range leaves {
		fmt.Printf("  - %s (%s)\n", leaf.Name, leaf.ID)
	}

	// Get execution levels (parallel execution groups)
	levels, err := dag.GetExecutionLevels()
	if err != nil {
		fmt.Printf("Failed to get execution levels: %v\n", err)
		return
	}
	fmt.Println("Execution levels:")
	for i, level := range levels {
		fmt.Printf("  Level %d: %v\n", i+1, level)
	}

	// Update node status
	dag.UpdateNodeStatus("node1", NodeSucceeded)
	fmt.Println("✓ Updated node1 status to Succeeded")

	// Get nodes by status
	succeededNodes := dag.GetNodesByStatus(NodeSucceeded)
	fmt.Println("Succeeded nodes:")
	for _, node := range succeededNodes {
		fmt.Printf("  - %s (%s)\n", node.Name, node.ID)
	}

	// Get dependents of a node
	dependents := dag.GetDependents("node1")
	fmt.Println("Dependents of node1:")
	for _, dep := range dependents {
		fmt.Printf("  - %s (%s)\n", dep.Name, dep.ID)
	}
}
