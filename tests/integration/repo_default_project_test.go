// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"testing"

	"code.gitea.io/gitea/models/db"
	issues_model "code.gitea.io/gitea/models/issues"
	project_model "code.gitea.io/gitea/models/project"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unit"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	issue_service "code.gitea.io/gitea/services/issue"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultProjectAutoAssignIssue(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})

	// Create a project for this repo
	project := &project_model.Project{
		Title:  "Test Default Project",
		RepoID: repo.ID,
		Type:   project_model.TypeRepository,
	}
	require.NoError(t, project_model.NewProject(t.Context(), project))
	assert.Greater(t, project.ID, int64(0))

	// Configure the repo to auto-assign issues to this project
	repoUnit, err := repo.GetUnit(t.Context(), unit.TypeProjects)
	require.NoError(t, err)
	cfg := repoUnit.ProjectsConfig()
	cfg.DefaultProjectID = project.ID
	cfg.AutoAssignIssues = true
	cfg.AutoAssignPRs = false
	repoUnit.Config = cfg
	_, err = db.GetEngine(t.Context()).ID(repoUnit.ID).Cols("config").Update(repoUnit)
	require.NoError(t, err)

	// Create an issue without specifying a project (projectID = 0)
	issue := &issues_model.Issue{
		RepoID:   repo.ID,
		PosterID: user.ID,
		Poster:   user,
		Title:    "Test auto-assign issue",
		Content:  "This should be auto-assigned to the default project",
	}
	err = issue_service.NewIssue(t.Context(), repo, issue, nil, nil, nil, 0)
	require.NoError(t, err)

	// Verify the issue was assigned to the project
	var pi project_model.ProjectIssue
	has, err := db.GetEngine(t.Context()).Where("issue_id=?", issue.ID).Get(&pi)
	require.NoError(t, err)
	assert.True(t, has, "issue should be assigned to a project")
	assert.Equal(t, project.ID, pi.ProjectID)
}

func TestDefaultProjectNoAutoAssignWhenDisabled(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})

	// Create a project but do NOT enable auto-assign
	project := &project_model.Project{
		Title:  "Disabled Default Project",
		RepoID: repo.ID,
		Type:   project_model.TypeRepository,
	}
	require.NoError(t, project_model.NewProject(t.Context(), project))

	repoUnit, err := repo.GetUnit(t.Context(), unit.TypeProjects)
	require.NoError(t, err)
	cfg := repoUnit.ProjectsConfig()
	cfg.DefaultProjectID = project.ID
	cfg.AutoAssignIssues = false
	cfg.AutoAssignPRs = false
	repoUnit.Config = cfg
	_, err = db.GetEngine(t.Context()).ID(repoUnit.ID).Cols("config").Update(repoUnit)
	require.NoError(t, err)

	// Create an issue — should NOT be auto-assigned
	issue := &issues_model.Issue{
		RepoID:   repo.ID,
		PosterID: user.ID,
		Poster:   user,
		Title:    "Should not auto-assign",
		Content:  "AutoAssignIssues is false",
	}
	err = issue_service.NewIssue(t.Context(), repo, issue, nil, nil, nil, 0)
	require.NoError(t, err)

	var pi project_model.ProjectIssue
	has, err := db.GetEngine(t.Context()).Where("issue_id=?", issue.ID).Get(&pi)
	require.NoError(t, err)
	assert.False(t, has, "issue should NOT be assigned to a project when auto-assign is disabled")
}

func TestDefaultProjectExplicitProjectWins(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})

	// Create two projects
	defaultProject := &project_model.Project{
		Title:  "Default Project",
		RepoID: repo.ID,
		Type:   project_model.TypeRepository,
	}
	require.NoError(t, project_model.NewProject(t.Context(), defaultProject))

	explicitProject := &project_model.Project{
		Title:  "Explicit Project",
		RepoID: repo.ID,
		Type:   project_model.TypeRepository,
	}
	require.NoError(t, project_model.NewProject(t.Context(), explicitProject))

	// Configure default project
	repoUnit, err := repo.GetUnit(t.Context(), unit.TypeProjects)
	require.NoError(t, err)
	cfg := repoUnit.ProjectsConfig()
	cfg.DefaultProjectID = defaultProject.ID
	cfg.AutoAssignIssues = true
	repoUnit.Config = cfg
	_, err = db.GetEngine(t.Context()).ID(repoUnit.ID).Cols("config").Update(repoUnit)
	require.NoError(t, err)

	// Create issue with EXPLICIT project — should use explicit, not default
	issue := &issues_model.Issue{
		RepoID:   repo.ID,
		PosterID: user.ID,
		Poster:   user,
		Title:    "Explicit project wins",
	}
	err = issue_service.NewIssue(t.Context(), repo, issue, nil, nil, nil, explicitProject.ID)
	require.NoError(t, err)

	var pi project_model.ProjectIssue
	has, err := db.GetEngine(t.Context()).Where("issue_id=?", issue.ID).Get(&pi)
	require.NoError(t, err)
	assert.True(t, has)
	assert.Equal(t, explicitProject.ID, pi.ProjectID, "explicit project should win over default")
}
