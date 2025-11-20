package workflow

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/robfig/cron/v3"
)

// Mock implementations for testing

// MockExecutor implements NodeExecutor interface
type MockExecutor struct {
	executeFunc func(ctx context.Context, data NodeData) Result
	callCount   int
	mu          sync.Mutex
}

func NewMockExecutor() *MockExecutor {
	return &MockExecutor{
		executeFunc: func(ctx context.Context, data NodeData) Result {
			return Result{Err: nil, Message: "success"}
		},
	}
}

func (m *MockExecutor) ExecuteWorkflowNode(ctx context.Context, data NodeData) Result {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount++
	return m.executeFunc(ctx, data)
}

func (m *MockExecutor) SetExecuteFunc(f func(ctx context.Context, data NodeData) Result) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.executeFunc = f
}

func (m *MockExecutor) GetCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

func (m *MockExecutor) ResetCallCount() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount = 0
}

// MockRepository implements WorkflowRepository interface
type MockRepository struct {
	workflows map[string]*Workflow
	mu        sync.RWMutex
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		workflows: make(map[string]*Workflow),
	}
}

func (r *MockRepository) GetWorkflowInstance(ctx context.Context, uid string) (*Workflow, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	wf, exists := r.workflows[uid]
	if !exists {
		return nil, errors.New("workflow not found")
	}
	return wf, nil
}

func (r *MockRepository) UpdateWorkflowInstance(ctx context.Context, wf *Workflow) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.workflows[wf.UID] = wf
	return nil
}

func (r *MockRepository) CreateWorkflowInstance(ctx context.Context, uid string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.workflows[uid] = &Workflow{Metadata: Metadata{UID: uid}}
	return nil
}

func (r *MockRepository) ListScheduledWorkflows(ctx context.Context) ([]*Workflow, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	workflows := make([]*Workflow, 0, len(r.workflows))
	for _, wf := range r.workflows {
		workflows = append(workflows, wf)
	}
	return workflows, nil
}

// MockLogger implements Logger interface
type MockLogger struct {
	infoLogs  []string
	errorLogs []string
	mu        sync.Mutex
}

func NewMockLogger() *MockLogger {
	return &MockLogger{
		infoLogs:  make([]string, 0),
		errorLogs: make([]string, 0),
	}
}

func (l *MockLogger) Info(msg string, keysAndValues ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.infoLogs = append(l.infoLogs, msg)
}

func (l *MockLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.errorLogs = append(l.errorLogs, msg)
}

// Helper functions for creating test workflows
func createTestWorkflow(name string) *Workflow {
	return &Workflow{
		Metadata: Metadata{
			Name:        name,
			DisplayName: name + " Display",
			UID:         name + "-uid",
		},
		Spec: Spec{
			Schedule: "",
		},
		Status: Status{
			Phase: Pending,
			Nodes: make(Nodes),
		},
	}
}

func createTestController() *WorkflowController {
	logger := NewMockLogger()
	cronScheduler := cron.New()
	return NewWorkflowController(cronScheduler, logger)
}

// Test NewEngine function and options
func TestNewEngine(t *testing.T) {
	ctx := context.Background()
	wf := createTestWorkflow("test-workflow")
	controller := createTestController()
	repository := NewMockRepository()

	t.Run("Basic NewEngine", func(t *testing.T) {
		engine, err := NewEngine(ctx, wf, controller, repository)
		if err != nil {
			t.Fatalf("NewEngine() failed: %v", err)
		}

		if engine == nil {
			t.Fatal("NewEngine() returned nil")
		}

		if engine.wf != wf {
			t.Error("Engine workflow should match input workflow")
		}

		if engine.woc != controller {
			t.Error("Engine controller should match input controller")
		}

		if engine.Repository != repository {
			t.Error("Engine repository should match input repository")
		}

		if engine.executorMap == nil {
			t.Error("Engine executorMap should not be nil")
		}

		if engine.dagExecutor == nil {
			t.Error("Engine dagExecutor should not be nil")
		}

		// Default skipSucceededNodes should be false
		if engine.skipSucceededNodes {
			t.Error("Default skipSucceededNodes should be false")
		}
	})

	t.Run("NewEngine with WithSkipSucceededNodes option", func(t *testing.T) {
		engine, err := NewEngine(ctx, wf, controller, repository, WithSkipSucceededNodes(true))
		if err != nil {
			t.Fatalf("NewEngine() with options failed: %v", err)
		}

		if !engine.skipSucceededNodes {
			t.Error("skipSucceededNodes should be true when WithSkipSucceededNodes(true) is used")
		}
	})

	t.Run("NewEngine with multiple options", func(t *testing.T) {
		engine, err := NewEngine(ctx, wf, controller, repository,
			WithSkipSucceededNodes(true),
			WithSkipSucceededNodes(false), // Should override previous
		)
		if err != nil {
			t.Fatalf("NewEngine() with multiple options failed: %v", err)
		}

		if engine.skipSucceededNodes {
			t.Error("Last option should override previous ones")
		}
	})

	t.Run("NewEngine with nil workflow", func(t *testing.T) {
		_, err := NewEngine(ctx, nil, controller, repository)
		if err == nil {
			t.Error("NewEngine() should fail with nil workflow")
		}
	})

	t.Run("NewEngine with nil controller", func(t *testing.T) {
		_, err := NewEngine(ctx, wf, nil, repository)
		if err == nil {
			t.Error("NewEngine() should fail with nil controller")
		}
	})

	t.Run("NewEngine with nil repository", func(t *testing.T) {
		_, err := NewEngine(ctx, wf, controller, nil)
		if err == nil {
			t.Error("NewEngine() should fail with nil repository")
		}
	})
}

