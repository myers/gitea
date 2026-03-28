// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package project

import (
	"context"

	project_model "code.gitea.io/gitea/models/project"
	user_model "code.gitea.io/gitea/models/user"
)

// CreateProjectOptions represents options for creating a project
type CreateProjectOptions struct {
	Title        string
	Description  string
	TemplateType project_model.TemplateType
	CardType     project_model.CardType
}

// UpdateProjectOptions represents options for updating a project
type UpdateProjectOptions struct {
	Title       *string
	Description *string
	CardType    *project_model.CardType
}

// CreateProject creates a new project for any owner type
func CreateProject(ctx context.Context, ownerID, repoID int64, projectType project_model.Type, creator *user_model.User, opts CreateProjectOptions) (*project_model.Project, error) {
	project := &project_model.Project{
		Title:        opts.Title,
		Description:  opts.Description,
		CreatorID:    creator.ID,
		Type:         projectType,
		TemplateType: opts.TemplateType,
		CardType:     opts.CardType,
	}

	if projectType == project_model.TypeRepository {
		project.RepoID = repoID
	} else {
		project.OwnerID = ownerID
	}

	return project, project_model.NewProject(ctx, project)
}

// UpdateProject updates an existing project's metadata fields.
// Status changes (open/close) should be handled separately via ChangeProjectStatus.
func UpdateProject(ctx context.Context, project *project_model.Project, opts UpdateProjectOptions) error {
	if opts.Title != nil {
		project.Title = *opts.Title
	}
	if opts.Description != nil {
		project.Description = *opts.Description
	}
	if opts.CardType != nil {
		project.CardType = *opts.CardType
	}

	return project_model.UpdateProject(ctx, project)
}

// DeleteProject deletes a project with all its columns and cards
func DeleteProject(ctx context.Context, project *project_model.Project) error {
	return project_model.DeleteProjectByID(ctx, project.ID)
}

// ChangeProjectStatus changes the open/closed status of a project
func ChangeProjectStatus(ctx context.Context, project *project_model.Project, isClosed bool) error {
	if project.IsClosed == isClosed {
		return nil // no change needed
	}
	return project_model.ChangeProjectStatus(ctx, project, isClosed)
}
