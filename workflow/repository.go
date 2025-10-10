package workflow

import (
	"context"
)

// WorkflowRepository defines the storage abstraction interface for workflow operations
type WorkflowRepository interface {
	// GetWorkflowInstance retrieves workflow by ID
	GetWorkflowInstance(ctx context.Context, uid string) (*Workflow, error)

	// UpdateWorkflowInstance updates an existing workflow
	UpdateWorkflowInstance(ctx context.Context, workflow *Workflow) error

	// ListScheduledWorkflows get running workflows
	ListScheduledWorkflows(ctx context.Context) ([]*Workflow, error)

	// CreateWorkflowInstance create workflow instance
	CreateWorkflowInstance(ctx context.Context, uid string) error
}
