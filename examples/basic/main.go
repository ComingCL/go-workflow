package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/ComingCL/go-workflow/workflow"
	"github.com/robfig/cron/v3"
)

// Node types - define locally since they might be missing from the package
const (
	NodeTypeStart   workflow.NodeType = "start"
	NodeTypeEnd     workflow.NodeType = "end"
	NodeTypeDeploy  workflow.NodeType = "deploy"
	NodeTypeApiCall workflow.NodeType = "api-call"
	NodeTypeBuild   workflow.NodeType = "build"
)

// Custom node executor
type MyExecutor struct{}

func (e *MyExecutor) ExecuteWorkflowNode(ctx context.Context, data workflow.NodeData) workflow.Result {
	// Implement your business logic here
	fmt.Println("Executing node:", data)
	return workflow.Result{
		Err:     nil,
		Message: "Node executed successfully",
	}
}

// Simple logger implementation
type SimpleLogger struct{}

func (l *SimpleLogger) Info(msg string, keysAndValues ...interface{}) {
	fmt.Printf("INFO: %s %v\n", msg, keysAndValues)
}

func (l *SimpleLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	fmt.Printf("ERROR: %s - %v %v\n", msg, err, keysAndValues)
}

// Example in-memory repository implementation
type MemoryRepository struct {
	workflows map[string]*workflow.Workflow
	mu        sync.RWMutex
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		workflows: make(map[string]*workflow.Workflow),
	}
}

func (r *MemoryRepository) GetWorkflowInstance(ctx context.Context, uid string) (*workflow.Workflow, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	wf, exists := r.workflows[uid]
	if !exists {
		return nil, fmt.Errorf("workflow not found: %s", uid)
	}
	return wf, nil
}

func (r *MemoryRepository) UpdateWorkflowInstance(ctx context.Context, wf *workflow.Workflow) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.workflows[wf.UID] = wf
	return nil
}

func (r *MemoryRepository) ListScheduledWorkflows(ctx context.Context) ([]*workflow.Workflow, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	workflows := make([]*workflow.Workflow, 0, len(r.workflows))
	for _, wf := range r.workflows {
		if wf.IsScheduled() {
			workflows = append(workflows, wf)
		}
	}
	return workflows, nil
}

func (r *MemoryRepository) CreateWorkflowInstance(ctx context.Context, uid string) error {
	// For this example, we'll just return nil
	// In a real implementation, this would create a new instance
	return nil
}

func main() {
	ctx := context.Background()

	// Create workflow
	wf := &workflow.Workflow{
		Metadata: workflow.Metadata{
			Name:        "my-workflow",
			DisplayName: "My Workflow",
			UID:         "wf-001",
			TID:         "template-001",
		},
		Spec: workflow.Spec{
			Schedule: "*/5 * * * *", // Execute every 5 minutes (5 fields format)
		},
	}

	// Create logger and cron
	logger := &SimpleLogger{}
	cronScheduler := cron.New()

	// Create controller and set repository
	controller := workflow.NewWorkflowController(cronScheduler, logger)
	repository := NewMemoryRepository()
	controller.SetRepository(repository)

	// Create engine
	engine, err := workflow.NewEngine(ctx, wf, controller, repository)
	if err != nil {
		panic(err)
	}

	// Register node executor
	executor := &MyExecutor{}
	err = engine.RegisterFunc(NodeTypeBuild, executor)
	if err != nil {
		fmt.Printf("Failed to register executor: %v\n", err)
		return
	}

	err = engine.RegisterFunc(NodeTypeDeploy, executor)
	if err != nil {
		fmt.Printf("Failed to register executor: %v\n", err)
		return
	}

	// Add nodes
	err = engine.AddWorkflowNode("node1", "Build Node", NodeTypeBuild, map[string]interface{}{
		"command": "go build",
	})
	if err != nil {
		fmt.Printf("Failed to add node1: %v\n", err)
		return
	}

	err = engine.AddWorkflowNode("node2", "Deploy Node", NodeTypeDeploy, map[string]interface{}{
		"target": "production",
	})
	if err != nil {
		fmt.Printf("Failed to add node2: %v\n", err)
		return
	}

	// Add dependency relationship (node1 -> node2)
	err = engine.AddWorkflowDependency("node1", "node2")
	if err != nil {
		fmt.Printf("Failed to add dependency: %v\n", err)
		return
	}

	// Execute workflow
	err = engine.ExecuteWorkflow(ctx)
	if err != nil {
		fmt.Printf("Workflow execution failed: %v\n", err)
	} else {
		fmt.Println("Workflow executed successfully!")
	}
}
