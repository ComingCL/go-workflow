package workflow

import (
	"testing"
)

func TestNewWorkflowDAG(t *testing.T) {
	dag := NewWorkflowDAG()

	if dag == nil {
		t.Fatal("NewWorkflowDAG() returned nil")
	}

	if dag.graph == nil {
		t.Error("graph should not be nil")
	}

	if dag.nodes == nil {
		t.Error("nodes map should not be nil")
	}

	if len(dag.nodes) != 0 {
		t.Error("new DAG should have no nodes")
	}
}

func TestAddNode(t *testing.T) {
	dag := NewWorkflowDAG()

	// Test adding valid node
	node := &WorkflowNode{
		ID:           "test1",
		Name:         "Test Node 1",
		Type:         "test",
		Status:       "pending",
		Dependencies: []string{},
	}

	err := dag.AddNode(node)
	if err != nil {
		t.Errorf("AddNode() failed: %v", err)
	}

	// Test duplicate node
	err = dag.AddNode(node)
	if err == nil {
		t.Error("AddNode() should fail for duplicate node ID")
	}

	// Test empty ID
	emptyNode := &WorkflowNode{ID: ""}
	err = dag.AddNode(emptyNode)
	if err == nil {
		t.Error("AddNode() should fail for empty node ID")
	}
}

func TestAddDependency(t *testing.T) {
	dag := NewWorkflowDAG()

	// Add nodes
	node1 := &WorkflowNode{ID: "node1", Name: "Node 1", Type: "test"}
	node2 := &WorkflowNode{ID: "node2", Name: "Node 2", Type: "test"}

	dag.AddNode(node1)
	dag.AddNode(node2)

	// Test valid dependency
	err := dag.AddDependency("node1", "node2")
	if err != nil {
		t.Errorf("AddDependency() failed: %v", err)
	}

	// Test duplicate dependency
	err = dag.AddDependency("node1", "node2")
	if err == nil {
		t.Error("AddDependency() should fail for duplicate edge")
	}

	// Test non-existent source node
	err = dag.AddDependency("nonexistent", "node2")
	if err == nil {
		t.Error("AddDependency() should fail for non-existent source node")
	}

	// Test non-existent target node
	err = dag.AddDependency("node1", "nonexistent")
	if err == nil {
		t.Error("AddDependency() should fail for non-existent target node")
	}
}

func TestValidateDAG(t *testing.T) {
	dag := NewWorkflowDAG()

	// Add nodes
	nodes := []*WorkflowNode{
		{ID: "a", Name: "A", Type: "test"},
		{ID: "b", Name: "B", Type: "test"},
		{ID: "c", Name: "C", Type: "test"},
	}

	for _, node := range nodes {
		dag.AddNode(node)
	}

	// Test valid DAG
	dag.AddDependency("a", "b")
	dag.AddDependency("b", "c")

	err := dag.ValidateDAG()
	if err != nil {
		t.Errorf("ValidateDAG() failed for valid DAG: %v", err)
	}

	// Test cycle detection
	dag.AddDependency("c", "a") // Creates a cycle

	err = dag.ValidateDAG()
	if err == nil {
		t.Error("ValidateDAG() should detect cycle")
	}
}

func TestTopologicalSort(t *testing.T) {
	dag := NewWorkflowDAG()

	// Create a simple DAG: A -> B -> C, A -> D -> C
	nodes := []*WorkflowNode{
		{ID: "a", Name: "A", Type: "test"},
		{ID: "b", Name: "B", Type: "test"},
		{ID: "c", Name: "C", Type: "test"},
		{ID: "d", Name: "D", Type: "test"},
	}

	for _, node := range nodes {
		dag.AddNode(node)
	}

	dag.AddDependency("a", "b")
	dag.AddDependency("a", "d")
	dag.AddDependency("b", "c")
	dag.AddDependency("d", "c")

	order, err := dag.GetExecutionOrder()
	if err != nil {
		t.Errorf("GetExecutionOrder() failed: %v", err)
	}

	if len(order) != 4 {
		t.Errorf("Expected 4 nodes in execution order, got %d", len(order))
	}

	// Check that 'a' comes before 'b' and 'd'
	aIndex := findIndex(order, "a")
	bIndex := findIndex(order, "b")
	dIndex := findIndex(order, "d")
	cIndex := findIndex(order, "c")

	if aIndex == -1 || bIndex == -1 || dIndex == -1 || cIndex == -1 {
		t.Error("All nodes should be in execution order")
	}

	if aIndex >= bIndex || aIndex >= dIndex {
		t.Error("Node 'a' should come before 'b' and 'd'")
	}

	if bIndex >= cIndex || dIndex >= cIndex {
		t.Error("Nodes 'b' and 'd' should come before 'c'")
	}
}

