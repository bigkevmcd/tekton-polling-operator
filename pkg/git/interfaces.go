package git

import (
	pollingv1 "github.com/bigkevmcd/tekton-polling-operator/pkg/apis/polling/v1alpha1"
)

// Commit is a polled Commit, specific to each implementation.
type Commit map[string]interface{}

// CommitPoller implementations can check with an upstream Git hosting service
// to determine the current SHA and ETag.
type CommitPoller interface {
	Poll(repo string, ps pollingv1.PollStatus) (pollingv1.PollStatus, Commit, error)
}
