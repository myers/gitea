// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package convert

import (
	issues_model "code.gitea.io/gitea/models/issues"
	project_model "code.gitea.io/gitea/models/project"
	api "code.gitea.io/gitea/modules/structs"
)

// ToAPIProjectMeta converts a project to a lightweight API representation
// for embedding in issue/PR responses. Uses pre-loaded fields from the issue
// (no additional DB queries).
func ToAPIProjectMeta(issue *issues_model.Issue, p *project_model.Project) *api.ProjectMeta {
	state := api.StateOpen
	if p.IsClosed {
		state = api.StateClosed
	}

	result := &api.ProjectMeta{
		ID:          p.ID,
		Title:       p.Title,
		Description: p.Description,
		State:       state,
		Created:     p.CreatedUnix.AsTime(),
		Updated:     p.UpdatedUnix.AsTimePtr(),
	}
	if p.IsClosed {
		result.Closed = p.ClosedDateUnix.AsTimePtr()
	}

	if issue != nil && issue.ProjectBoardID > 0 {
		result.ColumnID = issue.ProjectBoardID
		result.Column = issue.ProjectBoardTitle
	}
	return result
}

// ToAPIProject converts a project model to its full API representation
// for project API endpoints.
func ToAPIProject(p *project_model.Project) *api.Project {
	result := &api.Project{
		ID:           p.ID,
		Title:        p.Title,
		Description:  p.Description,
		TemplateType: uint8(p.TemplateType),
		CardType:     uint8(p.CardType),
		OpenIssues:   p.NumOpenIssues,
		ClosedIssues: p.NumClosedIssues,
		Created:      p.CreatedUnix.AsTime(),
		Updated:      p.UpdatedUnix.AsTimePtr(),
	}
	if p.IsClosed {
		result.State = api.StateClosed
		result.Closed = p.ClosedDateUnix.AsTimePtr()
	} else {
		result.State = api.StateOpen
	}
	return result
}

// ToAPIProjectColumn converts a column model to API struct
func ToAPIProjectColumn(c *project_model.Column) *api.ProjectColumn {
	return &api.ProjectColumn{
		ID:      c.ID,
		Title:   c.Title,
		Color:   c.Color,
		Sorting: int(c.Sorting),
		Default: c.Default,
		Created: c.CreatedUnix.AsTime(),
		Updated: c.UpdatedUnix.AsTime(),
	}
}

// ToAPIProjectCard converts a project issue model to API struct
func ToAPIProjectCard(pi *project_model.ProjectIssue) *api.ProjectCard {
	return &api.ProjectCard{
		ID:       pi.ID,
		IssueID:  pi.IssueID,
		ColumnID: pi.ProjectColumnID,
		Sorting:  pi.Sorting,
	}
}