// Test workflow execution
func TestExecuteWorkflow(t *testing.T) {
	ctx := context.Background()
	wf := createTestWorkflow("test-workflow")
	controller := createTestController()
	repository := NewMockRepository()

	t.Run("Execute simple workflow", func(t *testing.T) {
		engine, err := NewEngine(ctx, wf, controller, repository)
		if err != nil {
			t.Fatalf("NewEngine() failed: %v", err)
		}

		// Register executor
		executor := NewMockExecutor()
		nodeType := NodeType("test")
		err = engine.RegisterFunc(nodeType, executor)
		if err != nil {
			t.Fatalf("RegisterFunc() failed: %v", err)
		}

		// Add nodes
		err = engine.AddWorkflowNode("node1", "Node 1", nodeType, NodeData{})
		if err != nil {
			t.Fatalf("AddWorkflowNode() failed: %v", err)
		}

		// Execute workflow
		err = engine.ExecuteWorkflow(ctx)
		if err != nil {
			t.Errorf("ExecuteWorkflow() failed: %v", err)
		}

		// Verify execution
		if executor.GetCallCount() != 1 {
			t.Errorf("Executor should be called once, got %d calls", executor.GetCallCount())
		}

		// Verify workflow status
		if engine.wf.Status.Phase != Succeeded {
			t.Errorf("Workflow phase should be Succeeded, got %s", engine.wf.Status.Phase)
		}

		// Verify node status
		nodeStatus, err := engine.wf.Status.Nodes.Get("node1")
		if err != nil {
			t.Fatalf("Failed to get node status: %v", err)
		}

		if nodeStatus.Phase != NodeSucceeded {
			t.Errorf("Node phase should be Succeeded, got %s", nodeStatus.Phase)
		}
	})

	t.Run("Execute workflow with dependencies", func(t *testing.T) {
		wf2 := createTestWorkflow("test-workflow-2")
		engine, err := NewEngine(ctx, wf2, controller, repository)
		if err != nil {
			t.Fatalf("NewEngine() failed: %v", err)
		}

		// Register executor
		executor := NewMockExecutor()
		nodeType := NodeType("test")
		err = engine.RegisterFunc(nodeType, executor)
		if err != nil {
			t.Fatalf("RegisterFunc() failed: %v", err)
		}

		// Add nodes with dependency
		err = engine.AddWorkflowNode("node1", "Node 1", nodeType, NodeData{})
		if err != nil {
			t.Fatalf("AddWorkflowNode() failed: %v", err)
		}

		err = engine.AddWorkflowNode("node2", "Node 2", nodeType, NodeData{})
		if err != nil {
			t.Fatalf("AddWorkflowNode() failed: %v", err)
		}

		err = engine.AddWorkflowDependency("node1", "node2")
		if err != nil {
			t.Fatalf("AddWorkflowDependency() failed: %v", err)
		}

		// Execute workflow
		err = engine.ExecuteWorkflow(ctx)
		if err != nil {
			t.Errorf("ExecuteWorkflow() failed: %v", err)
		}

		// Verify both nodes executed
		if executor.GetCallCount() != 2 {
			t.Errorf("Executor should be called twice, got %d calls", executor.GetCallCount())
		}

		// Verify workflow status
		if engine.wf.Status.Phase != Succeeded {
			t.Errorf("Workflow phase should be Succeeded, got %s", engine.wf.Status.Phase)
		}
	})

	t.Run("Execute workflow with failing node", func(t *testing.T) {
		wf3 := createTestWorkflow("test-workflow-3")
		engine, err := NewEngine(ctx, wf3, controller, repository)
		if err != nil {
			t.Fatalf("NewEngine() failed: %v", err)
		}

		// Register executor that fails
		executor := NewMockExecutor()
		executor.SetExecuteFunc(func(ctx context.Context, data NodeData) Result {
			return Result{Err: errors.New("execution failed"), Message: "failed"}
		})

		nodeType := NodeType("test")
		err = engine.RegisterFunc(nodeType, executor)
		if err != nil {
			t.Fatalf("RegisterFunc() failed: %v", err)
		}

		// Add node
		err = engine.AddWorkflowNode("node1", "Node 1", nodeType, NodeData{})
		if err != nil {
			t.Fatalf("AddWorkflowNode() failed: %v", err)
		}

		// Execute workflow
		err = engine.ExecuteWorkflow(ctx)
		if err != nil {
			t.Errorf("ExecuteWorkflow() failed: %v", err)
		}

		// Verify workflow status
		if engine.wf.Status.Phase != Failed {
			t.Errorf("Workflow phase should be Failed, got %s", engine.wf.Status.Phase)
		}

		// Verify node status
		nodeStatus, err := engine.wf.Status.Nodes.Get("node1")
		if err != nil {
			t.Fatalf("Failed to get node status: %v", err)
		}

		if nodeStatus.Phase != NodeFailed {
			t.Errorf("Node phase should be Failed, got %s", nodeStatus.Phase)
		}
	})

	t.Run("Execute workflow with non-existent node", func(t *testing.T) {
		wf4 := createTestWorkflow("test-workflow-4")
		engine, err := NewEngine(ctx, wf4, controller, repository)
		if err != nil {
			t.Fatalf("NewEngine() failed: %v", err)
		}

		// Try to execute node that doesn't exist - should handle gracefully
		err = engine.executeNode(ctx, "non-existent-node")
		if err == nil {
			t.Error("executeNode() should fail for non-existent node")
		}
	})

	t.Run("Execute workflow with unregistered executor", func(t *testing.T) {
		wf5 := createTestWorkflow("test-workflow-5")
		engine, err := NewEngine(ctx, wf5, controller, repository)
		if err != nil {
			t.Fatalf("NewEngine() failed: %v", err)
		}

		// Add node without registering executor
		nodeType := NodeType("unregistered")
		err = engine.AddWorkflowNode("node1", "Node 1", nodeType, NodeData{})
		if err != nil {
			t.Fatalf("AddWorkflowNode() failed: %v", err)
		}

		// Execute workflow should handle missing executor gracefully
		err = engine.ExecuteWorkflow(ctx)
		if err != nil {
			t.Errorf("ExecuteWorkflow() failed: %v", err)
		}

		// Verify workflow failed
		if engine.wf.Status.Phase != Failed {
			t.Errorf("Workflow phase should be Failed when executor is missing, got %s", engine.wf.Status.Phase)
		}
	})
}

