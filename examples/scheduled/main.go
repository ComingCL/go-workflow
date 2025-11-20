package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

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

// Scheduled task executor
type ScheduledExecutor struct{}

func (e *ScheduledExecutor) ExecuteWorkflowNode(ctx context.Context, data workflow.NodeData) workflow.Result {
	fmt.Printf("✅ Scheduled task executed at %s with data: %v\n", time.Now().Format("2006-01-02 15:04:05"), data)
	return workflow.Result{
		Err:     nil,
		Message: "Scheduled task completed successfully",
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

// Simple repository implementation for scheduled workflows
type SimpleRepository struct {
	workflows map[string]*workflow.Workflow
	mu        sync.RWMutex
}

func NewSimpleRepository() *SimpleRepository {
	return &SimpleRepository{
		workflows: make(map[string]*workflow.Workflow),
	}
}

func (r *SimpleRepository) GetWorkflowInstance(ctx context.Context, uid string) (*workflow.Workflow, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	wf, exists := r.workflows[uid]
	if !exists {
		return nil, fmt.Errorf("workflow not found: %s", uid)
	}
	return wf, nil
}

func (r *SimpleRepository) UpdateWorkflowInstance(ctx context.Context, wf *workflow.Workflow) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.workflows[wf.UID] = wf
	return nil
}

func (r *SimpleRepository) ListScheduledWorkflows(ctx context.Context) ([]*workflow.Workflow, error) {
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

func (r *SimpleRepository) CreateWorkflowInstance(ctx context.Context, uid string) error {
	// For demo purposes, just log the creation
	fmt.Printf("📝 Creating workflow instance for template: %s\n", uid)
	return nil
}

// Simple manual scheduler for demonstration
func runScheduledWorkflow(ctx context.Context, engine *workflow.WorkflowEngine, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	fmt.Printf("🕐 Starting scheduled execution every %v\n", interval)

	for {
		select {
		case <-ctx.Done():
			fmt.Println("🛑 Scheduler stopped")
			return
		case <-ticker.C:
			fmt.Printf("\n⏰ Executing scheduled workflow at %s\n", time.Now().Format("15:04:05"))
			err := engine.ExecuteWorkflow(ctx)
			if err != nil {
				fmt.Printf("❌ Workflow execution failed: %v\n", err)
			} else {
				fmt.Printf("✅ Workflow executed successfully\n")
			}
		}
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create workflow
	wf := &workflow.Workflow{
		Metadata: workflow.Metadata{
			Name:        "scheduled-workflow",
			DisplayName: "Scheduled Workflow Demo",
			UID:         "scheduled-001",
			TID:         "scheduled-template-001",
		},
		Spec: workflow.Spec{
			Schedule: "*/10 * * * * *", // Every 10 seconds for demo
		},
	}

	// Create logger and cron
	logger := &SimpleLogger{}
	cronScheduler := cron.New()

	// Create controller and set repository
	controller := workflow.NewWorkflowController(cronScheduler, logger)
	repository := NewSimpleRepository()
	controller.SetRepository(repository)

	// Create engine
	engine, err := workflow.NewEngine(ctx, wf, controller, repository)
	if err != nil {
		fmt.Printf("❌ Failed to create engine: %v\n", err)
		return
	}

	// Register node executor
	executor := &ScheduledExecutor{}
	err = engine.RegisterFunc(NodeTypeBuild, executor)
	if err != nil {
		fmt.Printf("❌ Failed to register executor: %v\n", err)
		return
	}

	// Add a sample node to the workflow
	err = engine.AddWorkflowNode(workflow.AddNodeRequest{
		NodeID:   "task1",
		NodeName: "Scheduled Task",
		NodeType: NodeTypeBuild,
		Data: map[string]interface{}{
			"message":   "Hello from scheduled workflow",
			"timestamp": time.Now().Format("15:04:05"),
			"counter":   0,
		},
	})
	if err != nil {
		fmt.Printf("❌ Failed to add node: %v\n", err)
		return
	}

	fmt.Println("🚀 Starting scheduled workflow demo...")
	fmt.Println("📋 Workflow will execute every 10 seconds")
	fmt.Println("⏹️  Press Ctrl+C to stop")
	fmt.Println(strings.Repeat("-", 50))

	// Run our simple scheduler in a goroutine
	go runScheduledWorkflow(ctx, engine, 10*time.Second)

	// Keep the program running for demo
	time.Sleep(30 * time.Second)

	fmt.Println(strings.Repeat("-", 50))
	fmt.Println("🏁 Demo completed, stopping scheduler...")
	cancel()
	time.Sleep(1 * time.Second)
}
