# custom-enduro-workflows

This repository is a template for Enduro child workflow projects written in
Go. It contains a small preprocessing workflow, a Temporal worker, tests,
configuration, a container image, and resources for the Enduro development
cluster.

For Enduro's child workflow extension contracts, result handling, tasks,
metadata, and failure behavior, see the [Enduro custom child workflow guide].
Use this README to adapt, configure, test, and run the template.

* [Template behavior](#template-behavior)
* [Create a project](#create-a-project)
* [Repository structure](#repository-structure)
* [Register workflows and activities](#register-workflows-and-activities)
* [Configuration and logging](#configuration-and-logging)
* [Develop with Enduro](#develop-with-enduro)
* [Reusable activities](#reusable-activities)
* [Related projects](#related-projects)

## Template behavior

The example workflow accepts `childwf.PreprocessingParams`, resolves the SIP
path below the worker's shared storage root, and schedules the reusable
`bagcreate` activity. It records a task in `childwf.PreprocessingResult` and
returns the relative path of the modified SIP to Enduro. Modify this example
workflow as necessary to implement your own policy and processing requirements.

The local Kubernetes resources provide `preprocessing-pvc` as shared storage
for development on a single node. Enduro and the custom worker mount the same
claim so both can access the SIP. Production deployments must provide shared
storage that is suitable for their infrastructure.

Projects that implement only poststorage or postbatch child workflows can
remove the shared volume as they do not modify the SIP structure or contents.

## Create a project

Use the GitHub **Use this template** action to create a repository. When
adapting the template, update:

* the module path in `go.mod` and all Go import paths;
* the `custom-enduro-workflows` and `custom-enduro-worker` names;
* the `CUSTOM_ENDURO_WORKER` environment variable prefix;
* the binary, image, Kubernetes resource, Secret, and configuration names;
* the module path used by the version build scripts;
* the Tilt resource name and `Custom` label;
* the configuration structs and registered workflows and activities;
* the example workflow, tests, and Kubernetes resources; and
* this README, image build, and deployment instructions.

## Repository structure

The main template paths are:

| Path | Purpose |
| --- | --- |
| `cmd/worker/` | Worker command and Temporal registration. |
| `internal/config/` | Configuration loading and validation. |
| `internal/workflows/` | Workflow implementation and Temporal tests. |
| `hack/kube/` | Kubernetes Kustomize resource definitions for workers, storage volumes, and application configuration. |
| `hack/build_dist.sh` | Shell script to build the worker binary with application version metadata. |
| `hack/build_docker.sh` | Shell script to build the Docker worker image with versioned worker binary. |
| `Dockerfile` | Docker image build plan. |
| `Makefile` | Development helpers for building tools, and formatting, linting, and testing the code. |
| `Tiltfile` | Configuration file for the Tilt development environment. |

## Register workflows and activities

[`cmd/worker/workercmd/cmd.go`] creates the Temporal client and worker. It
registers the preprocessing implementation under the configured Workflow Type
name and registers `bagcreate` on the same worker:

```go
w.RegisterWorkflowWithOptions(
    workflows.NewPreprocessingWorkflow(cfg.Preprocessing).Execute,
    workflow.RegisterOptions{Name: cfg.Preprocessing.WorkflowName},
)

w.RegisterActivityWithOptions(
    bagcreate.New(cfg.Preprocessing.BagCreate).Execute,
    activity.RegisterOptions{Name: bagcreate.Name},
)
```

The complete command also installs the logging interceptor, enables Temporal
sessions, applies the configured session limit, and starts the worker.

Each new activity and any new workflows must be registered here for the worker
to run it, otherwise an error will returned at run time.

These values must agree with Enduro:

1. Enduro's global `[temporal].namespace` and the worker's
   `[temporal].namespace` must match.
2. The worker's `taskQueue` must match the child workflow `taskQueue` in
   Enduro.
3. The registered Workflow Type name must match the child `workflowName` in
   Enduro.

See [Enduro child workflow configuration] for all child workflow settings.

## Configuration and logging

An example worker configuration is:

```toml
debug = false
verbosity = 0

[temporal]
address = "temporal-frontend.enduro-sdps:7233"
namespace = "default"

[worker]
taskQueue = "custom-enduro"
maxConcurrentSessions = 1

[preprocessing]
workflowName = "preprocessing"
sharedPath = "/home/enduro/shared"

[preprocessing.bagCreate]
checksumAlgorithm = "sha512"
```

The worker requires a configuration file. It searches the working directory,
`$HOME/.config`, and `/etc` for `custom-enduro-worker.*`, or accepts a path
through `--config`. Environment variables override values from that file. They
use the `CUSTOM_ENDURO_WORKER` prefix and replace dots with underscores.

Configuration validation checks the Temporal address and namespace, task
queue, session limit, workflow name, shared path, and activity settings.

Logs are written to standard error as JSON by default. Set `debug = true` for
readable text and use `verbosity` to control the log level. The Temporal client
and worker use the same logger.

## Develop with Enduro

Local development runs the worker through [Enduro's development environment].
Clone this repository as a sibling of Enduro. In Enduro's `.tilt.env`, load the
template Tiltfile and mount the preprocessing volume in the active Enduro
worker:

```dotenv
CHILD_WORKFLOW_PATHS=../custom-enduro-workflows
MOUNT_PREPROCESSING_VOLUME=true
```

Add the child workflow section in Enduro's configuration file:

```toml
[[childWorkflows]]
type = "preprocessing"
taskQueue = "custom-enduro"
workflowName = "preprocessing"
extract = false
sharedPath = "/home/enduro/preprocessing"
```

Then start the Enduro cluster from Enduro's project directory:

```shell
tilt up
```

The template worker uses manual trigger mode by default. Trigger its Tilt
resource after a code change, or set `TRIGGER_MODE_AUTO=true` in this project's
`.tilt.env` to rebuild and deploy it automatically. See [Enduro Tilt
configuration] for the general cluster requirements and environment settings.

## Makefile

The project requires the Go version declared in `go.mod`, GNU Make, GCC, and
Docker. The Makefile installs its Go tools through [bine].

Run `make help` to list every target. Common targets are:

```shell
make tools
make fmt
make lint
make test
make test-race
make validate-tilt
make pre-commit
```

## Reusable activities

The [temporal-activities repository] lists reusable packages and their
configuration. Use an existing activity when its input, output, and behavior
match the workflow; implement one in the project when policy is specific to an
organization or requires a different contract.

## Related projects

* [sfa-enduro-workflows] contains preprocessing and two poststorage workflows,
  including examples of human decisions, structured task failures, custom
  metadata, and AIP integrations.
* [cva-enduro-workflows] contains preprocessing and postbatch workflows,
  including batch result processing and a worker that registers more than one
  extension type.

[`cmd/worker/workercmd/cmd.go`]: cmd/worker/workercmd/cmd.go
[bine]: https://github.com/artefactual-labs/bine
[cva-enduro-workflows]: https://github.com/artefactual-sdps/cva-enduro-workflows
[Enduro child workflow configuration]: https://enduro.readthedocs.io/admin-manual/configuration/#child-workflows
[Enduro custom child workflow guide]: https://enduro.readthedocs.io/dev-manual/child-workflows/
[Enduro Tilt configuration]: https://enduro.readthedocs.io/dev-manual/devel/#tilt-environment-configuration
[Enduro's development environment]: https://enduro.readthedocs.io/dev-manual/devel/
[extraction]: https://enduro.readthedocs.io/dev-manual/child-workflows/preprocessing/#extraction
[sfa-enduro-workflows]: https://github.com/artefactual-sdps/sfa-enduro-workflows
[shared path]: https://enduro.readthedocs.io/dev-manual/child-workflows/preprocessing/#shared-path
[temporal-activities repository]: https://github.com/artefactual-sdps/temporal-activities
