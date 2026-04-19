// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package project

import (
	"strings"
	"testing"

	"code.gitea.io/gitea/models/db"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	"code.gitea.io/gitea/modules/optional"
	"code.gitea.io/gitea/modules/timeutil"

	"github.com/stretchr/testify/assert"
)

func TestIsProjectTypeValid(t *testing.T) {
	const UnknownType Type = 15

	cases := []struct {
		typ   Type
		valid bool
	}{
		{TypeIndividual, true},
		{TypeRepository, true},
		{TypeOrganization, true},
		{UnknownType, false},
	}

	for _, v := range cases {
		assert.Equal(t, v.valid, IsTypeValid(v.typ))
	}
}

func TestGetProjects(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	projects, err := db.Find[Project](t.Context(), SearchOptions{RepoID: 1})
	assert.NoError(t, err)

	// 1 value for this repo exists in the fixtures
	assert.Len(t, projects, 1)

	projects, err = db.Find[Project](t.Context(), SearchOptions{RepoID: 3})
	assert.NoError(t, err)

	// 1 value for this repo exists in the fixtures
	assert.Len(t, projects, 1)
}

func TestProject(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	project := &Project{
		Type:         TypeRepository,
		TemplateType: TemplateTypeBasicKanban,
		CardType:     CardTypeTextOnly,
		Title:        "New Project",
		RepoID:       1,
		CreatedUnix:  timeutil.TimeStampNow(),
		CreatorID:    2,
	}

	assert.NoError(t, NewProject(t.Context(), project))

	_, err := GetProjectByID(t.Context(), project.ID)
	assert.NoError(t, err)

	// Update project
	project.Title = "Updated title"
	assert.NoError(t, UpdateProject(t.Context(), project))

	projectFromDB, err := GetProjectByID(t.Context(), project.ID)
	assert.NoError(t, err)

	assert.Equal(t, project.Title, projectFromDB.Title)

	assert.NoError(t, ChangeProjectStatus(t.Context(), project, true))

	// Retrieve from DB afresh to check if it is truly closed
	projectFromDB, err = GetProjectByID(t.Context(), project.ID)
	assert.NoError(t, err)

	assert.True(t, projectFromDB.IsClosed)
}

func TestChangeProjectStatusClosedDate(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	// Create a project
	p := &Project{
		Title:     "Close Date Test",
		RepoID:    1,
		Type:      TypeRepository,
		CreatorID: 1,
	}
	assert.NoError(t, NewProject(t.Context(), p))
	assert.Equal(t, timeutil.TimeStamp(0), p.ClosedDateUnix)

	// Close it
	assert.NoError(t, ChangeProjectStatus(t.Context(), p, true))
	got, err := GetProjectByID(t.Context(), p.ID)
	assert.NoError(t, err)
	assert.True(t, got.IsClosed)
	assert.NotEqual(t, timeutil.TimeStamp(0), got.ClosedDateUnix)

	// Reopen it
	assert.NoError(t, ChangeProjectStatus(t.Context(), got, false))
	got, err = GetProjectByID(t.Context(), p.ID)
	assert.NoError(t, err)
	assert.False(t, got.IsClosed)
	assert.Equal(t, timeutil.TimeStamp(0), got.ClosedDateUnix)
}

func TestProjectsSort(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	tests := []struct {
		sortType string
		wants    []int64
	}{
		{
			sortType: "default",
			wants:    []int64{1, 3, 2, 6, 5, 4},
		},
		{
			sortType: "oldest",
			wants:    []int64{4, 5, 6, 2, 3, 1},
		},
		{
			sortType: "recentupdate",
			wants:    []int64{1, 3, 2, 6, 5, 4},
		},
		{
			sortType: "leastupdate",
			wants:    []int64{4, 5, 6, 2, 3, 1},
		},
	}

	for _, tt := range tests {
		projects, count, err := db.FindAndCount[Project](t.Context(), SearchOptions{
			OrderBy: GetSearchOrderByBySortType(tt.sortType),
		})
		assert.NoError(t, err)
		assert.Equal(t, int64(6), count)
		if assert.Len(t, projects, 6) {
			for i := range projects {
				assert.Equal(t, tt.wants[i], projects[i].ID)
			}
		}
	}
}

