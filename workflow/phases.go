package workflow

// Phase the workflow's phase
type Phase string

const (
	Unknown   Phase = ""
	Pending   Phase = "Pending" // pending some set-up - rarely used
	Running   Phase = "Running" // any node has started
	Succeeded Phase = "Succeeded"
	Failed    Phase = "Failed" // it maybe that the workflow was terminated
	Error     Phase = "Error"
)

func (p Phase) Finished() bool {
	return p != Pending && p != Running
}

func RunningPhases() []Phase {
	return []Phase{
		Pending,
		Running,
	}
}

// NodePhase is a label for the condition of a node at the current time.
type NodePhase string

// Workflow and node statuses
const (
	// NodePending is waiting to run
	NodePending NodePhase = "Pending"
	// NodeRunning is running
	NodeRunning NodePhase = "Running"
	// NodeSucceeded finished with no errors
	NodeSucceeded NodePhase = "Succeeded"
	// NodeSkipped was skipped
	NodeSkipped NodePhase = "Skipped"
	// NodeFailed or child of node exited with non-0 code
	NodeFailed NodePhase = "Failed"
	// NodeError had an error other than a non 0 exit code
	NodeError NodePhase = "Error"
	// NodeOmitted was omitted because its `depends` condition was not met (only relevant in DAGs)
	NodeOmitted NodePhase = "Omitted"
)

func (n NodePhase) Completed() bool {
	return n == NodeSucceeded || n == NodeSkipped
}

// NodeType is the type of a node
type NodeType string

// Node types

const (
	NodeTypeStart   NodeType = "start"
	NodeTypeEnd     NodeType = "end"
	NodeTypeDeploy  NodeType = "deploy"
	NodeTypeApiCall NodeType = "api-call"
	NodeTypeBuild   NodeType = "build"
)
