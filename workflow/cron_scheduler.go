package workflow

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CronScheduler manages cron-based workflow scheduling
type CronScheduler struct {
	mu       sync.RWMutex
	cron     *cron.Cron
	entryIDs map[string]cron.EntryID // workflow template id -> cron entry ID

	// Workflow controller for executing workflows
	wfController *WorkflowController

	// Active workflows tracking (for concurrency control)
	activeWorkflows map[string]*Workflow // workflow name -> running workflow

	// Logger for scheduling events
	logger Logger
}

// NewCronScheduler creates a new CronScheduler
func NewCronScheduler(wfController *WorkflowController, logger Logger) *CronScheduler {
	return &CronScheduler{
		cron:            wfController.cron,
		entryIDs:        make(map[string]cron.EntryID),
		wfController:    wfController,
		activeWorkflows: make(map[string]*Workflow),
		logger:          logger,
	}
}

// Start starts the cron scheduler
func (cs *CronScheduler) Start(ctx context.Context) error {
	cs.logger.Info("Starting cron scheduler for workflows")

	// Load existing scheduled workflows
	if err := cs.loadScheduledWorkflows(ctx); err != nil {
		return fmt.Errorf("failed to load scheduled workflows: %v", err)
	}

	cs.cron.Start()
	return nil
}

// Stop stops the cron scheduler
func (cs *CronScheduler) Stop() {
	cs.logger.Info("Stopping cron scheduler for workflows")
	cs.cron.Stop()
}

// AddWorkflow adds or updates a workflow schedule
func (cs *CronScheduler) AddWorkflow(ctx context.Context, wf *Workflow) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Remove existing schedule if any
	if entryID, exists := cs.entryIDs[wf.TID]; exists {
		cs.cron.Remove(entryID)
		delete(cs.entryIDs, wf.TID)
	}

	// Create cron job
	job := &cronWorkflowJob{
		ctx:         ctx,
		scheduler:   cs,
		workflowUID: wf.TID,
	}

	entryID, err := cs.cron.AddJob(wf.Spec.Schedule, job)
	if err != nil {
		return fmt.Errorf("failed to add cron job for workflow %s: %v", wf.TID, err)
	}

	cs.entryIDs[wf.TID] = entryID

	return nil
}

func (cs *CronScheduler) CronWorkflowExist(sid string) bool {
	_, found := cs.entryIDs[sid]
	return found
}

// RemoveWorkflow removes a workflow from scheduling
func (cs *CronScheduler) RemoveWorkflow(workflowUID string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if entryID, exists := cs.entryIDs[workflowUID]; exists {
		cs.cron.Remove(entryID)
		delete(cs.entryIDs, workflowUID)
		cs.logger.Info("Removed cron schedule for workflow")
	}

	// Remove from active workflows if present
	delete(cs.activeWorkflows, workflowUID)
}

// loadScheduledWorkflows loads all scheduled workflows from repository
func (cs *CronScheduler) loadScheduledWorkflows(ctx context.Context) error {
	workflows, err := cs.wfController.repository.ListScheduledWorkflows(ctx)
	if err != nil {
		return err
	}

	for i := range workflows {
		if workflows[i].Suspend() {
			continue
		}
		if err = cs.AddWorkflow(ctx, workflows[i]); err != nil {
			cs.logger.Error(err, "Failed to add scheduled workflow")
		}
	}

	cs.logger.Info("Loaded scheduled workflows")
	return nil
}

// getNextScheduleTime gets the next scheduled time for a cron entry
func (cs *CronScheduler) getNextScheduleTime(entryID cron.EntryID) *metav1.Time {
	entry := cs.cron.Entry(entryID)
	if entry.Next.IsZero() {
		return nil
	}
	return &metav1.Time{Time: entry.Next}
}

// cronWorkflowJob implements the cron.Job interface for workflow execution
type cronWorkflowJob struct {
	ctx context.Context

	scheduler *CronScheduler
	// workflowUID workflow template id
	workflowUID string
}

// Run executes the scheduled workflow
func (job *cronWorkflowJob) Run() {
	ctx := job.ctx
	scheduledTime := time.Now()

	// Get the original workflow
	originalWf, err := job.scheduler.wfController.repository.GetWorkflowInstance(ctx, job.workflowUID)
	if err != nil {
		return
	}

	// Create workflow instance
	if err = job.scheduler.wfController.repository.CreateWorkflowInstance(ctx, job.workflowUID); err != nil {
		return
	}

	// Update last scheduled time
	originalWf.Status.LastScheduledTime = &metav1.Time{Time: scheduledTime}
	if err = job.scheduler.wfController.repository.UpdateWorkflowInstance(ctx, originalWf); err != nil {
		return
	}

	// Execute the workflow instance
	job.scheduler.wfController.AddWorkflow(ctx, &WorkflowEngine{
		ctx: ctx,
		wf:  originalWf,
		woc: job.scheduler.wfController,
	})
}
