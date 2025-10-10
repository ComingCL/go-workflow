package workflow

import (
	"context"
	"fmt"
	"time"

	"github.com/gogf/gf/util/gconv"
	"github.com/robfig/cron/v3"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"
)

const wfWorkers = 3

type WorkflowController struct {
	cron    *cron.Cron
	wfQueue workqueue.RateLimitingInterface

	// repository dependencies
	repository WorkflowRepository

	// Cron scheduler for scheduled workflows
	cronScheduler *CronScheduler

	// Logger instance
	logger Logger
}

func NewWorkflowController(c *cron.Cron, logger Logger) *WorkflowController {
	w := &WorkflowController{
		cron:    c,
		wfQueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "workflow_queue"),
		logger:  logger,
	}
	w.cronScheduler = NewCronScheduler(w, logger)

	return w
}

func (w *WorkflowController) SetRepository(repository WorkflowRepository) {
	w.repository = repository
}

func (w *WorkflowController) Run(ctx context.Context) {
	// Start cron scheduler if repositories are set
	if w.cronScheduler != nil {
		if err := w.cronScheduler.Start(ctx); err != nil {
			w.logger.Error(err, "Failed to start cron scheduler")
		}
	}

	for i := 0; i < wfWorkers; i++ {
		go wait.Until(w.runWorker, time.Second, ctx.Done())
	}
}

// Stop stops the workflow controller and cron scheduler
func (w *WorkflowController) Stop() {
	if w.cronScheduler != nil {
		w.cronScheduler.Stop()
	}
}

func (w *WorkflowController) AddWorkflow(ctx context.Context, engine *WorkflowEngine) {
	w.wfQueue.Add(engine)
}

func (w *WorkflowController) runWorker() {
	for w.processNextItem(context.TODO()) {
	}
}

func (w *WorkflowController) processNextItem(ctx context.Context) bool {
	wfEngineObj, quit := w.wfQueue.Get()
	if quit {
		return false
	}
	defer w.wfQueue.Done(wfEngineObj)
	wfEngine, ok := wfEngineObj.(*WorkflowEngine)
	if !ok {
		w.logger.Info("skip processing workflow event: " + gconv.String(wfEngineObj))
		return true
	}
	wfEngine.operator(ctx)
	return true
}

func (w *WorkflowController) CronWorkflowExist(sid string) bool {
	return w.cronScheduler.CronWorkflowExist(sid)
}

// AddScheduledWorkflow adds a workflow to the cron scheduler
func (w *WorkflowController) AddScheduledWorkflow(ctx context.Context, wf *Workflow) error {
	if w.cronScheduler == nil {
		return fmt.Errorf("cron scheduler not initialized")
	}
	if wf.Suspend() {
		return nil
	}
	return w.cronScheduler.AddWorkflow(ctx, wf)
}

// RemoveScheduledWorkflow removes a workflow from the cron scheduler
func (w *WorkflowController) RemoveScheduledWorkflow(workflowUid string) {
	if w.cronScheduler != nil {
		w.cronScheduler.RemoveWorkflow(workflowUid)
	}
}
