package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ComingCL/go-workflow/workflow"
	"github.com/robfig/cron/v3"
)

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
}

func NewSimpleRepository() *SimpleRepository {
	return &SimpleRepository{
		workflows: make(map[string]*workflow.Workflow),
	}
}

func (r *SimpleRepository) GetWorkflowInstance(ctx context.Context, uid string) (*workflow.Workflow, error) {
	wf, exists := r.workflows[uid]
	if !exists {
		return nil, fmt.Errorf("workflow not found: %s", uid)
	}
	return wf, nil
}

func (r *SimpleRepository) UpdateWorkflowInstance(ctx context.Context, wf *workflow.Workflow) error {
	r.workflows[wf.TID] = wf
	return nil
}

func (r *SimpleRepository) ListScheduledWorkflows(ctx context.Context) ([]*workflow.Workflow, error) {
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
	fmt.Printf("Creating workflow instance for template: %s\n", uid)
	return nil
}

func main() {
	ctx := context.Background()

	// Create workflow with scheduled execution
	wf := &workflow.Workflow{
		Metadata: workflow.Metadata{
			Name:        "scheduled-workflow",
			DisplayName: "Scheduled Workflow",
			UID:         "scheduled-001",
			TID:         "scheduled-template-001",
		},
		Spec: workflow.Spec{
			Schedule: "*/1 * * * *", // Execute every minute for demo (5 fields format)
		},
	}

	// Create logger and cron
	logger := &SimpleLogger{}
	cronScheduler := cron.New()

	// Create controller and set repository
	controller := workflow.NewWorkflowController(cronScheduler, logger)
	repository := NewSimpleRepository()
	controller.SetRepository(repository)

	// Store the workflow in repository for scheduling
	repository.workflows[wf.TID] = wf

	// Add scheduled workflow to controller
	err := controller.AddScheduledWorkflow(ctx, wf)
	if err != nil {
		fmt.Printf("Failed to add scheduled workflow: %v\n", err)
		return
	}

	fmt.Println("Starting scheduled workflow...")
	fmt.Println("Workflow will execute every minute. Press Ctrl+C to stop.")

	// Start the controller
	controller.Run(ctx)

	// Keep the program running for a while to see scheduled executions
	time.Sleep(5 * time.Minute)

	// Stop the controller
	controller.Stop()
	fmt.Println("Stopped scheduled workflow.")
}
