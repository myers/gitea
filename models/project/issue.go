// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package project

import (
	"context"
	"errors"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/util"
)

// ProjectIssue saves relation from issue to a project
type ProjectIssue struct { //revive:disable-line:exported
	ID        int64 `xorm:"pk autoincr"`
	IssueID   int64 `xorm:"INDEX"`
	ProjectID int64 `xorm:"INDEX"`

	// ProjectColumnID should not be zero since 1.22. If it's zero, the issue will not be displayed on UI and it might result in errors.
	ProjectColumnID int64 `xorm:"'project_board_id' INDEX"`

	// the sorting order on the column
	Sorting int64 `xorm:"NOT NULL DEFAULT 0"`
}

func init() {
	db.RegisterModel(new(ProjectIssue))
}

func deleteProjectIssuesByProjectID(ctx context.Context, projectID int64) error {
	_, err := db.GetEngine(ctx).Where("project_id=?", projectID).Delete(&ProjectIssue{})
	return err
}

// GetColumnIssueNextSorting returns the sorting value to append an issue at the end of the column.
func GetColumnIssueNextSorting(ctx context.Context, projectID, columnID int64) (int64, error) {
	res := struct {
		MaxSorting int64
		IssueCount int64
	}{}
	if _, err := db.GetEngine(ctx).Select("max(sorting) AS max_sorting, count(*) AS issue_count").
		Table("project_issue").
		Where("project_id=?", projectID).
		And("project_board_id=?", columnID).
		Get(&res); err != nil {
		return 0, err
	}
	return util.Iif(res.IssueCount > 0, res.MaxSorting+1, 0), nil
}

func moveIssuesToAnotherColumn(ctx context.Context, oldColumn, newColumn *Column) error {
	if oldColumn.ProjectID != newColumn.ProjectID {
		return errors.New("columns have to be in the same project")
	}

	if oldColumn.ID == newColumn.ID {
		return nil
	}

	movedIssues, err := oldColumn.GetIssues(ctx)
	if err != nil {
		return err
	}
	if len(movedIssues) == 0 {
		return nil
	}

	nextSorting, err := GetColumnIssueNextSorting(ctx, newColumn.ProjectID, newColumn.ID)
	if err != nil {
		return err
	}
	return db.WithTx(ctx, func(ctx context.Context) error {
		for i, issue := range movedIssues {
			issue.ProjectColumnID = newColumn.ID
			issue.Sorting = nextSorting + int64(i)
			if _, err := db.GetEngine(ctx).ID(issue.ID).Cols("project_board_id", "sorting").Update(issue); err != nil {
				return err
			}
		}
		return nil
	})
}

// DeleteAllProjectIssueByIssueIDsAndProjectIDs delete all project's issues by issue's and project's ids
func DeleteAllProjectIssueByIssueIDsAndProjectIDs(ctx context.Context, issueIDs, projectIDs []int64) error {
	_, err := db.GetEngine(ctx).In("project_id", projectIDs).In("issue_id", issueIDs).Delete(&ProjectIssue{})
	return err
}

// AddIssueToProject adds an issue to a project column. If sorting < 0, appends to end.
func AddIssueToProject(ctx context.Context, projectID, columnID, issueID, sorting int64) error {
	return db.WithTx(ctx, func(ctx context.Context) error {
		has, err := db.GetEngine(ctx).Where("project_id = ? AND issue_id = ?", projectID, issueID).Exist(&ProjectIssue{})
		if err != nil {
			return err
		}
		if has {
			return ErrCardAlreadyInProject{ProjectID: projectID, IssueID: issueID}
		}

		if sorting < 0 {
			var maxRes struct {
				MaxSorting int64
				Cnt        int64
			}
			if _, err := db.GetEngine(ctx).Table("project_issue").
				Select("COALESCE(MAX(sorting), -1) as max_sorting, COUNT(*) as cnt").
				Where("project_board_id = ?", columnID).
				Get(&maxRes); err != nil {
				return err
			}
			sorting = util.Iif(maxRes.Cnt > 0, maxRes.MaxSorting+1, 0)
		}

		return db.Insert(ctx, &ProjectIssue{
			IssueID:         issueID,
			ProjectID:       projectID,
			ProjectColumnID: columnID,
			Sorting:         sorting,
		})
	})
}

// RemoveIssueFromProject removes an issue from a project. No error if not found.
func RemoveIssueFromProject(ctx context.Context, projectID, issueID int64) error {
	_, err := db.GetEngine(ctx).Where("project_id = ? AND issue_id = ?", projectID, issueID).Delete(&ProjectIssue{})
	return err
}

// GetProjectCard returns the project card for a given project and issue
func GetProjectCard(ctx context.Context, projectID, issueID int64) (*ProjectIssue, error) {
	pi := new(ProjectIssue)
	has, err := db.GetEngine(ctx).Where("project_id = ? AND issue_id = ?", projectID, issueID).Get(pi)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrProjectCardNotExist{ProjectID: projectID, IssueID: issueID}
	}
	return pi, nil
}

// GetProjectIssueByID returns a project card by its ID
func GetProjectIssueByID(ctx context.Context, cardID int64) (*ProjectIssue, error) {
	pi := new(ProjectIssue)
	has, err := db.GetEngine(ctx).ID(cardID).Get(pi)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrProjectCardNotExist{CardID: cardID}
	}
	return pi, nil
}

// CountCardsInColumn returns the number of cards in a column
func CountCardsInColumn(ctx context.Context, columnID int64) (int64, error) {
	return db.GetEngine(ctx).Where("project_board_id = ?", columnID).Count(&ProjectIssue{})
}

// GetColumnCards returns paginated cards in a column with total count
func GetColumnCards(ctx context.Context, columnID int64, opts db.ListOptions) ([]*ProjectIssue, int64, error) {
	count, err := db.GetEngine(ctx).Where("project_board_id = ?", columnID).Count(new(ProjectIssue))
	if err != nil {
		return nil, 0, err
	}

	sess := db.GetEngine(ctx).Where("project_board_id = ?", columnID).OrderBy("sorting, id")
	if opts.PageSize > 0 {
		sess = db.SetSessionPagination(sess, &opts)
	}

	cards := make([]*ProjectIssue, 0, opts.PageSize)
	if err := sess.Find(&cards); err != nil {
		return nil, 0, err
	}
	return cards, count, nil
}

// GetProjectIssueColumnIDs returns a map of issue IDs to column IDs for a project
func GetProjectIssueColumnIDs(ctx context.Context, projectID int64) (map[int64]int64, error) {
	issues := make([]ProjectIssue, 0)
	if err := db.GetEngine(ctx).Where("project_id = ?", projectID).Find(&issues); err != nil {
		return nil, err
	}
	result := make(map[int64]int64, len(issues))
	for _, pi := range issues {
		result[pi.IssueID] = pi.ProjectColumnID
	}
	return result, nil
}
