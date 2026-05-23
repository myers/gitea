// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"testing"

	project_model "code.gitea.io/gitea/models/project"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newProjectsMetaForTest(open, closed []*project_model.Project) *IssuePageMetaData {
	return &IssuePageMetaData{
		ProjectsData: &issueSidebarProjectsData{
			OpenProjects:   open,
			ClosedProjects: closed,
		},
	}
}

func TestSetSelectedProjectTitles_HappyPath(t *testing.T) {
	d := newProjectsMetaForTest(
		[]*project_model.Project{{ID: 1, Title: "Triage"}},
		nil,
	)
	d.SetSelectedProjectTitles([]string{"Triage"})
	assert.Equal(t, []int64{1}, d.ProjectsData.SelectedProjectIDs)
}

func TestSetSelectedProjectTitles_RepoLevelWinsOnShadowing(t *testing.T) {
	// retrieveProjectsInternal appends owner-level after repo-level, so the
	// OpenProjects slice in production looks like [repoTriage, ownerTriage].
	d := newProjectsMetaForTest(
		[]*project_model.Project{
			{ID: 10, Title: "Triage"}, // repo-level (loaded first)
			{ID: 20, Title: "Triage"}, // owner-level (loaded second)
		},
		nil,
	)
	d.SetSelectedProjectTitles([]string{"Triage"})
	assert.Equal(t, []int64{10}, d.ProjectsData.SelectedProjectIDs)
}

func TestSetSelectedProjectTitles_UnknownTitleDropsSilently(t *testing.T) {
	d := newProjectsMetaForTest(
		[]*project_model.Project{{ID: 1, Title: "Triage"}},
		nil,
	)
	d.SetSelectedProjectTitles([]string{"Nonexistent"})
	assert.Empty(t, d.ProjectsData.SelectedProjectIDs)
}

func TestSetSelectedProjectTitles_CaseInsensitive(t *testing.T) {
	d := newProjectsMetaForTest(
		[]*project_model.Project{{ID: 1, Title: "Triage"}},
		nil,
	)
	d.SetSelectedProjectTitles([]string{"TRIAGE"})
	assert.Equal(t, []int64{1}, d.ProjectsData.SelectedProjectIDs)
}

func TestSetSelectedProjectTitles_ClosedProjectsResolvable(t *testing.T) {
	d := newProjectsMetaForTest(
		nil,
		[]*project_model.Project{{ID: 42, Title: "Old Roadmap"}},
	)
	d.SetSelectedProjectTitles([]string{"Old Roadmap"})
	assert.Equal(t, []int64{42}, d.ProjectsData.SelectedProjectIDs)
}

func TestSetSelectedProjectTitles_UnionsWithExistingQuerySelection(t *testing.T) {
	d := newProjectsMetaForTest(
		[]*project_model.Project{
			{ID: 1, Title: "Triage"},
			{ID: 2, Title: "Other"},
		},
		nil,
	)
	// Mimic what SetSelectedProjectIDs(parseProjectIDsFromQuery(ctx)) does
	// before setTemplateIfExists runs.
	d.SetSelectedProjectIDs([]int64{2})
	d.SetSelectedProjectTitles([]string{"Triage"})
	assert.Equal(t, []int64{2, 1}, d.ProjectsData.SelectedProjectIDs)
	require.Len(t, d.ProjectsData.ProjectCards, 2)
	assert.Equal(t, int64(2), d.ProjectsData.ProjectCards[0].Project.ID)
	assert.Equal(t, int64(1), d.ProjectsData.ProjectCards[1].Project.ID)
}

func TestSetSelectedProjectTitles_DuplicatesDeduped(t *testing.T) {
	d := newProjectsMetaForTest(
		[]*project_model.Project{
			{ID: 1, Title: "Triage"},
			{ID: 2, Title: "Other"},
		},
		nil,
	)
	d.SetSelectedProjectIDs([]int64{1}) // ?project=1 already picked Triage
	d.SetSelectedProjectTitles([]string{"Triage", "Other", "Triage"})
	assert.Equal(t, []int64{1, 2}, d.ProjectsData.SelectedProjectIDs)
	require.Len(t, d.ProjectsData.ProjectCards, 2)
	assert.Equal(t, int64(1), d.ProjectsData.ProjectCards[0].Project.ID)
	assert.Equal(t, int64(2), d.ProjectsData.ProjectCards[1].Project.ID)
}

func TestSetSelectedProjectTitles_NilTitlesIsNoOp(t *testing.T) {
	d := newProjectsMetaForTest(
		[]*project_model.Project{{ID: 1, Title: "Triage"}},
		nil,
	)
	d.SetSelectedProjectIDs([]int64{1})
	d.SetSelectedProjectTitles(nil)
	assert.Equal(t, []int64{1}, d.ProjectsData.SelectedProjectIDs)
}
