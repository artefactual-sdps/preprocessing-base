package workflow

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/artefactual-sdps/temporal-activities/bagit"
	"go.artefactual.dev/tools/temporal"
	temporalsdk_temporal "go.temporal.io/sdk/temporal"
	temporalsdk_workflow "go.temporal.io/sdk/workflow"
)

type PreprocessingWorkflowParams struct {
	RelativePath string
}

type PreprocessingWorkflowResult struct {
	RelativePath string
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
) (r *PreprocessingWorkflowResult, e error) {
	logger := temporalsdk_workflow.GetLogger(ctx)
	logger.Debug("PreprocessingWorkflow workflow running!", "params", params)

	if params == nil || params.RelativePath == "" {
		e = temporal.NewNonRetryableError(fmt.Errorf("error calling workflow with unexpected inputs"))
		return nil, e
	}

	// Bag the transfer for Enduro processing.
	e = temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		bagit.CreateBagActivityName,
		&bagit.CreateBagActivityParams{
			SourcePath: filepath.Join(w.sharedPath, params.RelativePath),
		},
	).Get(ctx, nil)
	if e != nil {
		return nil, temporal.NewNonRetryableError(fmt.Errorf("create bag: %v", e))
	}

	return &PreprocessingWorkflowResult{RelativePath: params.RelativePath}, e
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
