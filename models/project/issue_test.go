// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package project

import (
	"testing"

	"code.gitea.io/gitea/models/unittest"

	"github.com/stretchr/testify/assert"
)

func TestAddIssueToProject(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	// Add issue 4 to project 1, column 1, auto-append sorting
	err := AddIssueToProject(t.Context(), 1, 1, 4, -1)
	assert.NoError(t, err)

	// Verify it was added
	card, err := GetProjectCard(t.Context(), 1, 4)
	assert.NoError(t, err)
	assert.Equal(t, int64(4), card.IssueID)
	assert.Equal(t, int64(1), card.ProjectID)
	assert.Equal(t, int64(1), card.ProjectColumnID)
	// Column 1 already has issue 1 at sorting 0, so new card should be at sorting 1
	assert.Equal(t, int64(1), card.Sorting)

	// Try adding same issue again — expect ErrCardAlreadyInProject
	err = AddIssueToProject(t.Context(), 1, 1, 4, -1)
	assert.Error(t, err)
	assert.True(t, IsErrCardAlreadyInProject(err))
}

func TestAddIssueToProjectWithExplicitSorting(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	// Add issue with explicit sorting=0 to an empty column (column 4, project 4)
	err := AddIssueToProject(t.Context(), 4, 4, 10, 0)
	assert.NoError(t, err)

	card, err := GetProjectCard(t.Context(), 4, 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), card.Sorting)

	// Add another issue with auto-append (-1) — should get sorting=1
	err = AddIssueToProject(t.Context(), 4, 4, 11, -1)
	assert.NoError(t, err)

	card2, err := GetProjectCard(t.Context(), 4, 11)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), card2.Sorting)
}

func TestRemoveIssueFromProject(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	// Verify issue 1 is in project 1
	_, err := GetProjectCard(t.Context(), 1, 1)
	assert.NoError(t, err)

	// Remove issue 1 from project 1
	err = RemoveIssueFromProject(t.Context(), 1, 1)
	assert.NoError(t, err)

	// Verify it's gone
	_, err = GetProjectCard(t.Context(), 1, 1)
	assert.Error(t, err)
	assert.True(t, IsErrProjectCardNotExist(err))

	// Remove non-existent issue — no error (silent)
	err = RemoveIssueFromProject(t.Context(), 1, 999)
	assert.NoError(t, err)
}

func TestGetProjectCard(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	// Get card for project 1, issue 1 — should succeed
	card, err := GetProjectCard(t.Context(), 1, 1)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), card.IssueID)
	assert.Equal(t, int64(1), card.ProjectID)
	assert.Equal(t, int64(1), card.ProjectColumnID)

	// Get card for project 1, issue 999 — should return ErrProjectCardNotExist
	_, err = GetProjectCard(t.Context(), 1, 999)
	assert.Error(t, err)
	assert.True(t, IsErrProjectCardNotExist(err))
}

func TestGetProjectIssueByID(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	// Get card by ID 1 — should succeed, verify fields
	card, err := GetProjectIssueByID(t.Context(), 1)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), card.ID)
	assert.Equal(t, int64(1), card.IssueID)
	assert.Equal(t, int64(1), card.ProjectID)
	assert.Equal(t, int64(1), card.ProjectColumnID)

	// Get card by ID 999 — should return ErrProjectCardNotExist
	_, err = GetProjectIssueByID(t.Context(), 999)
	assert.Error(t, err)
	assert.True(t, IsErrProjectCardNotExist(err))
}

func TestCountCardsInColumn(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	// Count cards in column 1 of project 1 — should be 1 (issue 1)
	count, err := CountCardsInColumn(t.Context(), 1)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// Count cards in column 2 — should be 1 (issue 3)
	count, err = CountCardsInColumn(t.Context(), 2)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// Count cards in column 3 — should be 1 (issue 5)
	count, err = CountCardsInColumn(t.Context(), 3)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// Count cards in nonexistent column — should be 0
	count, err = CountCardsInColumn(t.Context(), 999)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestGetProjectIssueColumnIDs(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	// Get column IDs for project 1
	columnIDs, err := GetProjectIssueColumnIDs(t.Context(), 1)
	assert.NoError(t, err)
	assert.Len(t, columnIDs, 4)
	assert.Equal(t, int64(1), columnIDs[1]) // issue 1 -> column 1
	assert.Equal(t, int64(0), columnIDs[2]) // issue 2 -> column 0 (unassigned)
	assert.Equal(t, int64(2), columnIDs[3]) // issue 3 -> column 2
	assert.Equal(t, int64(3), columnIDs[5]) // issue 5 -> column 3

	// Get for nonexistent project — empty map
	columnIDs, err = GetProjectIssueColumnIDs(t.Context(), 999)
	assert.NoError(t, err)
	assert.Empty(t, columnIDs)
}

func TestAddAndRemoveIssueFromProject(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	// Add issue 6 to project 1, column 2
	err := AddIssueToProject(t.Context(), 1, 2, 6, -1)
	assert.NoError(t, err)

	// Verify via column IDs
	columnIDs, err := GetProjectIssueColumnIDs(t.Context(), 1)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), columnIDs[6])

	// Count should now be 2 in column 2
	count, err := CountCardsInColumn(t.Context(), 2)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), count)

	// Remove it
	err = RemoveIssueFromProject(t.Context(), 1, 6)
	assert.NoError(t, err)

	// Count back to 1
	count, err = CountCardsInColumn(t.Context(), 2)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)
}