// Test skip succeeded nodes functionality
func TestSkipSucceededNodes(t *testing.T) {
	ctx := context.Background()
	wf := createTestWorkflow("test-workflow")
	controller := createTestController()
	repository := NewMockRepository()

	t.Run("Skip succeeded nodes enabled", func(t *testing.T) {
		// Create engine with skip option enabled
		engine, err := NewEngine(ctx, wf, controller, repository, WithSkipSucceededNodes(true))
		if err != nil {
			t.Fatalf("NewEngine() failed: %v", err)
		}

		// Register executor
		executor := NewMockExecutor()
		nodeType := NodeType("test")
		err = engine.RegisterFunc(nodeType, executor)
		if err != nil {
			t.Fatalf("RegisterFunc() failed: %v", err)
		}

		// Add node
		err = engine.AddWorkflowNode("node1", "Node 1", nodeType, NodeData{})
		if err != nil {
			t.Fatalf("AddWorkflowNode() failed: %v", err)
		}

		// Manually set node as succeeded (simulating previous execution)
		msg := "Previously completed"
		err = engine.updateNodeStatus("node1", Params{
			Phase:   NodeSucceeded,
			Message: &msg,
		})
		if err != nil {
			t.Fatalf("updateNodeStatus() failed: %v", err)
		}

		executor.ResetCallCount()

		// Execute workflow
		err = engine.ExecuteWorkflow(ctx)
		if err != nil {
			t.Errorf("ExecuteWorkflow() failed: %v", err)
		}

		// Verify executor was not called (node was skipped)
		if executor.GetCallCount() != 0 {
			t.Errorf("Executor should not be called for succeeded node, got %d calls", executor.GetCallCount())
		}
	})

	t.Run("Skip succeeded nodes disabled", func(t *testing.T) {
		wf2 := createTestWorkflow("test-workflow-2")
		// Create engine with skip option disabled
		engine, err := NewEngine(ctx, wf2, controller, repository, WithSkipSucceededNodes(false))
		if err != nil {
			t.Fatalf("NewEngine() failed: %v", err)
		}

		// Register executor
		executor := NewMockExecutor()
		nodeType := NodeType("test")
		err = engine.RegisterFunc(nodeType, executor)
		if err != nil {
			t.Fatalf("RegisterFunc() failed: %v", err)
		}

		// Add node
		err = engine.AddWorkflowNode("node1", "Node 1", nodeType, NodeData{})
		if err != nil {
			t.Fatalf("AddWorkflowNode() failed: %v", err)
		}

		// Manually set node as succeeded (simulating previous execution)
		msg := "Previously completed"
		err = engine.updateNodeStatus("node1", Params{
			Phase:   NodeSucceeded,
			Message: &msg,
		})
		if err != nil {
			t.Fatalf("updateNodeStatus() failed: %v", err)
		}

		executor.ResetCallCount()

		// Execute workflow
		err = engine.ExecuteWorkflow(ctx)
		if err != nil {
			t.Errorf("ExecuteWorkflow() failed: %v", err)
		}

		// Verify executor was called (node was re-executed)
		if executor.GetCallCount() != 1 {
			t.Errorf("Executor should be called once for succeeded node when skip is disabled, got %d calls", executor.GetCallCount())
		}
	})
}

