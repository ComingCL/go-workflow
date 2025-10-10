package workflow

import (
	"fmt"

	"github.com/ComingCL/go-stl"
)

// WorkflowNode represents a node in the workflow DAG
type WorkflowNode struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Type         NodeType  `json:"type"`
	Status       NodePhase `json:"status"`
	Dependencies []string  `json:"dependencies"`
}

func NewWorkflowNode(id, name string, status NodePhase, nodeType NodeType) *WorkflowNode {
	return &WorkflowNode{
		ID:     id,
		Name:   name,
		Type:   nodeType,
		Status: status,
	}
}

// WorkflowDAG represents a Directed Acyclic Graph for workflow management
type WorkflowDAG struct {
	graph *stl.Graph[string]
	nodes map[string]*WorkflowNode
}

// NewWorkflowDAG creates a new workflow DAG
func NewWorkflowDAG() *WorkflowDAG {
	return &WorkflowDAG{
		graph: stl.NewGraph[string](),
		nodes: make(map[string]*WorkflowNode),
	}
}

// AddNode adds a workflow node to the DAG
func (wd *WorkflowDAG) AddNode(node *WorkflowNode) error {
	if node.ID == "" {
		return fmt.Errorf("node ID cannot be empty")
	}

	if _, exists := wd.nodes[node.ID]; exists {
		return fmt.Errorf("node with ID %s already exists", node.ID)
	}

	wd.nodes[node.ID] = node
	wd.graph.AddVertex(node.ID)

	return nil
}

// AddDependency adds a dependency relationship between two nodes
func (wd *WorkflowDAG) AddDependency(fromNodeID, toNodeID string) error {
	// Check if both nodes exist
	if _, exists := wd.nodes[fromNodeID]; !exists {
		return fmt.Errorf("source node %s does not exist", fromNodeID)
	}
	if _, exists := wd.nodes[toNodeID]; !exists {
		return fmt.Errorf("target node %s does not exist", toNodeID)
	}

	// Add edge (from -> to means "from" must complete before "to" can start)
	err := wd.graph.AddEdge(fromNodeID, toNodeID)
	if err != nil {
		return err
	}

	// Update node dependencies
	toNode := wd.nodes[toNodeID]
	toNode.Dependencies = append(toNode.Dependencies, fromNodeID)

	return nil
}

// ValidateDAG validates the workflow DAG for cycles and other issues
func (wd *WorkflowDAG) ValidateDAG() error {
	if wd.graph.HasCycle() {
		return fmt.Errorf("workflow DAG contains cycles, which is not allowed")
	}

	// Check for orphaned nodes (nodes with no connections)
	for nodeID := range wd.nodes {
		inDegree := wd.graph.GetInDegree(nodeID)
		outDegree := wd.graph.GetOutDegree(nodeID)

		if inDegree == 0 && outDegree == 0 && wd.graph.VertexCount() > 1 {
			return fmt.Errorf("node %s is orphaned (no connections)", nodeID)
		}
	}

	return nil
}

// GetExecutionOrder returns the execution order of nodes using topological sort
func (wd *WorkflowDAG) GetExecutionOrder() ([]string, error) {
	return wd.graph.TopologicalSort()
}

// GetRootNodes returns all nodes that have no dependencies (can start immediately)
func (wd *WorkflowDAG) GetRootNodes() []*WorkflowNode {
	rootIDs := wd.graph.GetRoots()
	roots := make([]*WorkflowNode, 0, len(rootIDs))

	for _, id := range rootIDs {
		if node, exists := wd.nodes[id]; exists {
			roots = append(roots, node)
		}
	}

	return roots
}

// GetLeafNodes returns all nodes that have no dependents (final nodes)
func (wd *WorkflowDAG) GetLeafNodes() []*WorkflowNode {
	leafIDs := wd.graph.GetLeaves()
	leaves := make([]*WorkflowNode, 0, len(leafIDs))

	for _, id := range leafIDs {
		if node, exists := wd.nodes[id]; exists {
			leaves = append(leaves, node)
		}
	}

	return leaves
}

// GetNodesByStatus returns all nodes with a specific status
func (wd *WorkflowDAG) GetNodesByStatus(status NodePhase) []*WorkflowNode {
	nodes := make([]*WorkflowNode, 0)

	for _, node := range wd.nodes {
		if node.Status == status {
			nodes = append(nodes, node)
		}
	}

	return nodes
}

// GetNode returns a node by its ID
func (wd *WorkflowDAG) GetNode(nodeID string) (*WorkflowNode, error) {
	if node, exists := wd.nodes[nodeID]; exists {
		return node, nil
	}
	return nil, fmt.Errorf("node with ID %s not found", nodeID)
}

