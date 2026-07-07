package workflow

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/artefactual-sdps/enduro/pkg/childwf"
	"github.com/artefactual-sdps/temporal-activities/bagcreate"
	"go.artefactual.dev/tools/temporal"
	temporalsdk_temporal "go.temporal.io/sdk/temporal"
	temporalsdk_workflow "go.temporal.io/sdk/workflow"

	"github.com/artefactual-sdps/custom-enduro-workflows/internal/config"
)

type PreprocessingWorkflow struct {
	cfg config.PreprocessingConfig
}

func NewPreprocessingWorkflow(cfg config.PreprocessingConfig) *PreprocessingWorkflow {
	return &PreprocessingWorkflow{cfg: cfg}
}

func (w *PreprocessingWorkflow) Execute(
	ctx temporalsdk_workflow.Context,
	params *childwf.PreprocessingParams,
) (*childwf.PreprocessingResult, error) {
	result := &childwf.PreprocessingResult{}
	logger := temporalsdk_workflow.GetLogger(ctx)
	logger.Debug("PreprocessingWorkflow workflow running!", "params", params)

	if params == nil || params.RelativePath == "" {
		return nil, temporal.NewNonRetryableError(fmt.Errorf("error calling workflow with unexpected inputs"))
	}
	result.RelativePath = params.RelativePath

	// Bag the SIP for Enduro processing.
	task := result.NewTask(temporalsdk_workflow.Now(ctx), "Bag SIP")
	var createBag bagcreate.Result
	err := temporalsdk_workflow.ExecuteActivity(
		withFilesystemActivityOpts(ctx),
		bagcreate.Name,
		&bagcreate.Params{
			SourcePath: filepath.Join(w.cfg.SharedPath, params.RelativePath),
		},
	).Get(ctx, &createBag)
	if err != nil {
		logger.Error("System error", "message", err.Error())
		result.SystemError(temporalsdk_workflow.Now(ctx), task, "bagging has failed")
		return result, nil
	}
	task.Succeed(temporalsdk_workflow.Now(ctx), "SIP has been bagged")

	return result, nil
}

func withFilesystemActivityOpts(ctx temporalsdk_workflow.Context) temporalsdk_workflow.Context {
	return temporalsdk_workflow.WithActivityOptions(ctx, temporalsdk_workflow.ActivityOptions{
		StartToCloseTimeout: time.Hour * 2,
		RetryPolicy: &temporalsdk_temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
	})
}
