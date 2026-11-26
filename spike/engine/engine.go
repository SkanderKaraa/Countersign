// Package engine is the boundary between Countersign's layer and a policy
// engine. The layer (store and replica) gathers facts from the permission
// store; the engine decides. Every engine in the spike (the Go reference,
// cedar-go, OPA) implements Engine, and the same tests run against each.
//
// Inputs are plain values, not model types, so an engine sees exactly what
// arrived: an approver that is really an agent reaches the engine as an
// agent, and the engine must refuse it.
package engine

import "time"

// Node is one grant in a chain, with the facts the layer computed from the
// log: its effective scope and end after narrowing, and whether it was
// withdrawn, its task closed, or its agent ended after it was created.
type Node struct {
	ID          string
	Principal   string // "person:<id>" or "agent:<id>"
	Action      string
	Scope       string // canonical path, exact or "<prefix>/**"
	EndsAt      time.Time
	Withdrawn   bool
	WithdrawnBy string
	WithdrawnAt time.Time
	TaskClosed  bool
	Task        string
	AgentEnded  bool
}

// CheckInput asks whether one candidate grant allows a check. Chain starts
// with the candidate and ends with its root grant.
type CheckInput struct {
	Principal string
	Action    string
	Resource  string
	Now       time.Time
	Chain     []Node
	MaxDepth  int // the system maximum delegation depth
}

// ApproveInput asks whether an approval may be accepted.
type ApproveInput struct {
	Approver        string // as received, possibly malformed or an agent
	Requester       string
	Action          string
	Resource        string
	AuthorityScopes []string // scopes of the approver's unexpired approval authority for Action
	ApproverIsAdmin bool
	Class           string
}

// HandDownInput asks whether a parent grant's holder may hand part of it down.
type HandDownInput struct {
	Holder          string
	ParentPrincipal string
	ParentScope     string // effective, after narrowing
	ParentEndsAt    time.Time
	ParentDepth     int
	ParentKind      string // "task" or "service"; a service grant is never handed down
	Class           string // the action class of the child's scope
	ChildScope      string
	ChildEndsAt     time.Time
	ChildDepth      int
	SharingLimit    int // standing sharing limit, if HasSharingLimit
	HasSharingLimit bool
	ChainLen        int // the child grant's position: 1 for a child of a root
	MaxDepth        int
}

// MarkingInput asks whether a marking may be set.
type MarkingInput struct {
	SetBy        string
	SetByIsAdmin bool
	Resource     string
	Class        string
	IsFloor      bool
}

// ServiceGrantInput asks whether a service grant may be proposed.
type ServiceGrantInput struct {
	Creator           string
	CreatorIsAdmin    bool
	CreatorAuthorised bool // an agent's explicit service-grant authorisation
	CreatorHolds      bool // the creator holds this access itself now
	Owner             string
	Resource          string
}

// Result is an engine's answer. Reason is empty when allowed. Policies names
// what decided it, for the explanation (FR-23).
type Result struct {
	Allow    bool
	Reason   string
	Policies []string
}

type Engine interface {
	Name() string
	Check(CheckInput) Result
	MayApprove(ApproveInput) Result
	MayHandDown(HandDownInput) Result
	MaySetMarking(MarkingInput) Result
	MayCreateServiceGrant(ServiceGrantInput) Result
}

// ListInput asks which of many items a principal may act on (FR-22).
// Chains holds every candidate grant's chain, as for a check.
type ListInput struct {
	Principal string
	Action    string
	Now       time.Time
	MaxDepth  int
	Chains    [][]Node
	Items     []string
}

// ListResult says, for each item in order, whether it is allowed, and the
// method used, so a speed difference isn't mistaken for a correctness one.
type ListResult struct {
	Allowed []bool
	Method  string
	Reason  string // set when the whole list is refused
}

// Lister is implemented by an engine with its own way of answering a list
// (OPA: partial evaluation). Other engines are asked one item at a time.
type Lister interface {
	List(ListInput) ListResult
}
