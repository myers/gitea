// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"testing"

	auth_model "code.gitea.io/gitea/models/auth"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
)

func TestAPIListProjects(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// repo1 (user2/repo1) has project ID 1 (open) in fixtures
	session := loginUser(t, "user2")
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeReadIssue)

	// List open projects (default)
	req := NewRequest(t, "GET", "/api/v1/repos/user2/repo1/projects").
		AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	var projects []api.Project
	DecodeJSON(t, resp, &projects)
	assert.NotEmpty(t, projects)
	for _, p := range projects {
		assert.Equal(t, api.StateOpen, p.State)
	}

	// Verify pagination header
	assert.NotEmpty(t, resp.Header().Get("X-Total-Count"))

	// List closed projects — fixture project 1 is open, so closed list should be empty for repo1
	req = NewRequest(t, "GET", "/api/v1/repos/user2/repo1/projects?state=closed").
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)

	var closedProjects []api.Project
	DecodeJSON(t, resp, &closedProjects)
	assert.Empty(t, closedProjects)

	// List all projects
	req = NewRequest(t, "GET", "/api/v1/repos/user2/repo1/projects?state=all").
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)

	var allProjects []api.Project
	DecodeJSON(t, resp, &allProjects)
	assert.GreaterOrEqual(t, len(allProjects), len(projects))
}

func TestAPICreateProject(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user2")
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteIssue)

	// Create a new project
	req := NewRequestWithJSON(t, "POST", "/api/v1/repos/user2/repo1/projects", api.CreateProjectOption{
		Title:        "API Test Project",
		Description:  "Created via API",
		TemplateType: 1,
	}).AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusCreated)

	var project api.Project
	DecodeJSON(t, resp, &project)
	assert.Equal(t, "API Test Project", project.Title)
	assert.Equal(t, "Created via API", project.Description)
	assert.Equal(t, api.StateOpen, project.State)
	assert.Positive(t, project.ID)

	// Verify the new project appears in the list
	req = NewRequest(t, "GET", "/api/v1/repos/user2/repo1/projects?state=all").
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)

	var projects []api.Project
	DecodeJSON(t, resp, &projects)
	found := false
	for _, p := range projects {
		if p.ID == project.ID {
			found = true
			break
		}
	}
	assert.True(t, found, "newly created project should appear in list")
}

func TestAPIGetProject(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user2")
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeReadIssue)

	// Get existing project (fixture ID 1 belongs to repo1)
	req := NewRequest(t, "GET", "/api/v1/repos/user2/repo1/projects/1").
		AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	var project api.Project
	DecodeJSON(t, resp, &project)
	assert.Equal(t, int64(1), project.ID)
	assert.Equal(t, "First project", project.Title)

	// 404 for nonexistent project
	req = NewRequest(t, "GET", "/api/v1/repos/user2/repo1/projects/99999").
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNotFound)
}

func TestAPIEditProject(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user2")
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteIssue)

	// First create a project to edit (avoid mutating shared fixture)
	req := NewRequestWithJSON(t, "POST", "/api/v1/repos/user2/repo1/projects", api.CreateProjectOption{
		Title: "Project to Edit",
	}).AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusCreated)

	var created api.Project
	DecodeJSON(t, resp, &created)

	urlStr := fmt.Sprintf("/api/v1/repos/user2/repo1/projects/%d", created.ID)

	// Update title
	newTitle := "Updated Title"
	req = NewRequestWithJSON(t, "PATCH", urlStr, api.EditProjectOption{
		Title: &newTitle,
	}).AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)

	var edited api.Project
	DecodeJSON(t, resp, &edited)
	assert.Equal(t, "Updated Title", edited.Title)

	// Close the project
	closedState := "closed"
	req = NewRequestWithJSON(t, "PATCH", urlStr, api.EditProjectOption{
		State: &closedState,
	}).AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)

	var closed api.Project
	DecodeJSON(t, resp, &closed)
	assert.Equal(t, api.StateClosed, closed.State)
	assert.NotNil(t, closed.Closed)

	// Reopen the project
	openState := "open"
	req = NewRequestWithJSON(t, "PATCH", urlStr, api.EditProjectOption{
		State: &openState,
	}).AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)

	var reopened api.Project
	DecodeJSON(t, resp, &reopened)
	assert.Equal(t, api.StateOpen, reopened.State)
}

func TestAPIDeleteProject(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user2")
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteIssue)

	// Create a project to delete
	req := NewRequestWithJSON(t, "POST", "/api/v1/repos/user2/repo1/projects", api.CreateProjectOption{
		Title: "Project to Delete",
	}).AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusCreated)

	var project api.Project
	DecodeJSON(t, resp, &project)

	urlStr := fmt.Sprintf("/api/v1/repos/user2/repo1/projects/%d", project.ID)

	// Delete the project
	req = NewRequest(t, "DELETE", urlStr).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNoContent)

	// Verify it's gone
	req = NewRequest(t, "GET", urlStr).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNotFound)
}

