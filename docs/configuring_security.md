# Configuring RBAC for other pipelines.

If you want to be able to execute pipelineruns in namespaces, other than the one
that the operator is deployed into, you'll need to grant extra access for the
controller.

There are two parts to this:

## ClusterRole

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: polling-operator-cluster-role
rules:
- apiGroups:
  - tekton.dev
  resources:
  - pipelineruns
  verbs:
  - create
```

This is a simple cluster role that only grants permission to create
pipelineruns, this is available in the [the examples](../examples/cluster_role.yaml).

## RoleBinding

You will need to grant the operator this role in your namespace.

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: polling-operator-role
  namespace: # insert namespace you need to grant access to
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: polling-operator-cluster-role
subjects:
- kind: ServiceAccount
  name: tekton-polling-operator
  namespace: # insert the namespace you deployed the operator in
```
