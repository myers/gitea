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

func TestCreateColumn(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	p, err := CreateProject(t.Context(), 0, 1, project_model.TypeRepository, user, CreateProjectOptions{
		Title: "Column Test Project",
	})
	assert.NoError(t, err)

	t.Run("Basic creation", func(t *testing.T) {
		col, err := CreateColumn(t.Context(), p, user, CreateColumnOptions{
			Title: "New Column",
			Color: "#ff0000",
		})
		assert.NoError(t, err)
		assert.Equal(t, "New Column", col.Title)
		assert.Equal(t, "#ff0000", col.Color)
		assert.Equal(t, p.ID, col.ProjectID)
		assert.Equal(t, user.ID, col.CreatorID)
	})

	t.Run("Invalid color rejected", func(t *testing.T) {
		_, err := CreateColumn(t.Context(), p, user, CreateColumnOptions{
			Title: "Bad Color",
			Color: "not-a-color",
		})
		assert.Error(t, err)
	})

	t.Run("Sequential sorting", func(t *testing.T) {
		col1, err := CreateColumn(t.Context(), p, user, CreateColumnOptions{Title: "Col A"})
		assert.NoError(t, err)
		col2, err := CreateColumn(t.Context(), p, user, CreateColumnOptions{Title: "Col B"})
		assert.NoError(t, err)

		assert.Less(t, col1.Sorting, col2.Sorting)
	})
}

func TestUpdateColumn(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	p, err := CreateProject(t.Context(), 0, 1, project_model.TypeRepository, user, CreateProjectOptions{
		Title: "Update Column Test",
	})
	assert.NoError(t, err)

	col, err := CreateColumn(t.Context(), p, user, CreateColumnOptions{
		Title: "Original",
		Color: "#000000",
	})
	assert.NoError(t, err)

	t.Run("Update title", func(t *testing.T) {
		newTitle := "Updated"
		err := UpdateColumn(t.Context(), col, UpdateColumnOptions{Title: &newTitle})
		assert.NoError(t, err)

		got, err := project_model.GetColumn(t.Context(), col.ID)
		assert.NoError(t, err)
		assert.Equal(t, "Updated", got.Title)
	})

	t.Run("Update color", func(t *testing.T) {
		newColor := "#00ff00"
		err := UpdateColumn(t.Context(), col, UpdateColumnOptions{Color: &newColor})
		assert.NoError(t, err)

		got, err := project_model.GetColumn(t.Context(), col.ID)
		assert.NoError(t, err)
		assert.Equal(t, "#00ff00", got.Color)
	})

	t.Run("Clear color", func(t *testing.T) {
		empty := ""
		err := UpdateColumn(t.Context(), col, UpdateColumnOptions{Color: &empty})
		assert.NoError(t, err)

		got, err := project_model.GetColumn(t.Context(), col.ID)
		assert.NoError(t, err)
		assert.Empty(t, got.Color)
	})

	t.Run("Invalid color rejected", func(t *testing.T) {
		bad := "red"
		err := UpdateColumn(t.Context(), col, UpdateColumnOptions{Color: &bad})
		assert.Error(t, err)
	})
}

func TestDeleteColumn(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	p, err := CreateProject(t.Context(), 0, 1, project_model.TypeRepository, user, CreateProjectOptions{
		Title:        "Delete Column Test",
		TemplateType: project_model.TemplateTypeBasicKanban,
	})
	assert.NoError(t, err)

	columns, err := p.GetColumns(t.Context())
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(columns), 2)

	// Find a non-default column to delete
	var nonDefault *project_model.Column
	var defaultCol *project_model.Column
	for _, c := range columns {
		if c.Default {
			defaultCol = c
		} else {
			nonDefault = c
		}
	}
	assert.NotNil(t, nonDefault, "need a non-default column")
	assert.NotNil(t, defaultCol, "need a default column")

	t.Run("Delete non-default column", func(t *testing.T) {
		err := DeleteColumn(t.Context(), nonDefault)
		assert.NoError(t, err)

		// Verify it's gone
		_, err = project_model.GetColumn(t.Context(), nonDefault.ID)
		assert.True(t, project_model.IsErrProjectColumnNotExist(err))
	})

	t.Run("Delete default column fails", func(t *testing.T) {
		err := DeleteColumn(t.Context(), defaultCol)
		assert.Error(t, err)
	})
}
