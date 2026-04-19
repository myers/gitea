// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package project

import (
	"context"
	"fmt"

	issues_model "code.gitea.io/gitea/models/issues"
	project_model "code.gitea.io/gitea/models/project"
	user_model "code.gitea.io/gitea/models/user"
)

// AddCardToColumn adds an issue to a project column with validation.
// It validates that the issue exists and belongs to the project's repository
// (for repo-scoped projects).
func AddCardToColumn(ctx context.Context, project *project_model.Project, column *project_model.Column, issueID, sorting int64) (*project_model.ProjectIssue, error) {
	// Validate issue exists
	issue, err := issues_model.GetIssueByID(ctx, issueID)
	if err != nil {
		return nil, err
	}

	// Validate issue belongs to project's repository (for repo projects)
	if project.RepoID > 0 && issue.RepoID != project.RepoID {
		return nil, fmt.Errorf("issue %d does not belong to project's repository", issueID)
	}

	if err := project_model.AddIssueToProject(ctx, project.ID, column.ID, issueID, sorting); err != nil {
		return nil, err
	}

	return project_model.GetProjectCard(ctx, project.ID, issueID)
}

// RemoveCardFromProject removes an issue from a project
func RemoveCardFromProject(ctx context.Context, project *project_model.Project, issueID int64) error {
	return project_model.RemoveIssueFromProject(ctx, project.ID, issueID)
}

// MoveCard moves a card to a target column at the given sorting position.
// It validates the target column belongs to the same project and creates
// timeline comments for column changes.
func MoveCard(ctx context.Context, doer *user_model.User, project *project_model.Project, card *project_model.ProjectIssue, targetColumn *project_model.Column, sorting int64) error {
	if targetColumn.ProjectID != project.ID {
		return fmt.Errorf("target column %d does not belong to project %d", targetColumn.ID, project.ID)
	}

	// Use the existing MoveIssuesOnProjectColumn service which handles
	// timeline comments and two-phase sorting updates
	return MoveIssuesOnProjectColumn(ctx, doer, targetColumn, map[int64]int64{sorting: card.IssueID})
}
