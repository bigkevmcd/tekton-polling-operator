# tekton-polling-operator

A simple git repository poller.

## Installation

This operator requires Tekton Pipelines to be installed first, the installation
instructions are [here](https://github.com/tektoncd/pipeline/blob/master/docs/install.md).

```shell
$ kubectl apply -f https://storage.googleapis.com/tekton-releases/pipeline/previous/v0.14.2/release.yaml
```

Then you'll need to install the polling-operator.

```shell
$ kubectl apply -f https://github.com/bigkevmcd/tekton-polling-operator/releases/download/v0.1.0/release-v0.1.0.yaml
```

## GitHub

This polls a GitHub repository, and triggers pipeline runs when the SHA of the
a specific ref changes.

It _does not_ use API tokens to do this, instead it uses the method documented
[here](https://developer.github.com/changes/2016-02-24-commit-reference-sha-api/)
and the ETag to fetch the commit.

## GitLab

This polls a GitLab repository, and triggers pipeline runs when the SHA of the
a specific ref changes.

## Pipelines

You'll want a pipeline to be executed on change.

```yaml
apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  name: demo-pipeline
spec:
  params:
  - name: sha
    type: string
    description: "the SHA of the recently detected change"
  - name: repoURL
    type: string
    description: "the cloneURL that the change was detected in"
  tasks:
    # insert the tasks below
```

This pipeline accepts two parameters, the new commit SHA, and the repoURL.

 sample pipeline is provided in the [examples](./examples) directory.

## Monitoring a Repository

To monitor a repository for changes, you'll need to create a `Repository` object
in Kubernetes.

```yaml
apiVersion: polling.tekton.dev/v1alpha1
kind: Repository
metadata:
  name: example-repository
spec:
  url: https://github.com/my-org/my-repo.git
  ref: main
  frequency: 5m
  type: github # can also be gitlab
  pipelineRef:
    name: github-poll-pipeline
    namespace: test-ns # optional: if provided, the pipelinerun will be created in this namespace to reference the pipeline.
    params:
    - name: sha
      expression: commit.sha
    - name: repoURL
      expression: repoURL
```

This defines a repository that monitors the `main` branch in
`https://github.com/my-org/my-repo.git`, checking every 5 minutes, and executing
the `github-poll-pipeline` when a change is detected.

The parameters are extracted from the commit body, the expressions are
[CEL](https://github.com/google/cel-go) expressions.

The expressions can access the data from the commit as `commit` and the
configured repository URL as `repoURL`.

For GitHub repositories, the commit data will have the structure [here](https://developer.github.com/v3/repos/commits/#get-a-commit).

You can also monitor `GitLab` repositories, specifying the type as `gitlab`.

In this case, the commit data will have the structure [here](https://docs.gitlab.com/ee/api/commits.html#list-repository-commits).

## Authenticating against a Private Repository

Of course, not every repo is public, to authenticate your requests, you'll
need to provide an auth token.

```yaml
apiVersion: polling.tekton.dev/v1alpha1
kind: Repository
metadata:
  name: example-repository
spec:
  url: https://github.com/my-org/my-repo.git
  ref: main
  frequency: 2m
  pipelineRef:
    name: github-poll-pipeline
    params:
    - name: sha
      expression: commit.sha
    - name: repoURL
      expression: repoURL
  auth:
    secretRef:
      name:  my-github-secret
    key: token
```

This will fetch the secret, and get the value in `token` and use that to
authenticate the API call to GitHub.

## Creating PipelineRuns in other namespaces

See the documentation [here](docs/configuring_security.md) for how to grant
access for the operator to create pipelineruns in different namespaces.