func TestIsValidSortType(t *testing.T) {
	// Valid sort types
	validTypes := []string{"oldest", "recentupdate", "leastupdate", "alphabetically", "reversealphabetically", "newest"}
	for _, st := range validTypes {
		assert.True(t, IsValidSortType(st), "expected %q to be valid", st)
	}

	// Invalid sort types
	invalidTypes := []string{"invalid", "", "random", "default"}
	for _, st := range invalidTypes {
		assert.False(t, IsValidSortType(st), "expected %q to be invalid", st)
	}
}

func TestGetProjectsByIDs(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	// Get projects [1, 2, 4] — verify all returned
	projects, err := GetProjectsByIDs(t.Context(), []int64{1, 2, 4})
	assert.NoError(t, err)
	assert.Len(t, projects, 3)
	assert.NotNil(t, projects[1])
	assert.NotNil(t, projects[2])
	assert.NotNil(t, projects[4])
	assert.Equal(t, "First project", projects[1].Title)
	assert.Equal(t, "second project", projects[2].Title)

	// Get with nonexistent ID [1, 999] — partial result (only 1)
	projects, err = GetProjectsByIDs(t.Context(), []int64{1, 999})
	assert.NoError(t, err)
	assert.Len(t, projects, 1)
	assert.NotNil(t, projects[1])

	// Empty list — empty map
	projects, err = GetProjectsByIDs(t.Context(), []int64{})
	assert.NoError(t, err)
	assert.Empty(t, projects)
}

func TestGetProjectForOrgByID(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	// No org projects in fixtures (all type 2 = TypeRepository), so create one
	orgProject := &Project{
		Type:      TypeOrganization,
		Title:     "Org Project",
		OwnerID:   3, // org in fixtures
		CreatorID: 2,
	}
	assert.NoError(t, NewProject(t.Context(), orgProject))

	// Should be found by org owner
	found, err := GetProjectForOrgByID(t.Context(), 3, orgProject.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Org Project", found.Title)

	// Wrong owner — not found
	_, err = GetProjectForOrgByID(t.Context(), 1, orgProject.ID)
	assert.Error(t, err)
	assert.True(t, IsErrProjectNotExist(err))

	// Nonexistent project — not found
	_, err = GetProjectForOrgByID(t.Context(), 3, 999)
	assert.Error(t, err)
	assert.True(t, IsErrProjectNotExist(err))
}

func TestDeleteProjectByID(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	// Verify project 1 exists with columns and cards
	project, err := GetProjectByID(t.Context(), 1)
	assert.NoError(t, err)
	assert.NotNil(t, project)

	columns, err := project.GetColumns(t.Context())
	assert.NoError(t, err)
	assert.NotEmpty(t, columns)

	columnIDs, err := GetProjectIssueColumnIDs(t.Context(), 1)
	assert.NoError(t, err)
	assert.NotEmpty(t, columnIDs)

	// Delete project
	assert.NoError(t, DeleteProjectByID(t.Context(), 1))

	// Project should be gone
	_, err = GetProjectByID(t.Context(), 1)
	assert.Error(t, err)
	assert.True(t, IsErrProjectNotExist(err))

	// Columns should be gone
	for _, col := range columns {
		_, err := GetColumn(t.Context(), col.ID)
		assert.Error(t, err)
		assert.True(t, IsErrProjectColumnNotExist(err))
	}

	// Cards should be gone
	columnIDs, err = GetProjectIssueColumnIDs(t.Context(), 1)
	assert.NoError(t, err)
	assert.Empty(t, columnIDs)

	// Deleting again should be a no-op
	assert.NoError(t, DeleteProjectByID(t.Context(), 1))
}

func TestCanBeAccessedByOwnerRepo(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	// Project 1: repo project (repo_id=1)
	repoProject := unittest.AssertExistsAndLoadBean(t, &Project{ID: 1})

	repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	repo2 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2})

	// Accessible by repo 1
	assert.True(t, repoProject.CanBeAccessedByOwnerRepo(0, repo1))
	// Not accessible by repo 2
	assert.False(t, repoProject.CanBeAccessedByOwnerRepo(0, repo2))
	// Not accessible with nil repo
	assert.False(t, repoProject.CanBeAccessedByOwnerRepo(0, nil))

	// Project 4: user project (owner_id=2, repo_id=0)
	// Note: fixture has type=2 (TypeRepository) but owner_id=2 and repo_id=0
	// For a proper test, create an individual project
	userProject := &Project{
		Type:      TypeIndividual,
		Title:     "User Project",
		OwnerID:   2,
		CreatorID: 2,
	}
	assert.NoError(t, NewProject(t.Context(), userProject))

	// Accessible by owner 2
	assert.True(t, userProject.CanBeAccessedByOwnerRepo(2, nil))
	// Not accessible by owner 1
	assert.False(t, userProject.CanBeAccessedByOwnerRepo(1, nil))
}

