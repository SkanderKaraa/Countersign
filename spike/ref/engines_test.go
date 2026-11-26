package ref

import (
	"os"
	"testing"

	"countersign/spike/cedarengine"
	"countersign/spike/engine"
	"countersign/spike/goref"
	"countersign/spike/opaengine"
)

// testEngine picks the engine under test from COUNTERSIGN_ENGINE, so the
// same tests run against every engine:
//
//	go test ./...                           # the Go reference
//	COUNTERSIGN_ENGINE=cedar go test ./...  # cedar-go
func testEngine(t *testing.T) engine.Engine {
	t.Helper()
	switch name := os.Getenv("COUNTERSIGN_ENGINE"); name {
	case "", "goref":
		return goref.New()
	case "cedar":
		e, err := cedarengine.New(DefaultLimits().SystemMaxDepth)
		if err != nil {
			t.Fatal(err)
		}
		return e
	case "opa":
		e, err := opaengine.New()
		if err != nil {
			t.Fatal(err)
		}
		return e
	case "denyall":
		return denyAll{}
	case "allowall":
		return allowAll{}
	default:
		t.Fatalf("unknown engine %q", name)
		return nil
	}
}

// denyAll refuses everything. It exists only to show that the permit-side
// contract tests fail on an engine that never allows anything.
type denyAll struct{}

func (denyAll) Name() string                          { return "denyall" }
func (denyAll) Check(engine.CheckInput) engine.Result { return engine.Result{Reason: "deny all"} }
func (denyAll) MayApprove(engine.ApproveInput) engine.Result {
	return engine.Result{Reason: "deny all"}
}
func (denyAll) MayHandDown(engine.HandDownInput) engine.Result {
	return engine.Result{Reason: "deny all"}
}
func (denyAll) MaySetMarking(engine.MarkingInput) engine.Result {
	return engine.Result{Reason: "deny all"}
}
func (denyAll) MayCreateServiceGrant(engine.ServiceGrantInput) engine.Result {
	return engine.Result{Reason: "deny all"}
}

// allowAll allows everything. It exists only to show that the refusal-side
// contract tests fail on an engine that never refuses.
type allowAll struct{}

func allowed() engine.Result { return engine.Result{Allow: true} }

func (allowAll) Name() string                                                 { return "allowall" }
func (allowAll) Check(engine.CheckInput) engine.Result                        { return allowed() }
func (allowAll) MayApprove(engine.ApproveInput) engine.Result                 { return allowed() }
func (allowAll) MayHandDown(engine.HandDownInput) engine.Result               { return allowed() }
func (allowAll) MaySetMarking(engine.MarkingInput) engine.Result              { return allowed() }
func (allowAll) MayCreateServiceGrant(engine.ServiceGrantInput) engine.Result { return allowed() }
