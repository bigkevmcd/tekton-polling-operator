# Workspace example

You **MUST** edit the `repository.yaml` and put your own repository in for this
to work.

Then you can apply this example:

```shell
$ kustomize build | kubectl apply -f -
```

**NOTE:** The workspace is not currently used in the task, it's there to
illustrate how to configure the resources.
