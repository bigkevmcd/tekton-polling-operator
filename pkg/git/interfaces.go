package git

import (
	pollingv1alpha1 "github.com/bigkevmcd/tekton-polling-operator/api/v1alpha1"
)

// CommitPoller implementations can check with an upstream Git hosting service
// to determine the current SHA and ETag.
type CommitPoller interface {
	Poll(repo string, ps pollingv1alpha1.PollStatus) (pollingv1alpha1.PollStatus, error)
}
