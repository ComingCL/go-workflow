package main

import (
	"context"
	"fmt"
	"strings"
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

// Custom node executor with enhanced logging
type MyExecutor struct{}

func (e *MyExecutor) ExecuteWorkflowNode(ctx context.Context, data workflow.NodeData) workflow.Result {
	// Enhanced business logic with clear logging
	nodeType := "unknown"
	if nt, ok := data["nodeType"]; ok {
		nodeType = fmt.Sprintf("%v", nt)
	}

	switch nodeType {
	case "build":
		fmt.Printf("🔨 Building project with command: %v\n", data["command"])
	case "deploy":
		fmt.Printf("🚀 Deploying to target: %v\n", data["target"])
	default:
		fmt.Printf("⚙️  Executing node with data: %v\n", data)
	}

	return workflow.Result{
		Err:     nil,
		Message: fmt.Sprintf("✅ %s node executed successfully", nodeType),
	}
}

// Enhanced logger implementation
type SimpleLogger struct{}

func (l *SimpleLogger) Info(msg string, keysAndValues ...interface{}) {
	fmt.Printf("ℹ️  INFO: %s %v\n", msg, keysAndValues)
}

func (l *SimpleLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	fmt.Printf("❌ ERROR: %s - %v %v\n", msg, err, keysAndValues)
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
	fmt.Printf("💾 Updated workflow instance: %s\n", wf.UID)
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
	fmt.Printf("📝 Creating workflow instance for template: %s\n", uid)
	return nil
}

func main() {
	ctx := context.Background()

	fmt.Println("🚀 Starting Basic Workflow Demo")
	fmt.Println(strings.Repeat("=", 50))

	// Create workflow
	wf := &workflow.Workflow{
		Metadata: workflow.Metadata{
			Name:        "my-workflow",
			DisplayName: "My Basic Workflow",
			UID:         "wf-001",
			TID:         "template-001",
		},
		Spec: workflow.Spec{
			Schedule: "*/5 * * * *", // Execute every 5 minutes (5 fields format)
		},
	}

	fmt.Printf("📋 Created workflow: %s (UID: %s)\n", wf.Metadata.DisplayName, wf.Metadata.UID)

	// Create logger and cron
	logger := &SimpleLogger{}
	cronScheduler := cron.New()

	// Create controller and set repository
	controller := workflow.NewWorkflowController(cronScheduler, logger)
	repository := NewMemoryRepository()
	controller.SetRepository(repository)

	fmt.Println("🔧 Initialized workflow controller and repository")

	// Create engine
	engine, err := workflow.NewEngine(ctx, wf, controller, repository)
	if err != nil {
		fmt.Printf("❌ Failed to create engine: %v\n", err)
		return
	}

	fmt.Println("⚙️  Created workflow engine")

	// Register node executor
	executor := &MyExecutor{}
	err = engine.RegisterFunc(NodeTypeBuild, executor)
	if err != nil {
		fmt.Printf("❌ Failed to register build executor: %v\n", err)
		return
	}

	err = engine.RegisterFunc(NodeTypeDeploy, executor)
	if err != nil {
		fmt.Printf("❌ Failed to register deploy executor: %v\n", err)
		return
	}

	fmt.Println("🔌 Registered node executors (Build, Deploy)")

	// Add nodes
	fmt.Println("\n📦 Adding workflow nodes...")

	err = engine.AddWorkflowNode(workflow.AddNodeRequest{
		NodeID:   "node1",
		NodeName: "Build Node",
		NodeType: NodeTypeBuild,
		Data: map[string]interface{}{
			"command":  "go build",
			"nodeType": "build",
		},
	})
	if err != nil {
		fmt.Printf("❌ Failed to add node1: %v\n", err)
		return
	}
	fmt.Println("  ✅ Added Build Node (node1)")

	err = engine.AddWorkflowNode(workflow.AddNodeRequest{
		NodeID:   "node2",
		NodeName: "Deploy Node",
		NodeType: NodeTypeDeploy,
		Data: map[string]interface{}{
			"target":   "production",
			"nodeType": "deploy",
		},
	})
	if err != nil {
		fmt.Printf("❌ Failed to add node2: %v\n", err)
		return
	}
	fmt.Println("  ✅ Added Deploy Node (node2)")

	// Add dependency relationship (node1 -> node2)
	err = engine.AddWorkflowDependency("node1", "node2")
	if err != nil {
		fmt.Printf("❌ Failed to add dependency: %v\n", err)
		return
	}
	fmt.Println("  🔗 Added dependency: Build → Deploy")

	fmt.Println("\n" + strings.Repeat("-", 50))
	fmt.Println("🎯 Executing workflow...")
	fmt.Println(strings.Repeat("-", 50))

	// Execute workflow
	err = engine.ExecuteWorkflow(ctx)
	if err != nil {
		fmt.Printf("❌ Workflow execution failed: %v\n", err)
	} else {
		fmt.Println(strings.Repeat("-", 50))
		fmt.Println("🎉 Workflow executed successfully!")
		fmt.Println("✨ All nodes completed without errors")
	}

	fmt.Println(strings.Repeat("=", 50))
	fmt.Println("🏁 Basic workflow demo completed")
}