func TestAPIProjectColumns(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user2")
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteIssue)

	// Create a project with kanban template (template_type=1 = basic kanban)
	req := NewRequestWithJSON(t, "POST", "/api/v1/repos/user2/repo1/projects", api.CreateProjectOption{
		Title:        "Kanban Project",
		TemplateType: 1,
	}).AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusCreated)

	var project api.Project
	DecodeJSON(t, resp, &project)

	projectURL := fmt.Sprintf("/api/v1/repos/user2/repo1/projects/%d", project.ID)
	columnsURL := projectURL + "/columns"

	// List columns — kanban template creates default columns
	req = NewRequest(t, "GET", columnsURL).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)

	var columns []api.ProjectColumn
	DecodeJSON(t, resp, &columns)
	assert.NotEmpty(t, columns)

	// Find the default column
	var defaultColumn *api.ProjectColumn
	for i := range columns {
		if columns[i].Default {
			defaultColumn = &columns[i]
			break
		}
	}
	assert.NotNil(t, defaultColumn, "project should have a default column")

	// Create a new column
	req = NewRequestWithJSON(t, "POST", columnsURL, api.CreateProjectColumnOption{
		Title: "Custom Column",
		Color: "#ff0000",
	}).AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusCreated)

	var newColumn api.ProjectColumn
	DecodeJSON(t, resp, &newColumn)
	assert.Equal(t, "Custom Column", newColumn.Title)
	assert.Equal(t, "#ff0000", newColumn.Color)

	// Get the new column
	columnURL := fmt.Sprintf("%s/%d", columnsURL, newColumn.ID)
	req = NewRequest(t, "GET", columnURL).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)

	var fetchedColumn api.ProjectColumn
	DecodeJSON(t, resp, &fetchedColumn)
	assert.Equal(t, newColumn.ID, fetchedColumn.ID)
	assert.Equal(t, "Custom Column", fetchedColumn.Title)

	// Edit the column
	updatedTitle := "Renamed Column"
	updatedColor := "#00ff00"
	req = NewRequestWithJSON(t, "PATCH", columnURL, api.EditProjectColumnOption{
		Title: &updatedTitle,
		Color: &updatedColor,
	}).AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)

	var editedColumn api.ProjectColumn
	DecodeJSON(t, resp, &editedColumn)
	assert.Equal(t, "Renamed Column", editedColumn.Title)
	assert.Equal(t, "#00ff00", editedColumn.Color)

	// Delete the non-default column — should succeed
	req = NewRequest(t, "DELETE", columnURL).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNoContent)

	// Delete the default column — should fail with 403
	if defaultColumn != nil {
		defaultColumnURL := fmt.Sprintf("%s/%d", columnsURL, defaultColumn.ID)
		req = NewRequest(t, "DELETE", defaultColumnURL).
			AddTokenAuth(token)
		MakeRequest(t, req, http.StatusForbidden)
	}

	// Move column — create two columns and reorder
	req = NewRequestWithJSON(t, "POST", columnsURL, api.CreateProjectColumnOption{
		Title: "Column A",
	}).AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusCreated)

	var colA api.ProjectColumn
	DecodeJSON(t, resp, &colA)

	moveURL := fmt.Sprintf("%s/%d/move", columnsURL, colA.ID)
	req = NewRequestWithJSON(t, "POST", moveURL, api.MoveProjectColumnOption{
		Sorting: 0,
	}).AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNoContent)
}

func TestAPIProjectCards(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user2")
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteIssue)

	// Use fixture: project 1 has columns (board 1 = "To Do", default), issue 1 is already in project 1
	// Create a fresh project to avoid fixture conflicts
	req := NewRequestWithJSON(t, "POST", "/api/v1/repos/user2/repo1/projects", api.CreateProjectOption{
		Title:        "Card Test Project",
		TemplateType: 1,
	}).AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusCreated)

	var project api.Project
	DecodeJSON(t, resp, &project)

	columnsURL := fmt.Sprintf("/api/v1/repos/user2/repo1/projects/%d/columns", project.ID)

	// Get columns to find the default one
	req = NewRequest(t, "GET", columnsURL).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)

	var columns []api.ProjectColumn
	DecodeJSON(t, resp, &columns)
	assert.NotEmpty(t, columns)

	defaultCol := columns[0]
	cardsURL := fmt.Sprintf("%s/%d/cards", columnsURL, defaultCol.ID)

	// Add issue 4 (exists in repo1 fixtures, not in any project) as a card
	req = NewRequestWithJSON(t, "POST", cardsURL, api.AddProjectCardOption{
		IssueID: 4,
	}).AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusCreated)

	var card api.ProjectCard
	DecodeJSON(t, resp, &card)
	assert.Equal(t, int64(4), card.IssueID)
	assert.Equal(t, defaultCol.ID, card.ColumnID)
	assert.Positive(t, card.ID)

	// List cards in column — should contain the card we just added
	req = NewRequest(t, "GET", cardsURL).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)

	var cards []api.ProjectCard
	DecodeJSON(t, resp, &cards)
	assert.NotEmpty(t, cards)
	found := false
	for _, c := range cards {
		if c.IssueID == 4 {
			found = true
			break
		}
	}
	assert.True(t, found, "issue 4 should be in the column")

	// Add duplicate card — should fail with 422
	req = NewRequestWithJSON(t, "POST", cardsURL, api.AddProjectCardOption{
		IssueID: 4,
	}).AddTokenAuth(token)
	MakeRequest(t, req, http.StatusUnprocessableEntity)

	// Create a second column and move card to it
	req = NewRequestWithJSON(t, "POST", columnsURL, api.CreateProjectColumnOption{
		Title: "Done",
	}).AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusCreated)

	var doneCol api.ProjectColumn
	DecodeJSON(t, resp, &doneCol)

	moveURL := fmt.Sprintf("%s/%d/cards/%d/move", columnsURL, defaultCol.ID, card.ID)
	req = NewRequestWithJSON(t, "POST", moveURL, api.MoveProjectCardOption{
		ColumnID: doneCol.ID,
		Sorting:  0,
	}).AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNoContent)

	// Delete card
	deleteURL := fmt.Sprintf("%s/%d/cards/%d", columnsURL, doneCol.ID, card.ID)
	req = NewRequest(t, "DELETE", deleteURL).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNoContent)

	// Verify card is gone — listing cards in the done column should not contain it
	doneCardsURL := fmt.Sprintf("%s/%d/cards", columnsURL, doneCol.ID)
	req = NewRequest(t, "GET", doneCardsURL).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)

	var remainingCards []api.ProjectCard
	DecodeJSON(t, resp, &remainingCards)
	for _, c := range remainingCards {
		assert.NotEqual(t, int64(4), c.IssueID, "deleted card should not appear")
	}
}

