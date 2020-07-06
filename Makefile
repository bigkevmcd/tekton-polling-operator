.PHONY: update-crd
update-crd:
	operator-sdk generate k8s
	operator-sdk generate crds

.PHONY: apply-crd
apply-crd:
	kubectl apply -f deploy/crds/polling.tekton.dev_repositories_crd.yaml
