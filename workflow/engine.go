package workflow

import (
	"context"
	"fmt"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type WorkflowEngine struct {
	ctx context.Context
	wf  *Workflow
	woc *WorkflowController

	// Node type executor mapping: nodeType -> executor function
	executorMap map[NodeType]NodeExecutor

	// DAG management for graph operations
	dagExecutor *WorkflowDAG

	// Execution state mutex
	mu sync.RWMutex

	// Repository workflow storage Repository
	Repository WorkflowRepository
}

type Result struct {
	Err     error
	Message string
}

// NodeExecutor defines the node executor interface
type NodeExecutor interface {
	ExecuteWorkflowNode(ctx context.Context, data NodeData) Result
}

func NewEngine(ctx context.Context, wf *Workflow, w *WorkflowController, repository WorkflowRepository) (*WorkflowEngine, error) {
	engine := &WorkflowEngine{
		ctx:         ctx,
		wf:          wf,
		woc:         w,
		executorMap: make(map[NodeType]NodeExecutor),
		dagExecutor: NewWorkflowDAG(),
		Repository:  repository,
	}

	return engine, engine.initializeDAGFromWorkflow()
}

func (oc *WorkflowEngine) GetWorkflow() *Workflow {
	return oc.wf
}

// registerNodeTypeExecutor registers an executor for a specific node type
func (oc *WorkflowEngine) registerNodeTypeExecutor(nodeType NodeType, executor NodeExecutor) error {
	oc.mu.Lock()
	defer oc.mu.Unlock()

	if _, exists := oc.executorMap[nodeType]; exists {
		return fmt.Errorf("executor for node type %s already registered", nodeType)
	}

	oc.executorMap[nodeType] = executor
	oc.woc.logger.Info("[WorkflowEngine] Registered executor for node type", "nodeType", nodeType)
	return nil
}

// AddWorkflowNode adds a workflow node to both DAG and workflow
func (oc *WorkflowEngine) AddWorkflowNode(nodeID, nodeName string, nodeType NodeType, data NodeData) error {
	oc.mu.Lock()
	defer oc.mu.Unlock()

	// Add to DAG
	if err := oc.dagExecutor.AddNode(&WorkflowNode{
		ID:   nodeID,
		Name: nodeName,
		Type: nodeType,
	}); err != nil {
		return fmt.Errorf("failed to add node to DAG: %v", err)
	}

	// Add to workflow status
	nodeStatus := NodeStatus{
		ID:          nodeID,
		Name:        nodeName,
		Type:        nodeType,
		Phase:       NodePending,
		Data:        data,
		DisplayName: oc.wf.Name,
		Children:    []string{},
	}

	if oc.wf.Status.Nodes == nil {
		oc.wf.Status.Nodes = make(Nodes)
	}
	oc.wf.Status.Nodes.Set(nodeID, nodeStatus)

	oc.woc.logger.Info("[WorkflowEngine] Added workflow node", "nodeID", nodeID, "nodeType", nodeType)
	return nil
}

func (oc *WorkflowEngine) SetWorkflowNodeInputs(nodeID string, data map[string]interface{}) error {
	oc.mu.Lock()
	defer oc.mu.Unlock()
	status, err := oc.wf.Status.Nodes.Get(nodeID)
	if err != nil {
		return err
	}
	status.Data = data
	return nil
}

// AddWorkflowDependency adds a dependency relationship between nodes, toNodeID depends on fromNodeID
func (oc *WorkflowEngine) AddWorkflowDependency(fromNodeID, toNodeID string) error {
	oc.mu.Lock()
	defer oc.mu.Unlock()

	// Add to DAG
	if err := oc.dagExecutor.AddDependency(fromNodeID, toNodeID); err != nil {
		return fmt.Errorf("failed to add dependency to DAG: %v", err)
	}

	// Update children relationship in workflow
	fromNode, err := oc.wf.Status.Nodes.Get(fromNodeID)
	if err != nil {
		return fmt.Errorf("source node not found: %v", err)
	}

	// Check if dependency already exists
	for _, childID := range fromNode.Children {
		if childID == toNodeID {
			return nil // Already exists, don't add duplicate
		}
	}

	fromNode.Children = append(fromNode.Children, toNodeID)
	oc.wf.Status.Nodes.Set(fromNodeID, *fromNode)

	oc.woc.logger.Info("[WorkflowEngine] Added dependency", "from", fromNodeID, "to", toNodeID)
	return nil
}

// RemoveNode removes a node from both DAG and workflow
func (oc *WorkflowEngine) RemoveNode(nodeID string) error {
	oc.mu.Lock()
	defer oc.mu.Unlock()

	// Remove from workflow
	delete(oc.wf.Status.Nodes, nodeID)

	// Remove from DAG (call WorkflowDAG method directly)
	oc.dagExecutor.graph.RemoveVertex(nodeID)

	return nil
}

// RemoveDependency removes dependency relationship between nodes
func (oc *WorkflowEngine) RemoveDependency(fromNodeID, toNodeID string) error {
	oc.mu.Lock()
	defer oc.mu.Unlock()

	// Remove dependency from DAG
	err := oc.dagExecutor.graph.RemoveEdge(fromNodeID, toNodeID)
	if err != nil {
		return err
	}

	// Remove children relationship from workflow
	if fromNode, err := oc.wf.Status.Nodes.Get(fromNodeID); err == nil {
		newChildren := make([]string, 0)
		for _, childID := range fromNode.Children {
			if childID != toNodeID {
				newChildren = append(newChildren, childID)
			}
		}
		fromNode.Children = newChildren
		oc.wf.Status.Nodes.Set(fromNodeID, *fromNode)
	}

	return nil
}

// ExecuteWorkflow executes the entire workflow
func (oc *WorkflowEngine) ExecuteWorkflow(ctx context.Context) error {
	oc.woc.logger.Info("[WorkflowEngine] Starting workflow execution", "workflowName", oc.wf.Metadata.Name)

	// Validate DAG
	if err := oc.dagExecutor.ValidateDAG(); err != nil {
		return fmt.Errorf("DAG validate failed: %v", err)
	}

	// Update workflow status to running
	oc.wf.Status.Phase = Running

	// ExecuteWorkflowNode DAG using the new DAG execution method
	{
		err := oc.executeDAG(ctx)
		if err != nil {
			oc.woc.logger.Error(err, "[WorkflowEngine] Failed to execute DAG")
		}
	}
	// Check if all nodes completed successfully
	if oc.isWorkflowCompleted() {
		oc.wf.Status.Phase = Succeeded
		oc.wf.Status.Message = "[WorkflowEngine] Workflow execute succeeded"
	} else {
		oc.wf.Status.Phase = Failed
		oc.wf.Status.Message = "[WorkflowEngine] Workflow execute failed"
	}

	// Update record
	return oc.Repository.UpdateWorkflowInstance(ctx, oc.wf)
}

// executeDAG executes the DAG with DFS-based parallel execution
func (oc *WorkflowEngine) executeDAG(ctx context.Context) error {
	oc.woc.logger.Info("[WorkflowEngine] Starting DFS-based DAG execution")

	// Get root nodes (nodes with no dependencies)
	rootNodes := oc.dagExecutor.GetRootNodes()
	if len(rootNodes) == 0 {
		oc.woc.logger.Info("[WorkflowEngine] No root nodes found")
		return nil
	}

	oc.woc.logger.Info("[WorkflowEngine] Found root nodes", "count", len(rootNodes))

	// ExecuteWorkflowNode from all root nodes in parallel
	return oc.executeRootNodes(ctx, rootNodes)
}

// executeRootNodes executes all root nodes in parallel
func (oc *WorkflowEngine) executeRootNodes(ctx context.Context, rootNodes []*WorkflowNode) error {
	if len(rootNodes) == 1 {
		return oc.executeDFSFromNode(ctx, rootNodes[0])
	}

	// ExecuteWorkflowNode multiple root nodes in parallel
	var wg sync.WaitGroup
	errChan := make(chan error, len(rootNodes))

	for _, rootNode := range rootNodes {
		wg.Add(1)
		go func(node *WorkflowNode) {
			defer wg.Done()
			if err := oc.executeDFSFromNode(ctx, node); err != nil {
				errChan <- err
			}
		}(rootNode)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// executeDFSFromNode executes DFS traversal starting from a specific node
func (oc *WorkflowEngine) executeDFSFromNode(ctx context.Context, node *WorkflowNode) error {
	// ExecuteWorkflowNode current node
	if err := oc.executeNode(ctx, node.ID); err != nil {
		return fmt.Errorf("node %s execution failed: %v", node.ID, err)
	}

	// Get dependent nodes (children)
	dependents := oc.dagExecutor.GetDependents(node.ID)
	if len(dependents) == 0 {
		oc.woc.logger.Info("[WorkflowEngine] Node completed (leaf node)", "nodeID", node.ID)
		return nil
	}

	// Find ready dependents (all dependencies completed)
	readyDependents := oc.getReadyDependents(dependents)
	if len(readyDependents) == 0 {
		oc.woc.logger.Info("[WorkflowEngine] Node completed (no ready dependents)", "nodeID", node.ID)
		return nil
	}

	oc.woc.logger.Info("[WorkflowEngine] Executing dependents", "nodeID", node.ID, "dependentCount", len(readyDependents))

	// ExecuteWorkflowNode ready dependents
	return oc.executeReadyDependents(ctx, readyDependents)
}

// executeReadyDependents executes ready dependent nodes (parallel if multiple, serial if single)
func (oc *WorkflowEngine) executeReadyDependents(ctx context.Context, readyDependents []*WorkflowNode) error {
	if len(readyDependents) == 1 {
		// Single dependent - execute serially
		return oc.executeDFSFromNode(ctx, readyDependents[0])
	}

	// Multiple dependents - execute in parallel
	var wg sync.WaitGroup
	errChan := make(chan error, len(readyDependents))

	for _, dependent := range readyDependents {
		wg.Add(1)
		go func(dep *WorkflowNode) {
			defer wg.Done()
			if err := oc.executeDFSFromNode(ctx, dep); err != nil {
				errChan <- err
			}
		}(dependent)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// getReadyDependents filters dependents that are ready to execute (all dependencies completed)
func (oc *WorkflowEngine) getReadyDependents(dependents []*WorkflowNode) []*WorkflowNode {
	readyDependents := make([]*WorkflowNode, 0)

	for _, dependent := range dependents {
		if oc.areAllDependenciesCompleted(dependent) {
			readyDependents = append(readyDependents, dependent)
		}
	}

	return readyDependents
}

// areAllDependenciesCompleted checks if all dependencies of a node are completed
func (oc *WorkflowEngine) areAllDependenciesCompleted(node *WorkflowNode) bool {
	for _, depID := range node.Dependencies {
		if depNode, err := oc.dagExecutor.GetNode(depID); err == nil {
			if !depNode.Status.Completed() {
				return false
			}
		} else {
			return false // Dependency not found
		}
	}
	return true
}

// executeNode executes a single node
func (oc *WorkflowEngine) executeNode(ctx context.Context, nodeID string) error {
	oc.mu.Lock()
	node, err := oc.wf.Status.Nodes.Get(nodeID)
	if err != nil {
		oc.mu.Unlock()
		return fmt.Errorf("node %s not found: %v", nodeID, err)
	}

	nodeType := node.Type
	executor, exists := oc.executorMap[nodeType]
	if !exists {
		oc.mu.Unlock()
		return fmt.Errorf("no executor registered for node type: %s", nodeType)
	}
	oc.mu.Unlock()

	oc.woc.logger.Info("[WorkflowEngine] Executing node", "nodeID", nodeID, "nodeType", nodeType)

	now := metav1.Now()
	// Update node status to running
	if err = oc.updateNodeStatus(nodeID, Params{
		Phase: NodeRunning,
	}); err != nil {
		return err
	}

	res := executor.ExecuteWorkflowNode(ctx, node.Data)
	phase := NodeSucceeded
	// ExecuteWorkflowNode the node
	if err = res.Err; err != nil {
		// Execution failed, update status
		phase = NodeFailed
	}

	// Execution succeeded, update status
	if err = oc.updateNodeStatus(nodeID, Params{
		Phase:      phase,
		Message:    &res.Message,
		FinishedAt: &now,
	}); err != nil {
		return err
	}
	oc.woc.logger.Info("[WorkflowEngine] Node executed successfully", "nodeID", nodeID)

	return nil
}

type Params struct {
	// required
	Phase      NodePhase
	Message    *string
	StartedAt  *metav1.Time
	FinishedAt *metav1.Time
}

// updateNodeStatus updates the node status
func (oc *WorkflowEngine) updateNodeStatus(nodeID string, params Params) error {
	oc.mu.Lock()
	defer oc.mu.Unlock()

	if node, err := oc.wf.Status.Nodes.Get(nodeID); err == nil {
		node.Phase = params.Phase
		if params.Message != nil {
			node.Message = *params.Message
		}
		if params.StartedAt != nil {
			node.StartedAt = *params.StartedAt
		}
		if params.FinishedAt != nil {
			node.FinishedAt = *params.FinishedAt
		}
		oc.wf.Status.Nodes.Set(nodeID, *node)

		// Sync update status in DAG
		if err = oc.dagExecutor.UpdateNodeStatus(nodeID, params.Phase); err != nil {
			return err
		}
		if err = oc.Repository.UpdateWorkflowInstance(context.TODO(), oc.wf); err != nil {
			return fmt.Errorf("WorkflowEngine update workflow instance failed: %w", err)
		}
	}

	return nil
}

// isWorkflowCompleted checks if the workflow is completed
func (oc *WorkflowEngine) isWorkflowCompleted() bool {
	oc.mu.RLock()
	defer oc.mu.RUnlock()

	for _, node := range oc.wf.Status.Nodes {
		if !node.Phase.Completed() {
			return false
		}
	}
	return true
}

// initializeDAGFromWorkflow initializes DAG from existing workflow
func (oc *WorkflowEngine) initializeDAGFromWorkflow() error {
	if oc.wf.Status.Nodes == nil {
		return nil
	}
	var err error

	// Add all nodes to DAG
	for nodeID, nodeStatus := range oc.wf.Status.Nodes {
		if err = oc.dagExecutor.AddNode(NewWorkflowNode(nodeID, nodeStatus.Name, nodeStatus.Phase, nodeStatus.Type)); err != nil {
			return err
		}
	}

	// Add dependency relationships
	for nodeID, nodeStatus := range oc.wf.Status.Nodes {
		for _, childID := range nodeStatus.Children {
			if err = oc.dagExecutor.AddDependency(nodeID, childID); err != nil {
				return err
			}
		}
	}

	return nil
}

// GetWorkflowStatus gets the workflow status
func (oc *WorkflowEngine) GetWorkflowStatus() *Status {
	oc.mu.RLock()
	defer oc.mu.RUnlock()
	return &oc.wf.Status
}

// GetNodeStatus gets the node status
func (oc *WorkflowEngine) GetNodeStatus(nodeID string) (*NodeStatus, error) {
	oc.mu.RLock()
	defer oc.mu.RUnlock()
	return oc.wf.Status.Nodes.Get(nodeID)
}

// Legacy methods for backward compatibility

// RegisterFunc registers a method for legacy interface compatibility (deprecated, use RegisterNodeTypeExecutor instead)
func (oc *WorkflowEngine) RegisterFunc(key NodeType, f NodeExecutor) error {
	return oc.registerNodeTypeExecutor(key, f)
}

func (oc *WorkflowEngine) operator(ctx context.Context) {
	err := oc.ExecuteWorkflow(ctx)
	if err != nil {
		oc.woc.logger.Error(err, "[WorkflowEngine] execute workflow failed")
		return
	}
}
