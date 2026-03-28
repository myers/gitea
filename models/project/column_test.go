// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package project

import (
	"fmt"
	"testing"

	"code.gitea.io/gitea/models/unittest"

	"github.com/stretchr/testify/assert"
)

func TestGetDefaultColumn(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	projectWithoutDefault, err := GetProjectByID(t.Context(), 5)
	assert.NoError(t, err)

	// check if default column was added
	column, err := projectWithoutDefault.MustDefaultColumn(t.Context())
	assert.NoError(t, err)
	assert.Equal(t, int64(5), column.ProjectID)
	assert.Equal(t, "Done", column.Title)

	projectWithMultipleDefaults, err := GetProjectByID(t.Context(), 6)
	assert.NoError(t, err)

	// check if multiple defaults were removed
	column, err = projectWithMultipleDefaults.MustDefaultColumn(t.Context())
	assert.NoError(t, err)
	assert.Equal(t, int64(6), column.ProjectID)
	assert.Equal(t, int64(9), column.ID) // there are 2 default columns in the test data, use the latest one

	// set 8 as default column
	assert.NoError(t, SetDefaultColumn(t.Context(), column.ProjectID, 8))

	// then 9 will become a non-default column
	column, err = GetColumn(t.Context(), 9)
	assert.NoError(t, err)
	assert.Equal(t, int64(6), column.ProjectID)
	assert.False(t, column.Default)
}

func Test_moveIssuesToAnotherColumn(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	column1 := unittest.AssertExistsAndLoadBean(t, &Column{ID: 1, ProjectID: 1})

	issues, err := column1.GetIssues(t.Context())
	assert.NoError(t, err)
	assert.Len(t, issues, 1)
	assert.EqualValues(t, 1, issues[0].ID)

	column2 := unittest.AssertExistsAndLoadBean(t, &Column{ID: 2, ProjectID: 1})
	issues, err = column2.GetIssues(t.Context())
	assert.NoError(t, err)
	assert.Len(t, issues, 1)
	assert.EqualValues(t, 3, issues[0].ID)

	err = column1.moveIssuesToAnotherColumn(t.Context(), column2)
	assert.NoError(t, err)

	issues, err = column1.GetIssues(t.Context())
	assert.NoError(t, err)
	assert.Empty(t, issues)

	issues, err = column2.GetIssues(t.Context())
	assert.NoError(t, err)
	assert.Len(t, issues, 2)
	assert.EqualValues(t, 3, issues[0].ID)
	assert.EqualValues(t, 0, issues[0].Sorting)
	assert.EqualValues(t, 1, issues[1].ID)
	assert.EqualValues(t, 1, issues[1].Sorting)
}

func Test_MoveColumnsOnProject(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	project1 := unittest.AssertExistsAndLoadBean(t, &Project{ID: 1})
	columns, err := project1.GetColumns(t.Context())
	assert.NoError(t, err)
	assert.Len(t, columns, 3)
	assert.EqualValues(t, 0, columns[0].Sorting)
	assert.EqualValues(t, 1, columns[1].Sorting)
	assert.EqualValues(t, 2, columns[2].Sorting)

	err = MoveColumnsOnProject(t.Context(), project1, map[int64]int64{
		0: columns[1].ID,
		1: columns[2].ID,
		2: columns[0].ID,
	})
	assert.NoError(t, err)

	columnsAfter, err := project1.GetColumns(t.Context())
	assert.NoError(t, err)
	assert.Len(t, columnsAfter, 3)
	assert.Equal(t, columns[1].ID, columnsAfter[0].ID)
	assert.Equal(t, columns[2].ID, columnsAfter[1].ID)
	assert.Equal(t, columns[0].ID, columnsAfter[2].ID)
}