// Test Nodes.Get method edge cases
func TestNodesGet(t *testing.T) {
	t.Run("Get from nil Nodes", func(t *testing.T) {
		var nodes Nodes // nil map
		_, err := nodes.Get("test")
		if err == nil {
			t.Error("Get() should fail for nil Nodes map")
		}
	})

	t.Run("Get non-existent key", func(t *testing.T) {
		nodes := make(Nodes)
		_, err := nodes.Get("non-existent")
		if err == nil {
			t.Error("Get() should fail for non-existent key")
		}
	})

	t.Run("Get existing key", func(t *testing.T) {
		nodes := make(Nodes)
		expected := NodeStatus{
			ID:    "test",
			Name:  "Test Node",
			Phase: NodePending,
		}
		nodes.Set("test", expected)

		result, err := nodes.Get("test")
		if err != nil {
			t.Fatalf("Get() failed: %v", err)
		}

		if result.ID != expected.ID {
			t.Errorf("Expected ID %s, got %s", expected.ID, result.ID)
		}
	})
}

// Test engine initialization edge cases
func TestEngineInitialization(t *testing.T) {
	ctx := context.Background()
	controller := createTestController()
	repository := NewMockRepository()

	t.Run("Engine with nil Nodes map", func(t *testing.T) {
		wf := createTestWorkflow("test-workflow")
		wf.Status.Nodes = nil // Set to nil

		engine, err := NewEngine(ctx, wf, controller, repository)
		if err != nil {
			t.Fatalf("NewEngine() failed: %v", err)
		}

		// Should initialize Nodes map
		if engine.wf.Status.Nodes == nil {
			t.Error("Engine should initialize nil Nodes map")
		}
	})

	t.Run("Engine executor map initialization", func(t *testing.T) {
		wf := createTestWorkflow("test-workflow")
		engine, err := NewEngine(ctx, wf, controller, repository)
		if err != nil {
			t.Fatalf("NewEngine() failed: %v", err)
		}

		// executorMap should be initialized
		if engine.executorMap == nil {
			t.Error("Engine executorMap should not be nil")
		}

		// Should be able to register executors
		executor := NewMockExecutor()
		err = engine.RegisterFunc("test", executor)
		if err != nil {
			t.Errorf("RegisterFunc() failed: %v", err)
		}
	})
}