func TestGetRootAndLeafNodes(t *testing.T) {
	dag := NewWorkflowDAG()

	// Create DAG: A -> B -> C
	nodes := []*WorkflowNode{
		{ID: "a", Name: "A", Type: "test"},
		{ID: "b", Name: "B", Type: "test"},
		{ID: "c", Name: "C", Type: "test"},
	}

	for _, node := range nodes {
		dag.AddNode(node)
	}

	dag.AddDependency("a", "b")
	dag.AddDependency("b", "c")

	// Test root nodes
	roots := dag.GetRootNodes()
	if len(roots) != 1 || roots[0].ID != "a" {
		t.Errorf("Expected root node 'a', got %v", roots)
	}

	// Test leaf nodes
	leaves := dag.GetLeafNodes()
	if len(leaves) != 1 || leaves[0].ID != "c" {
		t.Errorf("Expected leaf node 'c', got %v", leaves)
	}
}

func TestGetExecutionLevels(t *testing.T) {
	dag := NewWorkflowDAG()

	// Create DAG with parallel branches: A -> B, A -> C, B -> D, C -> D
	nodes := []*WorkflowNode{
		{ID: "a", Name: "A", Type: "test"},
		{ID: "b", Name: "B", Type: "test"},
		{ID: "c", Name: "C", Type: "test"},
		{ID: "d", Name: "D", Type: "test"},
	}

	for _, node := range nodes {
		dag.AddNode(node)
	}

	dag.AddDependency("a", "b")
	dag.AddDependency("a", "c")
	dag.AddDependency("b", "d")
	dag.AddDependency("c", "d")

	levels, err := dag.GetExecutionLevels()
	if err != nil {
		t.Errorf("GetExecutionLevels() failed: %v", err)
	}

	if len(levels) != 3 {
		t.Errorf("Expected 3 execution levels, got %d", len(levels))
	}

	// Level 0: should contain 'a'
	if len(levels[0]) != 1 || levels[0][0] != "a" {
		t.Errorf("Level 0 should contain 'a', got %v", levels[0])
	}

	// Level 1: should contain 'b' and 'c' (parallel)
	if len(levels[1]) != 2 {
		t.Errorf("Level 1 should contain 2 nodes, got %d", len(levels[1]))
	}

	// Level 2: should contain 'd'
	if len(levels[2]) != 1 || levels[2][0] != "d" {
		t.Errorf("Level 2 should contain 'd', got %v", levels[2])
	}
}

func TestGetCriticalPath(t *testing.T) {
	dag := NewWorkflowDAG()

	// Create DAG: A -> B -> D, A -> C -> D (B->D is longer path)
	nodes := []*WorkflowNode{
		{ID: "a", Name: "A", Type: "test"},
		{ID: "b", Name: "B", Type: "test"},
		{ID: "c", Name: "C", Type: "test"},
		{ID: "d", Name: "D", Type: "test"},
	}

	for _, node := range nodes {
		dag.AddNode(node)
	}

	dag.AddDependency("a", "b")
	dag.AddDependency("a", "c")
	dag.AddDependency("b", "d")
	dag.AddDependency("c", "d")

	path, err := dag.GetCriticalPath()
	if err != nil {
		t.Errorf("GetCriticalPath() failed: %v", err)
	}

	// Critical path should be A -> B -> D or A -> C -> D (both have same length)
	if len(path) != 3 {
		t.Errorf("Expected critical path length 3, got %d", len(path))
	}

	if path[0] != "a" || path[2] != "d" {
		t.Errorf("Critical path should start with 'a' and end with 'd', got %v", path)
	}
}

func TestClone(t *testing.T) {
	dag := NewWorkflowDAG()

	// Create original DAG
	nodes := []*WorkflowNode{
		{ID: "a", Name: "A", Type: "test", Status: "completed"},
		{ID: "b", Name: "B", Type: "test", Status: "pending"},
	}

	for _, node := range nodes {
		dag.AddNode(node)
	}
	dag.AddDependency("a", "b")

	// Clone the DAG
	cloned := dag.Clone()

	// Verify clone has same structure
	if cloned.graph.VertexCount() != dag.graph.VertexCount() {
		t.Error("Cloned DAG should have same number of vertices")
	}

	if cloned.graph.EdgeCount() != dag.graph.EdgeCount() {
		t.Error("Cloned DAG should have same number of edges")
	}

	// Verify nodes are copied (not referenced)
	originalNode, _ := dag.GetNode("a")
	clonedNode, _ := cloned.GetNode("a")

	if originalNode == clonedNode {
		t.Error("Cloned nodes should not be the same reference")
	}

	if originalNode.Status != clonedNode.Status {
		t.Error("Cloned node should have same status")
	}

	// Modify original and verify clone is unaffected
	dag.UpdateNodeStatus("a", "failed")
	if clonedNode.Status != "completed" {
		t.Error("Cloned node should not be affected by original modifications")
	}
}

// Helper functions
func findIndex(slice []string, item string) int {
	for i, v := range slice {
		if v == item {
			return i
		}
	}
	return -1
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			findSubstring(s, substr))))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
