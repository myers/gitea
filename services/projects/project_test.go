// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package project

import (
	"testing"

	project_model "code.gitea.io/gitea/models/project"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"

	"github.com/stretchr/testify/assert"
)

func TestCreateProject(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	t.Run("Repository project", func(t *testing.T) {
		p, err := CreateProject(t.Context(), 0, 1, project_model.TypeRepository, user, CreateProjectOptions{
			Title:        "Repo Project",
			Description:  "A test project",
			TemplateType: project_model.TemplateTypeBasicKanban,
			CardType:     project_model.CardTypeTextOnly,
		})
		assert.NoError(t, err)
		assert.Equal(t, "Repo Project", p.Title)
		assert.Equal(t, "A test project", p.Description)
		assert.Equal(t, int64(1), p.RepoID)
		assert.Equal(t, int64(0), p.OwnerID)
		assert.Equal(t, project_model.TypeRepository, p.Type)
		assert.Equal(t, user.ID, p.CreatorID)
		assert.False(t, p.IsClosed)

		// Verify columns were created (BasicKanban template)
		columns, err := p.GetColumns(t.Context())
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(columns), 2) // At least Backlog + template columns
	})

	t.Run("Organization project", func(t *testing.T) {
		p, err := CreateProject(t.Context(), 3, 0, project_model.TypeOrganization, user, CreateProjectOptions{
			Title: "Org Project",
		})
		assert.NoError(t, err)
		assert.Equal(t, int64(3), p.OwnerID)
		assert.Equal(t, int64(0), p.RepoID)
		assert.Equal(t, project_model.TypeOrganization, p.Type)
	})

	t.Run("Individual project", func(t *testing.T) {
		p, err := CreateProject(t.Context(), user.ID, 0, project_model.TypeIndividual, user, CreateProjectOptions{
			Title: "My Project",
		})
		assert.NoError(t, err)
		assert.Equal(t, user.ID, p.OwnerID)
		assert.Equal(t, project_model.TypeIndividual, p.Type)
	})
}

func TestUpdateProject(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	// Create a project to update
	p, err := CreateProject(t.Context(), 0, 1, project_model.TypeRepository, user, CreateProjectOptions{
		Title:       "Original Title",
		Description: "Original Description",
		CardType:    project_model.CardTypeTextOnly,
	})
	assert.NoError(t, err)

	t.Run("Update title only", func(t *testing.T) {
		newTitle := "Updated Title"
		err := UpdateProject(t.Context(), p, UpdateProjectOptions{
			Title: &newTitle,
		})
		assert.NoError(t, err)
		assert.Equal(t, "Updated Title", p.Title)
		assert.Equal(t, "Original Description", p.Description) // unchanged
	})

	t.Run("Update description only", func(t *testing.T) {
		newDesc := "Updated Description"
		err := UpdateProject(t.Context(), p, UpdateProjectOptions{
			Description: &newDesc,
		})
		assert.NoError(t, err)
		assert.Equal(t, "Updated Description", p.Description)
		assert.Equal(t, "Updated Title", p.Title) // unchanged from previous
	})

	t.Run("Update card type", func(t *testing.T) {
		newCardType := project_model.CardTypeImagesAndText
		err := UpdateProject(t.Context(), p, UpdateProjectOptions{
			CardType: &newCardType,
		})
		assert.NoError(t, err)
		assert.Equal(t, project_model.CardTypeImagesAndText, p.CardType)
	})

	t.Run("No-op when no fields set", func(t *testing.T) {
		err := UpdateProject(t.Context(), p, UpdateProjectOptions{})
		assert.NoError(t, err)
	})
}

func TestDeleteProject(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	p, err := CreateProject(t.Context(), 0, 1, project_model.TypeRepository, user, CreateProjectOptions{
		Title:        "To Delete",
		TemplateType: project_model.TemplateTypeBasicKanban,
	})
	assert.NoError(t, err)

	// Verify it exists
	_, err = project_model.GetProjectByID(t.Context(), p.ID)
	assert.NoError(t, err)

	// Delete it
	err = DeleteProject(t.Context(), p)
	assert.NoError(t, err)

	// Verify it's gone
	_, err = project_model.GetProjectByID(t.Context(), p.ID)
	assert.True(t, project_model.IsErrProjectNotExist(err))

	// Verify columns were cascaded
	columns, err := p.GetColumns(t.Context())
	assert.NoError(t, err)
	assert.Empty(t, columns)
}

func TestChangeProjectStatus(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	t.Run("Close and reopen", func(t *testing.T) {
		p, err := CreateProject(t.Context(), 0, 1, project_model.TypeRepository, user, CreateProjectOptions{
			Title: "Status Test",
		})
		assert.NoError(t, err)
		assert.False(t, p.IsClosed)

		// Close it
		err = ChangeProjectStatus(t.Context(), p, true)
		assert.NoError(t, err)

		// Re-read to verify
		got, err := project_model.GetProjectByID(t.Context(), p.ID)
		assert.NoError(t, err)
		assert.True(t, got.IsClosed)

		// Reopen it
		err = ChangeProjectStatus(t.Context(), got, false)
		assert.NoError(t, err)

		got, err = project_model.GetProjectByID(t.Context(), p.ID)
		assert.NoError(t, err)
		assert.False(t, got.IsClosed)
	})

	t.Run("No-op when status unchanged", func(t *testing.T) {
		p, err := CreateProject(t.Context(), 0, 1, project_model.TypeRepository, user, CreateProjectOptions{
			Title: "No-op Test",
		})
		assert.NoError(t, err)

		// Already open, closing to open is no-op
		err = ChangeProjectStatus(t.Context(), p, false)
		assert.NoError(t, err)
	})
}
