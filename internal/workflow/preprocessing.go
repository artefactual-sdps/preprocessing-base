package workflow

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/artefactual-sdps/temporal-activities/bagcreate"
	"go.artefactual.dev/tools/temporal"
	temporalsdk_temporal "go.temporal.io/sdk/temporal"
	temporalsdk_workflow "go.temporal.io/sdk/workflow"

	"github.com/artefactual-sdps/preprocessing-base/internal/enums"
	"github.com/artefactual-sdps/preprocessing-base/internal/eventlog"
)

type Outcome int

const (
	OutcomeSuccess Outcome = iota
	OutcomeSystemError
	OutcomeContentError
)

type PreprocessingWorkflowParams struct {
	RelativePath string
}

type PreprocessingWorkflowResult struct {
	Outcome           Outcome
	RelativePath      string
	PreservationTasks []*eventlog.Event
}

func (r *PreprocessingWorkflowResult) newEvent(ctx temporalsdk_workflow.Context, name string) *eventlog.Event {
	ev := eventlog.NewEvent(temporalsdk_workflow.Now(ctx), name)
	r.PreservationTasks = append(r.PreservationTasks, ev)

	return ev
}

func (r *PreprocessingWorkflowResult) systemError(
	ctx temporalsdk_workflow.Context,
	err error,
	ev *eventlog.Event,
	msg string,
) *PreprocessingWorkflowResult {
	logger := temporalsdk_workflow.GetLogger(ctx)
	logger.Error("System error", "message", err.Error())

	// Complete last preservation task event.
	ev.Complete(
		temporalsdk_workflow.Now(ctx),
		enums.EventOutcomeSystemFailure,
		"System error: %s",
		msg,
	)
	r.Outcome = OutcomeSystemError

	return r
}

type PreprocessingWorkflow struct {
	sharedPath string
}

func NewPreprocessingWorkflow(sharedPath string) *PreprocessingWorkflow {
	return &PreprocessingWorkflow{
		sharedPath: sharedPath,
	}
}

func (w *PreprocessingWorkflow) Execute(
	ctx temporalsdk_workflow.Context,
	params *PreprocessingWorkflowParams,
) (*PreprocessingWorkflowResult, error) {
	var (
		result PreprocessingWorkflowResult
		e      error
	)

	logger := temporalsdk_workflow.GetLogger(ctx)
	logger.Debug("PreprocessingWorkflow workflow running!", "params", params)

	if params == nil || params.RelativePath == "" {
		e = temporal.NewNonRetryableError(fmt.Errorf("error calling workflow with unexpected inputs"))
		return nil, e
	}
	result.RelativePath = params.RelativePath

	// Bag the SIP for Enduro processing.
	ev := result.newEvent(ctx, "Bag SIP")
	var createBag bagcreate.Result
	e = temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		bagcreate.Name,
		&bagcreate.Params{
			SourcePath: filepath.Join(w.sharedPath, params.RelativePath),
		},
	).Get(ctx, &createBag)
	if e != nil {
		return result.systemError(ctx, e, ev, "bagging has failed"), nil
	}
	ev.Succeed(temporalsdk_workflow.Now(ctx), "SIP has been bagged")

	return &result, e
}

func withLocalActOpts(ctx temporalsdk_workflow.Context) temporalsdk_workflow.Context {
	return temporalsdk_workflow.WithActivityOptions(
		ctx,
		temporalsdk_workflow.ActivityOptions{
			ScheduleToCloseTimeout: 5 * time.Minute,
			RetryPolicy: &temporalsdk_temporal.RetryPolicy{
				MaximumAttempts: 1,
			},
		},
	)
}
