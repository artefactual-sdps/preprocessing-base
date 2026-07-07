package workflow_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/artefactual-sdps/enduro/pkg/childwf"
	"github.com/artefactual-sdps/temporal-activities/bagcreate"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	temporalsdk_worker "go.temporal.io/sdk/worker"

	"github.com/artefactual-sdps/custom-enduro-workflows/internal/config"
	"github.com/artefactual-sdps/custom-enduro-workflows/internal/workflow"
)

const sharedPath = "/shared/path/"

type PreprocessingTestSuite struct {
	suite.Suite
	temporalsdk_testsuite.WorkflowTestSuite

	env      *temporalsdk_testsuite.TestWorkflowEnvironment
	workflow *workflow.PreprocessingWorkflow
}

func (s *PreprocessingTestSuite) SetupTest(cfg config.Configuration) {
	s.env = s.NewTestWorkflowEnvironment()
	s.env.SetWorkerOptions(temporalsdk_worker.Options{EnableSessionWorker: true})

	// Register activities.
	s.env.RegisterActivityWithOptions(
		bagcreate.New(cfg.Preprocessing.BagCreate).Execute,
		temporalsdk_activity.RegisterOptions{Name: bagcreate.Name},
	)

	cfg.Preprocessing.SharedPath = sharedPath
	s.workflow = workflow.NewPreprocessingWorkflow(cfg.Preprocessing)
}

func (s *PreprocessingTestSuite) AfterTest(suiteName, testName string) {
	s.env.AssertExpectations(s.T())
}

func TestPreprocessingWorkflow(t *testing.T) {
	suite.Run(t, new(PreprocessingTestSuite))
}

func (s *PreprocessingTestSuite) TestSuccess() {
	relPath := "transfer"
	s.SetupTest(config.Configuration{})

	// Mock activities.
	sessionCtx := mock.AnythingOfType("*context.timerCtx")
	s.env.OnActivity(
		bagcreate.Name,
		sessionCtx,
		&bagcreate.Params{SourcePath: filepath.Join(sharedPath, relPath)},
	).Return(
		&bagcreate.Result{BagPath: filepath.Join(sharedPath, relPath)},
		nil,
	)

	s.env.ExecuteWorkflow(
		s.workflow.Execute,
		&childwf.PreprocessingParams{RelativePath: relPath},
	)

	s.True(s.env.IsWorkflowCompleted())

	var result childwf.PreprocessingResult
	err := s.env.GetWorkflowResult(&result)
	s.NoError(err)
	s.Equal(
		&childwf.PreprocessingResult{
			Outcome:      childwf.OutcomeSuccess,
			RelativePath: relPath,
			Tasks: []*childwf.Task{
				{
					Name:        "Bag SIP",
					Message:     "SIP has been bagged",
					Outcome:     childwf.TaskOutcomeSuccess,
					StartedAt:   s.env.Now().UTC(),
					CompletedAt: s.env.Now().UTC(),
				},
			},
		},
		&result,
	)
}

func (s *PreprocessingTestSuite) TestSystemError() {
	relPath := "transfer"
	s.SetupTest(config.Configuration{})

	// Mock activities.
	sessionCtx := mock.AnythingOfType("*context.timerCtx")
	s.env.OnActivity(
		bagcreate.Name,
		sessionCtx,
		&bagcreate.Params{SourcePath: filepath.Join(sharedPath, relPath)},
	).Return(
		nil,
		fmt.Errorf(
			"bagcreate: failed to open %s: permission denied",
			filepath.Join(sharedPath, relPath),
		),
	)

	s.env.ExecuteWorkflow(
		s.workflow.Execute,
		&childwf.PreprocessingParams{RelativePath: relPath},
	)

	s.True(s.env.IsWorkflowCompleted())

	var result childwf.PreprocessingResult
	err := s.env.GetWorkflowResult(&result)
	s.NoError(err)
	s.Equal(
		&childwf.PreprocessingResult{
			Outcome:      childwf.OutcomeSystemError,
			RelativePath: relPath,
			Tasks: []*childwf.Task{
				{
					Name:        "Bag SIP",
					Message:     "System error: bagging has failed",
					Outcome:     childwf.TaskOutcomeSystemFailure,
					StartedAt:   s.env.Now().UTC(),
					CompletedAt: s.env.Now().UTC(),
				},
			},
		},
		&result,
	)
}