func Test_NewColumn(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	project1 := unittest.AssertExistsAndLoadBean(t, &Project{ID: 1})
	columns, err := project1.GetColumns(t.Context())
	assert.NoError(t, err)
	assert.Len(t, columns, 3)

	for i := range maxProjectColumns - 3 {
		err := NewColumn(t.Context(), &Column{
			Title:     fmt.Sprintf("column-%d", i+4),
			ProjectID: project1.ID,
		})
		assert.NoError(t, err)
	}
	err = NewColumn(t.Context(), &Column{
		Title:     "column-21",
		ProjectID: project1.ID,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "maximum number of columns reached")
}

func TestBatchCountCardsInColumns(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	// Count cards in columns [1, 2, 3] of project 1
	counts, err := BatchCountCardsInColumns(t.Context(), []int64{1, 2, 3})
	assert.NoError(t, err)
	assert.Equal(t, int64(1), counts[1]) // column 1: issue 1
	assert.Equal(t, int64(1), counts[2]) // column 2: issue 3
	assert.Equal(t, int64(1), counts[3]) // column 3: issue 5

	// Empty list returns empty map
	counts, err = BatchCountCardsInColumns(t.Context(), []int64{})
	assert.NoError(t, err)
	assert.Empty(t, counts)

	// Nonexistent columns return empty map (no entries)
	counts, err = BatchCountCardsInColumns(t.Context(), []int64{999})
	assert.NoError(t, err)
	assert.Empty(t, counts)
}

func TestBatchCountProjectColumns(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	// Count columns in projects [1, 2, 4]
	counts, err := BatchCountProjectColumns(t.Context(), []int64{1, 2, 4})
	assert.NoError(t, err)
	assert.Equal(t, int64(3), counts[1]) // project 1: columns 1, 2, 3
	assert.Equal(t, int64(1), counts[2]) // project 2: column 5
	assert.Equal(t, int64(2), counts[4]) // project 4: columns 4, 6

	// Empty list returns empty map
	counts, err = BatchCountProjectColumns(t.Context(), []int64{})
	assert.NoError(t, err)
	assert.Empty(t, counts)

	// Nonexistent project returns empty map
	counts, err = BatchCountProjectColumns(t.Context(), []int64{999})
	assert.NoError(t, err)
	assert.Empty(t, counts)
}

func TestUpdateColumnColor(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	column := unittest.AssertExistsAndLoadBean(t, &Column{ID: 1})

	// Update to valid color — success
	column.Color = "#ff0000"
	err := UpdateColumn(t.Context(), column)
	assert.NoError(t, err)

	updated, err := GetColumn(t.Context(), 1)
	assert.NoError(t, err)
	assert.Equal(t, "#ff0000", updated.Color)

	// Update to invalid color — error
	column.Color = "red"
	err = UpdateColumn(t.Context(), column)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bad color code")

	// Update to empty string — success (clear color)
	column.Color = ""
	err = UpdateColumn(t.Context(), column)
	assert.NoError(t, err)

	updated, err = GetColumn(t.Context(), 1)
	assert.NoError(t, err)
	assert.Empty(t, updated.Color)
}

func TestDeleteColumnByID(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	// Delete non-default column 2 (project 1) — cards move to default column 1
	err := DeleteColumnByID(t.Context(), 2)
	assert.NoError(t, err)

	// Column 2 should no longer exist
	_, err = GetColumn(t.Context(), 2)
	assert.Error(t, err)
	assert.True(t, IsErrProjectColumnNotExist(err))

	// Issue 3 was in column 2, should now be in default column 1
	card, err := GetProjectCard(t.Context(), 1, 3)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), card.ProjectColumnID)

	// Try delete default column 1 — error
	err = DeleteColumnByID(t.Context(), 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot delete default column")

	// Delete nonexistent column — no error
	err = DeleteColumnByID(t.Context(), 999)
	assert.NoError(t, err)
}

func TestCreateDefaultColumnsForProject(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	// Create project with BasicKanban template — verify columns with sequential sorting
	project := &Project{
		Type:         TypeRepository,
		TemplateType: TemplateTypeBasicKanban,
		CardType:     CardTypeTextOnly,
		Title:        "Kanban Test",
		RepoID:       1,
		CreatorID:    2,
	}
	assert.NoError(t, NewProject(t.Context(), project))

	columns, err := project.GetColumns(t.Context())
	assert.NoError(t, err)
	// BasicKanban creates "Backlog" (default) + the template columns
	assert.GreaterOrEqual(t, len(columns), 2)

	// Verify sequential sorting
	for i, col := range columns {
		assert.EqualValues(t, i, col.Sorting)
	}

	// First column should be default
	assert.True(t, columns[0].Default)
	assert.Equal(t, "Backlog", columns[0].Title)

	// Create project with None template — verify no columns
	projectNone := &Project{
		Type:         TypeRepository,
		TemplateType: TemplateTypeNone,
		CardType:     CardTypeTextOnly,
		Title:        "No Template Test",
		RepoID:       1,
		CreatorID:    2,
	}
	assert.NoError(t, NewProject(t.Context(), projectNone))

	columnsNone, err := projectNone.GetColumns(t.Context())
	assert.NoError(t, err)
	assert.Empty(t, columnsNone)
}

func TestNewColumnSortingAssignment(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	// Use project 2 which has 1 column (id 5, sorting 0)
	project2 := unittest.AssertExistsAndLoadBean(t, &Project{ID: 2})

	// Add new columns and verify sorting
	for i := 1; i <= 3; i++ {
		err := NewColumn(t.Context(), &Column{
			Title:     fmt.Sprintf("col-%d", i),
			ProjectID: project2.ID,
		})
		assert.NoError(t, err)
	}

	columns, err := project2.GetColumns(t.Context())
	assert.NoError(t, err)
	assert.Len(t, columns, 4) // 1 original + 3 new

	for i, col := range columns {
		assert.EqualValues(t, i, col.Sorting, "column %d (%s) should have sorting %d", col.ID, col.Title, i)
	}
}

func TestMoveColumnsOnProjectErrorPaths(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	project1 := unittest.AssertExistsAndLoadBean(t, &Project{ID: 1})

	// Pass subset of columns (only 2 of 3) — should succeed since MoveColumnsOnProject
	// only validates that the passed column IDs exist in the project
	columns, err := project1.GetColumns(t.Context())
	assert.NoError(t, err)
	assert.Len(t, columns, 3)

	// Pass column from wrong project — error (column not found in project)
	err = MoveColumnsOnProject(t.Context(), project1, map[int64]int64{
		0: columns[0].ID,
		1: columns[1].ID,
		2: 999, // nonexistent
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "some columns do not exist")
}
