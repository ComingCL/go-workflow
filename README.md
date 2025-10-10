# Go-Workflow

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

A lightweight, high-performance workflow engine designed for seamless integration with business code. Features storage-separated architecture, DAG (Directed Acyclic Graph) workflow orchestration, and flexible node execution with dependency management.

## 🚀 Features

- **Lightweight & Fast**: Minimal dependencies, optimized for performance
- **DAG Workflows**: Support for complex directed acyclic graph orchestration
- **Storage Agnostic**: Pluggable storage backends with clean interface separation
- **Scheduled Execution**: Built-in Cron scheduler for automated workflows
- **Type Safe**: Strong typing with comprehensive error handling
- **Extensible**: Plugin-based node executors for custom business logic
- **Concurrent**: Intelligent parallel execution with dependency resolution
- **Production Ready**: Comprehensive logging and monitoring capabilities

## 📦 Installation

```bash
go get github.com/ComingCL/go-workflow
```

### Core Components

- **WorkflowEngine**: Executes workflows with DAG parsing and node orchestration
- **WorkflowDAG**: Manages directed acyclic graphs with topological sorting
- **WorkflowController**: Handles workflow lifecycle and scheduling
- **CronScheduler**: Provides cron-based automated execution
- **WorkflowRepository**: Abstraction layer for data persistence

## 🚀 Quick Start

### Basic Workflow

```go
import "github.com/ComingCL/go-workflow/workflow"

// Create workflow
wf := &workflow.Workflow{
    Metadata: workflow.Metadata{
        Name: "my-workflow",
        UID:  "wf-001",
    },
}

// Initialize engine
engine, err := workflow.NewEngine(ctx, wf, controller, repository)
if err != nil {
    log.Fatal(err)
}

// Add nodes with dependencies
engine.AddWorkflowNode("build", "Build", NodeTypeBuild, buildData)
engine.AddWorkflowNode("test", "Test", NodeTypeTest, testData)
engine.AddWorkflowNode("deploy", "Deploy", NodeTypeDeploy, deployData)

// Define execution order
engine.AddWorkflowDependency("build", "test")
engine.AddWorkflowDependency("test", "deploy")

// Execute
err = engine.ExecuteWorkflow(ctx)
```

### Scheduled Workflow

```go
// Create scheduled workflow
wf := &workflow.Workflow{
    Spec: workflow.Spec{
        Schedule: "0 2 * * *", // Daily at 2 AM
    },
}

// Add to scheduler
controller.AddScheduledWorkflow(ctx, wf)
controller.Run(ctx)
```

## 📖 Examples

Comprehensive examples are available in the [`examples/`](./examples/) directory:

- **[Basic Usage](./examples/basic/)** - Complete workflow setup and execution
- **[Scheduled Workflows](./examples/scheduled/)** - Cron-based automation
- **[DAG Management](./examples/dag/)** - Advanced graph operations

Each example includes full source code and can be run independently:

```bash
cd examples/basic && go run main.go
cd examples/scheduled && go run main.go
cd examples/dag && go run main.go
```

## 🔧 Configuration

### Custom Node Executor

```go
type CustomExecutor struct{}

func (e *CustomExecutor) ExecuteWorkflowNode(ctx context.Context, data workflow.NodeData) workflow.Result {
    // Your business logic here
    return workflow.Result{
        Err:     nil,
        Message: "Success",
    }
}

// Register executor
engine.RegisterFunc(NodeTypeBuild, &CustomExecutor{})
```

### Storage Repository

```go
type MyRepository struct{}

func (r *MyRepository) GetWorkflowInstance(ctx context.Context, uid string) (*workflow.Workflow, error) {
    // Implement your storage logic
}

func (r *MyRepository) UpdateWorkflowInstance(ctx context.Context, wf *workflow.Workflow) error {
    // Implement your storage logic
}

// Set repository
controller.SetRepository(&MyRepository{})
```

## 📊 Node States

### Node States
- `NodePending` - Awaiting execution
- `NodeRunning` - Currently executing
- `NodeSucceeded` - Completed successfully
- `NodeFailed` - Execution failed
- `NodeSkipped` - Conditionally skipped

## 🧪 Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./workflow -v
```

## 🔄 Integration Patterns

### Embedded Integration
Perfect for monolithic applications where workflows are tightly coupled with business logic.

### Microservice Integration
Ideal for distributed systems with workflow orchestration as a separate service.

### Event-Driven Integration
Suitable for reactive systems with message queue triggers.

## 📋 Dependencies

- **Go 1.21+** - Core runtime requirement
- **github.com/ComingCL/go-stl** - Graph data structures
- **github.com/robfig/cron/v3** - Cron scheduling
- **k8s.io/apimachinery** - Metadata handling
- **k8s.io/klog/v2** - Structured logging

## 🛣️ Roadmap

- [ ] Web-based workflow designer
- [ ] REST API endpoints
- [ ] Prometheus metrics integration
- [ ] Distributed execution support
- [ ] Conditional workflow branches
- [ ] Workflow templates and inheritance

## 🤝 Contributing

Contributions are welcome! Please follow these guidelines:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Guidelines

- Follow Go coding standards and best practices
- Add comprehensive tests for new features
- Update documentation for API changes
- Ensure all tests pass before submitting

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [robfig/cron](https://github.com/robfig/cron) - Reliable cron scheduling
- [Kubernetes](https://kubernetes.io/) - API design patterns and best practices
- [Go STL](https://github.com/ComingCL/go-stl) - Efficient graph algorithms

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/ComingCL/go-workflow/issues)
- **Discussions**: [GitHub Discussions](https://github.com/ComingCL/go-workflow/discussions)
- **Documentation**: [Project Wiki](https://github.com/ComingCL/go-workflow/wiki)

---

**Go-Workflow** - Simplifying workflow orchestration for Go applications 🚀