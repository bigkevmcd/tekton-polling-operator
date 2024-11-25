# tekton-polling-operator ![Go](https://github.com/bigkevmcd/tekton-polling-operator/workflows/Go/badge.svg)

**NOTE: This has been deprecated in favour of a different approach at https://github.com/gitops-tools/gitpoller-controller - it's still a bit early, but it's a more maintainable approach**

A simple git repository poller.

This is a Git repository poller that detects changes in a Git repository, and triggers the execution of a Tekton PipelineRun.

When polling GitHub, it uses a special endpoint that should not consume any API tokens, and the schedule can be configured per repository.

## Installation

This operator requires Tekton Pipelines to be installed first, the installation
instructions are [here](https://github.com/tektoncd/pipeline/blob/master/docs/install.md).

```shell
$ kubectl apply -f https://storage.googleapis.com/tekton-releases/pipeline/previous/v0.14.2/release.yaml
```

Then you'll need to install the polling-operator.

```shell
$ kubectl apply -f https://github.com/bigkevmcd/tekton-polling-operator/releases/download/v0.4.0/release-v0.4.0.yaml
```

## GitHub

This polls a GitHub repository, and triggers pipeline runs when the SHA of the
a specific ref changes.

It _does not_ impact on API rate limits to do this, instead it uses the method documented
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
    serviceAccountName: demo-sa # Optional ServiceAccount to execute the pipeline
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
  type: github # can also be gitlab
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

This will fetch the secret, and get the value in `token` and use that to authenticate the API call to GitHub. The secret may contain multiple values and you can configure which key within the `Secret` by setting the `key` field in the `spec.auth` configuration, this defaults to `token`.
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: github-user-pass
  annotations:
    tekton.dev/git-0: https://github.com/
type: kubernetes.io/basic-auth
stringData:
    username: "githubUsername"
    password: "githubAccessToken(PAT)"
```
In such a case, the auth part will be:
```yaml
  auth:
    secretRef:
      name:  github-user-pass
    key: password
```
## Creating PipelineRuns in other namespaces

See the documentation [here](docs/configuring_security.md) for how to grant
access for the operator to create pipelineruns in different namespaces.

## Resource bindings

Yes...resources are deprecated and probably going away in their current form,
but you might want to still use them...

```yaml
apiVersion: polling.tekton.dev/v1alpha1
kind: Repository
metadata:
  name: demo-repository
spec:
  url: https://github.com/bigkevmcd/tekton-polling-operator.git
  ref: main
  type: github
  frequency: 30s
  pipelineRef:
    name: github-poll-pipeline-with-resource
    params:
    - name: sha
      expression: commit.sha
    - name: repoURL
      expression: repoURL
    resources:
    - name: app-git
      resourceSpec:
        type: git
        params:
        - name: revision
          value: $(params.sha)
        - name: url
          value: $(params.repoURL)
```

## Workspaces

Pipelines use workspaces to communicate between tasks, and this allows you to mount a workspace in the pipeline:

```yaml
apiVersion: polling.tekton.dev/v1alpha1
kind: Repository
metadata:
  name: example-repository
spec:
  url: https://github.com/my-org/my-repo.git
  ref: main
  frequency: 5m
  type: github
  pipelineRef:
    name: github-poll-pipeline
    serviceAccountName: example-sa
    params:
    - name: sha
      expression: commit.sha
    - name: repoURL
      expression: repoURL
    workspaces:
      - name: git-source
        persistentVolumeClaim:
          claimName: tekton-volume
```

## Local Development

This uses the operator-sdk, and hasn't yet been upgraded to work with newer
versions than v0.19

### Running tests

```shell
$ go test -v ./...
```

### Building an image

```shell
$ operator-sdk build <docker image reference> # This requires pre-v1.0.0 of the operator-sdk
```

## FAQ

### Do you support TriggerBindings and TriggerTemplates?

Not yet, they're fairly tightly bound to TektonTriggers just now, so this will
require some work to tease that apart, but the plan is to support them.

### How do I insert a static string in to a CEL param?

Ahhh...this is a trick...

```yaml
params:
- name: this-text
  expression: "'this-message'"
```

Note the use of _double_ quotes and _single_ quotes, the value of this is the
string `'this-message'`.

YAML makes this tricky, but now you know...
