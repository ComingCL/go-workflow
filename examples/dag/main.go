package main

import (
	"fmt"
	"strings"

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

func printHeader(title string) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Printf("🔍 %s\n", title)
	fmt.Println(strings.Repeat("=", 60))
}

func printSection(title string) {
	fmt.Printf("\n📋 %s\n", title)
	fmt.Println(strings.Repeat("-", 40))
}

func main() {
	printHeader("DAG (Directed Acyclic Graph) Management Demo")

	// Create DAG
	fmt.Println("🏗️  Creating new WorkflowDAG...")
	dag := workflow.NewWorkflowDAG()
	fmt.Println("✅ DAG created successfully")

	printSection("Adding Nodes to DAG")

	// Add nodes with enhanced logging
	nodes := []struct {
		id       string
		name     string
		nodeType workflow.NodeType
		emoji    string
	}{
		{"node1", "Start Node", NodeTypeStart, "🟢"},
		{"node2", "Build Node", NodeTypeBuild, "🔨"},
		{"node3", "Test Node", NodeTypeBuild, "🧪"},
		{"node4", "Deploy Node", NodeTypeDeploy, "🚀"},
		{"node5", "End Node", NodeTypeEnd, "🏁"},
	}

	for _, n := range nodes {
		node := workflow.NewWorkflowNode(n.id, n.name, NodePending, n.nodeType)
		dag.AddNode(node)
		fmt.Printf("  %s Added: %s (%s) - Type: %s\n", n.emoji, n.name, n.id, n.nodeType)
	}

	printSection("Building DAG Dependencies")

	dependencies := []struct {
		from, to string
		desc     string
	}{
		{"node1", "node2", "Start → Build"},
		{"node1", "node3", "Start → Test"},
		{"node2", "node4", "Build → Deploy"},
		{"node3", "node4", "Test → Deploy"},
		{"node4", "node5", "Deploy → End"},
	}

	fmt.Println("🔗 Adding dependencies:")
	for _, dep := range dependencies {
		dag.AddDependency(dep.from, dep.to)
		fmt.Printf("  ✅ %s\n", dep.desc)
	}

	printSection("DAG Validation")

	// Validate DAG
	err := dag.ValidateDAG()
	if err != nil {
		fmt.Printf("❌ DAG validation failed: %v\n", err)
		return
	}
	fmt.Println("✅ DAG validation passed - No cycles detected!")

	printSection("Execution Analysis")

	// Get execution order
	order, err := dag.GetExecutionOrder()
	if err != nil {
		fmt.Printf("❌ Failed to get execution order: %v\n", err)
		return
	}
	fmt.Printf("📊 Execution order: %v\n", order)

	// Get root nodes (nodes with no dependencies)
	roots := dag.GetRootNodes()
	fmt.Printf("\n🌱 Root nodes (entry points):\n")
	for _, root := range roots {
		fmt.Printf("  🟢 %s (%s)\n", root.Name, root.ID)
	}

	// Get leaf nodes (nodes with no dependents)
	leaves := dag.GetLeafNodes()
	fmt.Printf("\n🍃 Leaf nodes (exit points):\n")
	for _, leaf := range leaves {
		fmt.Printf("  🏁 %s (%s)\n", leaf.Name, leaf.ID)
	}

	// Get execution levels (parallel execution groups)
	levels, err := dag.GetExecutionLevels()
	if err != nil {
		fmt.Printf("❌ Failed to get execution levels: %v\n", err)
		return
	}
	fmt.Printf("\n🔄 Execution levels (parallel groups):\n")
	for i, level := range levels {
		fmt.Printf("  Level %d: %v\n", i+1, level)
		if i == 0 {
			fmt.Printf("    └─ 🟢 Can start immediately\n")
		} else {
			fmt.Printf("    └─ ⏳ Waits for Level %d completion\n", i)
		}
	}

	printSection("Node Status Management")

	// Update node status
	fmt.Println("🔄 Simulating node execution...")
	dag.UpdateNodeStatus("node1", NodeSucceeded)
	fmt.Println("  ✅ Updated node1 (Start Node) status to Succeeded")

	// Get nodes by status
	succeededNodes := dag.GetNodesByStatus(NodeSucceeded)
	fmt.Printf("\n🎉 Succeeded nodes:\n")
	for _, node := range succeededNodes {
		fmt.Printf("  ✅ %s (%s)\n", node.Name, node.ID)
	}

	// Get dependents of a node
	dependents := dag.GetDependents("node1")
	fmt.Printf("\n🔗 Dependents of node1 (nodes that can now execute):\n")
	for _, dep := range dependents {
		fmt.Printf("  ➡️  %s (%s)\n", dep.Name, dep.ID)
	}

	printSection("DAG Structure Summary")

	fmt.Printf("📊 Total nodes: %d\n", len(nodes))
	fmt.Printf("🔗 Total dependencies: %d\n", len(dependencies))
	fmt.Printf("🌱 Root nodes: %d\n", len(roots))
	fmt.Printf("🍃 Leaf nodes: %d\n", len(leaves))
	fmt.Printf("🔄 Execution levels: %d\n", len(levels))

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("🎯 DAG Demo completed successfully!")
	fmt.Println("💡 This demonstrates how to build and analyze workflow dependencies")
	fmt.Println(strings.Repeat("=", 60))
}
