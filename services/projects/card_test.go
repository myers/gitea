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

func TestAddCardToColumn(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	// Use fixture project 1 (repo 1) which has columns 1, 2, 3
	project, err := project_model.GetProjectByID(t.Context(), 1)
	assert.NoError(t, err)
	column, err := project_model.GetColumn(t.Context(), 2) // "In Progress" column
	assert.NoError(t, err)

	t.Run("Add issue to column", func(t *testing.T) {
		// Issue 11 exists in repo 1 and is not yet in project 1
		card, err := AddCardToColumn(t.Context(), project, column, 11, -1)
		assert.NoError(t, err)
		assert.Equal(t, int64(11), card.IssueID)
		assert.Equal(t, column.ID, card.ProjectColumnID)
		assert.Equal(t, project.ID, card.ProjectID)
	})

	t.Run("Duplicate card rejected", func(t *testing.T) {
		// Issue 1 is already in project 1 (from fixtures)
		_, err := AddCardToColumn(t.Context(), project, column, 1, -1)
		assert.Error(t, err)
		assert.True(t, project_model.IsErrCardAlreadyInProject(err))
	})

	t.Run("Nonexistent issue", func(t *testing.T) {
		_, err := AddCardToColumn(t.Context(), project, column, 99999, -1)
		assert.Error(t, err)
	})
}

func TestRemoveCardFromProject(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	project, err := project_model.GetProjectByID(t.Context(), 1)
	assert.NoError(t, err)

	t.Run("Remove existing card", func(t *testing.T) {
		// Issue 3 is in project 1 (from fixtures)
		err := RemoveCardFromProject(t.Context(), project, 3)
		assert.NoError(t, err)

		// Verify it's gone
		_, err = project_model.GetProjectCard(t.Context(), project.ID, 3)
		assert.True(t, project_model.IsErrProjectCardNotExist(err))
	})

	t.Run("Remove nonexistent card is silent", func(t *testing.T) {
		err := RemoveCardFromProject(t.Context(), project, 99999)
		assert.NoError(t, err) // no error, silent no-op
	})
}

func TestMoveCard(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	// Use fixture project 1 with columns 1 (default), 2, 3
	project, err := project_model.GetProjectByID(t.Context(), 1)
	assert.NoError(t, err)
	column1, err := project_model.GetColumn(t.Context(), 1)
	assert.NoError(t, err)
	column2, err := project_model.GetColumn(t.Context(), 2)
	assert.NoError(t, err)

	t.Run("Move card between columns", func(t *testing.T) {
		// Issue 1 is in column 1 (from fixtures)
		card, err := project_model.GetProjectCard(t.Context(), project.ID, 1)
		assert.NoError(t, err)
		assert.Equal(t, column1.ID, card.ProjectColumnID)

		// Move to column 2
		err = MoveCard(t.Context(), user, project, card, column2, 0)
		assert.NoError(t, err)

		// Verify it moved
		card, err = project_model.GetProjectCard(t.Context(), project.ID, 1)
		assert.NoError(t, err)
		assert.Equal(t, column2.ID, card.ProjectColumnID)
	})

	t.Run("Move to column in wrong project fails", func(t *testing.T) {
		card, err := project_model.GetProjectCard(t.Context(), project.ID, 1)
		assert.NoError(t, err)

		// Create a column in a different project
		otherProject, err := CreateProject(t.Context(), 0, 1, project_model.TypeRepository, user, CreateProjectOptions{
			Title: "Other Project",
		})
		assert.NoError(t, err)
		otherColumn, err := CreateColumn(t.Context(), otherProject, user, CreateColumnOptions{Title: "Other"})
		assert.NoError(t, err)

		err = MoveCard(t.Context(), user, project, card, otherColumn, 0)
		assert.Error(t, err)
	})
}

func TestAddCardAndMoveRoundTrip(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	// Create a fresh project with columns
	p, err := CreateProject(t.Context(), 0, 1, project_model.TypeRepository, user, CreateProjectOptions{
		Title:        "Round Trip Test",
		TemplateType: project_model.TemplateTypeBasicKanban,
	})
	assert.NoError(t, err)

	columns, err := p.GetColumns(t.Context())
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(columns), 2)

	col1 := columns[0]
	col2 := columns[1]

	// Add issue 11 (repo 1) to first column
	card, err := AddCardToColumn(t.Context(), p, col1, 11, -1)
	assert.NoError(t, err)
	assert.Equal(t, col1.ID, card.ProjectColumnID)

	// Move to second column
	err = MoveCard(t.Context(), user, p, card, col2, 0)
	assert.NoError(t, err)

	// Verify final position
	card, err = project_model.GetProjectCard(t.Context(), p.ID, 11)
	assert.NoError(t, err)
	assert.Equal(t, col2.ID, card.ProjectColumnID)

	// Remove
	err = RemoveCardFromProject(t.Context(), p, 11)
	assert.NoError(t, err)

	_, err = project_model.GetProjectCard(t.Context(), p.ID, 11)
	assert.True(t, project_model.IsErrProjectCardNotExist(err))
}
