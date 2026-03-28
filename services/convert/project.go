// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package convert

import (
	"context"

	"code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/models/perm"
	project_model "code.gitea.io/gitea/models/project"
	"code.gitea.io/gitea/models/unit"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/cache"
	api "code.gitea.io/gitea/modules/structs"
)

// ToProjectRef converts a project to a minimal reference for embedding in issue/PR responses.
func ToProjectRef(p *project_model.Project) *api.ProjectRef {
	if p == nil {
		return nil
	}
	return &api.ProjectRef{
		ID:    p.ID,
		Title: p.Title,
	}
}

// canDoerSeeProject checks if the doer has permission to see a project.
// For repo-level projects, repo read access is sufficient (already checked by API handler).
// For org/user-level projects, checks org visibility and projects unit permission.
// Results are cached per owner ID in the provided EphemeralCache.
func canDoerSeeProject(ctx context.Context, permCache *cache.EphemeralCache, doer *user_model.User, p *project_model.Project) bool {
	if p.RepoID > 0 {
		return true
	}
	if p.OwnerID == 0 {
		return false
	}
	if doer != nil && doer.IsAdmin {
		return true
	}
	accessMode, _ := cache.GetWithEphemeralCache(ctx, permCache, "org-project-perm", p.OwnerID, func(ctx context.Context, ownerID int64) (perm.AccessMode, error) {
		owner, err := user_model.GetUserByID(ctx, ownerID)
		if err != nil {
			return perm.AccessModeNone, err
		}
		if !organization.HasOrgOrUserVisible(ctx, owner, doer) {
			return perm.AccessModeNone, nil
		}
		return organization.OrgFromUser(owner).UnitPermission(ctx, doer, unit.TypeProjects), nil
	})
	return accessMode >= perm.AccessModeRead
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