func TestGetProjectForRepoByID(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	// Project 1 belongs to repo 1
	project, err := GetProjectForRepoByID(t.Context(), 1, 1)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), project.ID)

	// Wrong repo — not found
	_, err = GetProjectForRepoByID(t.Context(), 2, 1)
	assert.Error(t, err)
	assert.True(t, IsErrProjectNotExist(err))

	// Nonexistent project — not found
	_, err = GetProjectForRepoByID(t.Context(), 1, 999)
	assert.Error(t, err)
	assert.True(t, IsErrProjectNotExist(err))
}

func TestGetSearchOrderByBySortType(t *testing.T) {
	assert.Equal(t, db.SearchOrderByOldest, GetSearchOrderByBySortType("oldest"))
	assert.Equal(t, db.SearchOrderByRecentUpdated, GetSearchOrderByBySortType("recentupdate"))
	assert.Equal(t, db.SearchOrderByLeastUpdated, GetSearchOrderByBySortType("leastupdate"))
	assert.Equal(t, db.SearchOrderBy("title ASC"), GetSearchOrderByBySortType("alphabetically"))
	assert.Equal(t, db.SearchOrderBy("title DESC"), GetSearchOrderByBySortType("reversealphabetically"))
	assert.Equal(t, db.SearchOrderByNewest, GetSearchOrderByBySortType("default"))
	assert.Equal(t, db.SearchOrderByNewest, GetSearchOrderByBySortType(""))
}

func TestNewProjectTitleTruncation(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	// Create a project with a very long title
	longTitle := strings.Repeat("a", 300)

	p := &Project{
		Title:     longTitle,
		RepoID:    1,
		Type:      TypeRepository,
		CreatorID: 1,
	}
	assert.NoError(t, NewProject(t.Context(), p))

	got, err := GetProjectByID(t.Context(), p.ID)
	assert.NoError(t, err)
	assert.LessOrEqual(t, len(got.Title), 255)
}

func TestProjectClosedStatus(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	// Search for open projects only
	projects, err := db.Find[Project](t.Context(), SearchOptions{
		IsClosed: optional.Some(false),
	})
	assert.NoError(t, err)
	for _, p := range projects {
		assert.False(t, p.IsClosed)
	}

	// Search for closed projects only
	closedProjects, err := db.Find[Project](t.Context(), SearchOptions{
		IsClosed: optional.Some(true),
	})
	assert.NoError(t, err)
	for _, p := range closedProjects {
		assert.True(t, p.IsClosed)
	}
	// Project 3 is closed in fixtures
	assert.NotEmpty(t, closedProjects)
}

func TestProjectErrorTypes(t *testing.T) {
	// Test error type implementations
	err1 := ErrProjectNotExist{ID: 42, RepoID: 1}
	assert.Contains(t, err1.Error(), "42")
	assert.True(t, IsErrProjectNotExist(err1))
	assert.False(t, IsErrProjectNotExist(assert.AnError))

	err2 := ErrProjectColumnNotExist{ColumnID: 7}
	assert.Contains(t, err2.Error(), "7")
	assert.True(t, IsErrProjectColumnNotExist(err2))
	assert.False(t, IsErrProjectColumnNotExist(assert.AnError))

	err3 := ErrProjectCardNotExist{CardID: 3, ProjectID: 1, IssueID: 5}
	assert.Contains(t, err3.Error(), "3")
	assert.True(t, IsErrProjectCardNotExist(err3))
	assert.False(t, IsErrProjectCardNotExist(assert.AnError))

	err4 := ErrCardAlreadyInProject{ProjectID: 1, IssueID: 2}
	assert.Contains(t, err4.Error(), "1")
	assert.True(t, IsErrCardAlreadyInProject(err4))
	assert.False(t, IsErrCardAlreadyInProject(assert.AnError))
}
