// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package swagger

import api "code.gitea.io/gitea/modules/structs"

// swagger:response Project
type swaggerResponseProject struct {
	// in:body
	Body api.Project `json:"body"`
}

// swagger:response ProjectList
type swaggerResponseProjectList struct {
	// in:body
	Body []api.Project `json:"body"`
}

// swagger:response ProjectColumn
type swaggerResponseProjectColumn struct {
	// in:body
	Body api.ProjectColumn `json:"body"`
}

// swagger:response ProjectColumnList
type swaggerResponseProjectColumnList struct {
	// in:body
	Body []api.ProjectColumn `json:"body"`
}

// swagger:response ProjectCard
type swaggerResponseProjectCard struct {
	// in:body
	Body api.ProjectCard `json:"body"`
}

// swagger:response ProjectCardList
type swaggerResponseProjectCardList struct {
	// in:body
	Body []api.ProjectCard `json:"body"`
}