// UpdateNodeStatus updates the status of a node
func (wd *WorkflowDAG) UpdateNodeStatus(nodeID string, status NodePhase) error {
	node, err := wd.GetNode(nodeID)
	if err != nil {
		return err
	}

	node.Status = status
	return nil
}

// GetDependents returns all nodes that depend on the given node
func (wd *WorkflowDAG) GetDependents(nodeID string) []*WorkflowNode {
	dependentIDs := wd.graph.GetAdjacent(nodeID)
	dependents := make([]*WorkflowNode, 0, len(dependentIDs))

	for _, id := range dependentIDs {
		if node, exists := wd.nodes[id]; exists {
			dependents = append(dependents, node)
		}
	}

	return dependents
}

// GetDependencies returns all nodes that the given node depends on
func (wd *WorkflowDAG) GetDependencies(nodeID string) []*WorkflowNode {
	node, err := wd.GetNode(nodeID)
	if err != nil {
		return []*WorkflowNode{}
	}

	dependencies := make([]*WorkflowNode, 0, len(node.Dependencies))
	for _, depID := range node.Dependencies {
		if depNode, exists := wd.nodes[depID]; exists {
			dependencies = append(dependencies, depNode)
		}
	}

	return dependencies
}

// GetExecutionLevels returns nodes grouped by execution levels (parallel execution groups)
func (wd *WorkflowDAG) GetExecutionLevels() ([][]string, error) {
	// Use a simpler approach based on in-degrees
	levels := make([][]string, 0)
	processed := make(map[string]bool)

	// Create a copy of in-degree map using public method
	inDegreeMap := make(map[string]int)
	for nodeID := range wd.nodes {
		inDegreeMap[nodeID] = wd.graph.GetInDegree(nodeID)
	}

	for len(processed) < len(wd.nodes) {
		currentLevel := make([]string, 0)

		// Find all nodes with in-degree 0 that haven't been processed
		for nodeID := range wd.nodes {
			if !processed[nodeID] && inDegreeMap[nodeID] == 0 {
				currentLevel = append(currentLevel, nodeID)
			}
		}

		if len(currentLevel) == 0 {
			// This shouldn't happen in a valid DAG, but let's handle it
			return levels, fmt.Errorf("unable to find nodes ready for execution - possible cycle")
		}

		// Mark current level nodes as processed
		for _, nodeID := range currentLevel {
			processed[nodeID] = true

			// Decrease in-degree of dependent nodes
			dependents := wd.GetDependents(nodeID)
			for _, dependent := range dependents {
				inDegreeMap[dependent.ID]--
			}
		}

		levels = append(levels, currentLevel)
	}

	return levels, nil
}

// GetCriticalPath finds the critical path (longest path) in the DAG
func (wd *WorkflowDAG) GetCriticalPath() ([]string, error) {
	order, err := wd.GetExecutionOrder()
	if err != nil {
		return nil, err
	}

	// Calculate the longest path to each node
	distance := make(map[string]int)
	parent := make(map[string]string)

	// Initialize distances
	for _, nodeID := range order {
		distance[nodeID] = 0
	}

	// Calculate longest paths
	for _, nodeID := range order {
		for _, dependent := range wd.GetDependents(nodeID) {
			newDist := distance[nodeID] + 1
			if newDist > distance[dependent.ID] {
				distance[dependent.ID] = newDist
				parent[dependent.ID] = nodeID
			}
		}
	}

	// Find the node with maximum distance
	maxDist := -1
	endNode := ""
	for nodeID, dist := range distance {
		if dist > maxDist {
			maxDist = dist
			endNode = nodeID
		}
	}

	// Reconstruct the critical path
	path := make([]string, 0)
	current := endNode
	for current != "" {
		path = append([]string{current}, path...)
		current = parent[current]
	}

	return path, nil
}

// Clone creates a deep copy of the workflow DAG
func (wd *WorkflowDAG) Clone() *WorkflowDAG {
	newDAG := NewWorkflowDAG()

	// Copy nodes
	for _, node := range wd.nodes {
		newNode := &WorkflowNode{
			ID:           node.ID,
			Name:         node.Name,
			Type:         node.Type,
			Status:       node.Status,
			Dependencies: make([]string, len(node.Dependencies)),
		}

		copy(newNode.Dependencies, node.Dependencies)

		newDAG.AddNode(newNode)
	}

	// Copy edges
	for nodeID := range wd.nodes {
		dependents := wd.GetDependents(nodeID)
		for _, dependent := range dependents {
			newDAG.AddDependency(nodeID, dependent.ID)
		}
	}

	return newDAG
}
