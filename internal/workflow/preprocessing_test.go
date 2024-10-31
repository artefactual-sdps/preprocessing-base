package workflow_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/artefactual-sdps/temporal-activities/bagcreate"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	temporalsdk_worker "go.temporal.io/sdk/worker"

	"github.com/artefactual-sdps/preprocessing-base/internal/config"
	"github.com/artefactual-sdps/preprocessing-base/internal/enums"
	"github.com/artefactual-sdps/preprocessing-base/internal/eventlog"
	"github.com/artefactual-sdps/preprocessing-base/internal/workflow"
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
		bagcreate.New(cfg.Bagit).Execute,
		temporalsdk_activity.RegisterOptions{Name: bagcreate.Name},
	)

	s.workflow = workflow.NewPreprocessingWorkflow(sharedPath)
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
		&workflow.PreprocessingWorkflowParams{RelativePath: relPath},
	)

	s.True(s.env.IsWorkflowCompleted())

	var result workflow.PreprocessingWorkflowResult
	err := s.env.GetWorkflowResult(&result)
	s.NoError(err)
	s.Equal(
		&workflow.PreprocessingWorkflowResult{
			Outcome:      workflow.OutcomeSuccess,
			RelativePath: relPath,
			PreservationTasks: []*eventlog.Event{
				{
					Name:        "Bag SIP",
					Message:     "SIP has been bagged",
					Outcome:     enums.EventOutcomeSuccess,
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
		&workflow.PreprocessingWorkflowParams{RelativePath: relPath},
	)

	s.True(s.env.IsWorkflowCompleted())

	var result workflow.PreprocessingWorkflowResult
	err := s.env.GetWorkflowResult(&result)
	s.NoError(err)
	s.Equal(
		&workflow.PreprocessingWorkflowResult{
			Outcome:      workflow.OutcomeSystemError,
			RelativePath: relPath,
			PreservationTasks: []*eventlog.Event{
				{
					Name:        "Bag SIP",
					Message:     "System error: bagging has failed",
					Outcome:     enums.EventOutcomeSystemFailure,
					StartedAt:   s.env.Now().UTC(),
					CompletedAt: s.env.Now().UTC(),
				},
			},
		},
		&result,
	)
}