func TestAPIProjectPermissions(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// user4 is NOT a collaborator on user2/repo1, but repo1 is public so read works
	session := loginUser(t, "user4")
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteIssue)

	// Read should succeed — repo1 is public, user4 has read access
	req := NewRequest(t, "GET", "/api/v1/repos/user2/repo1/projects").
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusOK)

	// GET single project should succeed
	req = NewRequest(t, "GET", "/api/v1/repos/user2/repo1/projects/1").
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusOK)

	// GET columns should succeed
	req = NewRequest(t, "GET", "/api/v1/repos/user2/repo1/projects/1/columns").
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusOK)

	// Write operations should fail with 403 — user4 is not a writer
	req = NewRequestWithJSON(t, "POST", "/api/v1/repos/user2/repo1/projects", api.CreateProjectOption{
		Title: "Unauthorized Project",
	}).AddTokenAuth(token)
	MakeRequest(t, req, http.StatusForbidden)

	// PATCH should fail
	newTitle := "Hacked Title"
	req = NewRequestWithJSON(t, "PATCH", "/api/v1/repos/user2/repo1/projects/1", api.EditProjectOption{
		Title: &newTitle,
	}).AddTokenAuth(token)
	MakeRequest(t, req, http.StatusForbidden)

	// DELETE should fail
	req = NewRequest(t, "DELETE", "/api/v1/repos/user2/repo1/projects/1").
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusForbidden)

	// POST column should fail
	req = NewRequestWithJSON(t, "POST", "/api/v1/repos/user2/repo1/projects/1/columns", api.CreateProjectColumnOption{
		Title: "Unauthorized Column",
	}).AddTokenAuth(token)
	MakeRequest(t, req, http.StatusForbidden)
}

func TestAPIListProjectsOnExistingFixtures(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Verify the fixture project (ID 1) for repo1 is accessible and has expected data
	session := loginUser(t, "user2")
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeReadIssue)

	req := NewRequest(t, "GET", "/api/v1/repos/user2/repo1/projects/1").
		AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	var project api.Project
	DecodeJSON(t, resp, &project)
	assert.Equal(t, int64(1), project.ID)
	assert.Equal(t, "First project", project.Title)
	assert.Equal(t, api.StateOpen, project.State)

	// Fixture project 1 has 3 columns: "To Do" (default), "In Progress", "Done"
	req = NewRequest(t, "GET", "/api/v1/repos/user2/repo1/projects/1/columns").
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)

	var columns []api.ProjectColumn
	DecodeJSON(t, resp, &columns)
	assert.Len(t, columns, 3)

	// Verify column titles
	titles := make([]string, len(columns))
	for i, c := range columns {
		titles[i] = c.Title
	}
	assert.Contains(t, titles, "To Do")
	assert.Contains(t, titles, "In Progress")
	assert.Contains(t, titles, "Done")

	// Fixture has cards: issue 1 in board 1, issue 3 in board 2, issue 5 in board 3
	// Check cards in the default "To Do" column (board ID 1)
	var defaultCol api.ProjectColumn
	for _, c := range columns {
		if c.Default {
			defaultCol = c
			break
		}
	}
	assert.NotZero(t, defaultCol.ID, "should find default column")

	cardsURL := fmt.Sprintf("/api/v1/repos/user2/repo1/projects/1/columns/%d/cards", defaultCol.ID)
	req = NewRequest(t, "GET", cardsURL).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)

	var cards []api.ProjectCard
	DecodeJSON(t, resp, &cards)
	// Board 1 (To Do, default) has issue 1 per fixtures
	assert.NotEmpty(t, cards)
}
