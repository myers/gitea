// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package structs

// ProjectRef is a project summary embedded in issue/PR responses.
// swagger:model
type ProjectRef struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
}
