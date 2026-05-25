// Copyright (c) 2026 NorvikOps. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package admin

import (
	"time"

	"github.com/hibiken/asynq"
)

const (
	TaskDeleteOrg        = "admin:org:delete"
	TaskSCIMTokenExpiry  = "admin:scim:token_expiry"
)

// NewSCIMTokenExpiryTask creates the daily SCIM token auto-revocation task.
func NewSCIMTokenExpiryTask() *asynq.Task {
	return asynq.NewTask(TaskSCIMTokenExpiry, nil, asynq.Unique(23*time.Hour))
}
