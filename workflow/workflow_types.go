package workflow

import (
	"encoding/json"
	"fmt"
	"hash/fnv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

// Workflow is the definition of a workflow resource
type Workflow struct {
	metav1.TypeMeta `json:",inline"`
	Metadata        `json:"metadata"`
	Spec            Spec   `json:"spec"`
	Status          Status `json:"status,omitempty"`
}

type Metadata struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`

	// TID workflow template id.
	TID string `json:"tid"`
	// UID workflow id.
	UID string `json:"uid"`
}

// NodeID creates a deterministic node ID based on a node name
func (wf *Workflow) NodeID(name string) string {
	if wf.Name == name {
		return wf.Name
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(name))
	return fmt.Sprintf("%s-%v", wf.Name, h.Sum32())
}

func (wf *Workflow) GetNodeByName(name string) (*NodeStatus, error) {
	nodeID := wf.NodeID(name)
	return wf.Status.Nodes.Get(nodeID)
}

type Spec struct {
	// Suspend will suspend the workflow and prevent execution of any future steps in the workflow
	Suspend *bool `json:"suspend,omitempty"`

	// Schedule is a schedule to run the Workflow in Cron format.
	Schedule string `json:"schedule,omitempty"`
}

type Status struct {
	// Phase a simple, high-level summary of where the workflow is in its lifecycle.
	// Will be "" (Unknown), "Pending", or "Running" before the workflow is completed, and "Succeeded",
	// "Failed" or "Error" once the workflow has completed.
	Phase Phase `json:"phase,omitempty"`

	// Nodes is a mapping between a node ID and the node's status.
	Nodes Nodes `json:"nodes,omitempty"`

	// A human readable message indicating details about why the workflow is in this condition.
	Message string `json:"message,omitempty"`

	// LastScheduledTime is the last time the workflow was scheduled (for cron workflows)
	LastScheduledTime *metav1.Time `json:"lastScheduledTime,omitempty"`

	// NextScheduledTime is the next time the workflow will be scheduled (for cron workflows)
	NextScheduledTime *metav1.Time `json:"nextScheduledTime,omitempty"`

	// ScheduleActive indicates if the cron schedule is currently active
	ScheduleActive bool `json:"scheduleActive,omitempty"`
}

type Nodes map[string]NodeStatus

// Set the status of a node by key
func (n Nodes) Set(key string, status NodeStatus) {
	if status.Name == "" {
		klog.V(3).Info("Name was not set for key ", key)
	}
	if status.ID == "" {
		klog.V(3).Info("ID was not set for key ", key)
	}
	_, ok := n[key]
	if ok {
		klog.V(3).Info("Changing NodeStatus for ", key, " to ", status)
	}
	n[key] = status
}

// Get a NodeStatus from the hashmap of Nodes.
// Return a nil along with an error if non existent.
func (n Nodes) Get(key string) (*NodeStatus, error) {
	val, ok := n[key]
	if !ok {
		return nil, fmt.Errorf("key was not found for %s", key)
	}
	return &val, nil
}

// GetName the name of a node by key
func (n Nodes) GetName(key string) (string, error) {
	val, ok := n[key]
	if !ok {
		return "", nil
	}
	return val.Name, nil
}

// NodeStatus contains status information about an individual node in the workflow
type NodeStatus struct {
	// ID is a unique identifier of a node within the workflow
	// It is implemented as a hash of the node name, which makes the ID deterministic
	ID string `json:"id"`

	// Name is unique name in the node tree used to generate the node ID
	Name string `json:"name"`

	// DisplayName is a human readable representation of the node. Unique within a template boundary
	DisplayName string `json:"displayName,omitempty"`

	// Type indicates type of node
	Type NodeType `json:"type"`

	// Phase a simple, high-level summary of where the node is in its lifecycle.
	// Can be used as a state machine.
	// Will be one of these values "Pending", "Running" before the node is completed, or "Succeeded",
	// "Skipped", "Failed", "Error", or "Omitted" as a final state.
	Phase NodePhase `json:"phase,omitempty"`

	// Children is a list of child node IDs
	Children []string `json:"children,omitempty"`

	// Data is the data the node has
	Data NodeData `json:"data,omitempty"`

	// Time at which this node started
	StartedAt metav1.Time `json:"startedAt,omitempty" protobuf:"bytes,10,opt,name=startedAt"`

	// Time at which this node completed
	FinishedAt metav1.Time `json:"finishedAt,omitempty" protobuf:"bytes,11,opt,name=finishedAt"`

	// A human readable message indicating details about why the node is in this condition.
	Message string `json:"message,omitempty"`
}

type NodeData map[string]interface{}

// IsScheduled checks if the workflow has a schedule configured
func (wf *Workflow) IsScheduled() bool {
	return wf.Spec.Schedule != ""
}

// Suspend checks if the workflow is suspended
func (wf *Workflow) Suspend() bool {
	return wf.Spec.Suspend != nil && *wf.Spec.Suspend
}

// IsRetryable checks if the workflow can be retried based on its current state
func (wf *Workflow) IsRetryable() bool {
	// Can retry if workflow is in failed state or has failed/pending nodes
	if wf.Status.Phase == Failed || wf.Status.Phase == Error {
		return true
	}

	// Check if there are any failed or pending nodes
	for _, node := range wf.Status.Nodes {
		if node.Phase == NodeFailed || node.Phase == NodeError || node.Phase == NodePending {
			return true
		}
	}

	return false
}

// Template is a reusable and composable unit of execution in a workflow
type Template struct {
	// Name is the name of the template
	Name string `json:"name,omitempty"`
}

func (wf *Workflow) ToJson() (string, error) {
	b, err := json.Marshal(wf)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (wf *Workflow) SetUID(uid string) {
	wf.UID = uid
}
