// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package v1_26

import (
	"xorm.io/xorm"
	"xorm.io/xorm/schemas"
)

// projectBoardMigration330 mirrors project_board table for migration
type projectBoardMigration330 struct {
	ID        int64 `xorm:"pk autoincr"`
	ProjectID int64 `xorm:"INDEX NOT NULL"`
	Sorting   int8  `xorm:"NOT NULL DEFAULT 0"`
}

func (projectBoardMigration330) TableName() string {
	return "project_board"
}

func (*projectBoardMigration330) TableIndices() []*schemas.Index {
	idx := schemas.NewIndex("UQE_project_board_project_sorting", schemas.UniqueType)
	idx.AddColumn("project_id", "sorting")
	return []*schemas.Index{idx}
}

type projectIssueMigration330 struct {
	ID              int64 `xorm:"pk autoincr"`
	IssueID         int64 `xorm:"INDEX NOT NULL"`
	ProjectID       int64 `xorm:"INDEX NOT NULL"`
	ProjectColumnID int64 `xorm:"'project_board_id' INDEX NOT NULL DEFAULT 0"`
	Sorting         int64 `xorm:"NOT NULL DEFAULT 0"`
}

func (projectIssueMigration330) TableName() string {
	return "project_issue"
}

func (*projectIssueMigration330) TableIndices() []*schemas.Index {
	indices := make([]*schemas.Index, 0, 2)

	piUnique := schemas.NewIndex("UQE_project_issue_project_issue", schemas.UniqueType)
	piUnique.AddColumn("project_id", "issue_id")
	indices = append(indices, piUnique)

	csUnique := schemas.NewIndex("UQE_project_issue_column_sorting", schemas.UniqueType)
	csUnique.AddColumn("project_board_id", "sorting")
	indices = append(indices, csUnique)

	return indices
}

func FixProjectSortingAndAddUniqueConstraints(x *xorm.Engine) error {
	// Step 1: Fix duplicate sorting in project_board per project
	type boardDup struct {
		ProjectID int64
		Sorting   int8
		Cnt       int
	}
	var boardDups []boardDup
	if err := x.SQL("SELECT project_id, sorting, COUNT(*) as cnt FROM project_board GROUP BY project_id, sorting HAVING COUNT(*) > 1").Find(&boardDups); err != nil {
		return err
	}
	for _, d := range boardDups {
		type idRow struct {
			ID int64
		}
		var ids []idRow
		if err := x.SQL("SELECT id FROM project_board WHERE project_id = ? AND sorting = ? ORDER BY id", d.ProjectID, d.Sorting).Find(&ids); err != nil {
			return err
		}
		type maxResult struct {
			MaxSorting int8
		}
		var maxRes maxResult
		if _, err := x.SQL("SELECT COALESCE(MAX(sorting), 0) as max_sorting FROM project_board WHERE project_id = ?", d.ProjectID).Get(&maxRes); err != nil {
			return err
		}
		nextSorting := maxRes.MaxSorting
		// Skip first (keep its sorting), reassign rest
		for i := 1; i < len(ids); i++ {
			nextSorting++
			if _, err := x.Exec("UPDATE project_board SET sorting = ? WHERE id = ?", nextSorting, ids[i].ID); err != nil {
				return err
			}
		}
	}

	// Step 2: Fix duplicate sorting in project_issue per column
	type issueDup struct {
		ProjectBoardID int64 `xorm:"project_board_id"`
		Sorting        int64
		Cnt            int
	}
	var issueDups []issueDup
	if err := x.SQL("SELECT project_board_id, sorting, COUNT(*) as cnt FROM project_issue GROUP BY project_board_id, sorting HAVING COUNT(*) > 1").Find(&issueDups); err != nil {
		return err
	}
	for _, d := range issueDups {
		type idRow struct {
			ID int64
		}
		var ids []idRow
		if err := x.SQL("SELECT id FROM project_issue WHERE project_board_id = ? AND sorting = ? ORDER BY id", d.ProjectBoardID, d.Sorting).Find(&ids); err != nil {
			return err
		}
		type maxResult struct {
			MaxSorting int64
		}
		var maxRes maxResult
		if _, err := x.SQL("SELECT COALESCE(MAX(sorting), 0) as max_sorting FROM project_issue WHERE project_board_id = ?", d.ProjectBoardID).Get(&maxRes); err != nil {
			return err
		}
		nextSorting := maxRes.MaxSorting
		for i := 1; i < len(ids); i++ {
			nextSorting++
			if _, err := x.Exec("UPDATE project_issue SET sorting = ? WHERE id = ?", nextSorting, ids[i].ID); err != nil {
				return err
			}
		}
	}

	// Step 3: Remove duplicate project_issue rows (same project_id + issue_id)
	type projIssueDup struct {
		ProjectID int64
		IssueID   int64
		Cnt       int
	}
	var piDups []projIssueDup
	if err := x.SQL("SELECT project_id, issue_id, COUNT(*) as cnt FROM project_issue GROUP BY project_id, issue_id HAVING COUNT(*) > 1").Find(&piDups); err != nil {
		return err
	}
	for _, d := range piDups {
		type idRow struct {
			ID int64
		}
		var ids []idRow
		if err := x.SQL("SELECT id FROM project_issue WHERE project_id = ? AND issue_id = ? ORDER BY id", d.ProjectID, d.IssueID).Find(&ids); err != nil {
			return err
		}
		// Keep the first, delete the rest
		for i := 1; i < len(ids); i++ {
			if _, err := x.Exec("DELETE FROM project_issue WHERE id = ?", ids[i].ID); err != nil {
				return err
			}
		}
	}

	// Step 4: Sync structs to add unique constraints
	if err := x.Sync(new(projectBoardMigration330)); err != nil {
		return err
	}
	return x.Sync(new(projectIssueMigration330))
}
