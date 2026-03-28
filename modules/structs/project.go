// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package structs

import "time"

// Project represents a project
// swagger:model
type Project struct {
	ID           int64     `json:"id"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	TemplateType uint8     `json:"template_type"`
	CardType     uint8     `json:"card_type"`
	State        StateType `json:"state"`
	OpenIssues   int64     `json:"open_issues"`
	ClosedIssues int64     `json:"closed_issues"`
	// swagger:strfmt date-time
	Created time.Time `json:"created_at"`
	// swagger:strfmt date-time
	Updated *time.Time `json:"updated_at"`
	// swagger:strfmt date-time
	Closed *time.Time `json:"closed_at"`
	// ColumnID is the project column the issue is in (only set when embedded in issue responses)
	ColumnID int64 `json:"column_id,omitempty"`
	// Column is the project column name (only set when embedded in issue responses)
	Column string `json:"column,omitempty"`
}

// CreateProjectOption options for creating a project
type CreateProjectOption struct {
	// required: true
	Title        string `json:"title" binding:"Required;MaxSize(255)"`
	Description  string `json:"description"`
	TemplateType uint8  `json:"template_type"`
	CardType     uint8  `json:"card_type"`
}

// EditProjectOption options for editing a project
type EditProjectOption struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	CardType    *uint8  `json:"card_type"`
	// enum: open,closed
	State *string `json:"state"`
}

// ProjectColumn represents a project column (board)
type ProjectColumn struct {
	ID      int64  `json:"id"`
	Title   string `json:"title"`
	Color   string `json:"color"`
	Sorting int    `json:"sorting"`
	Default bool   `json:"default"`
	// swagger:strfmt date-time
	Created time.Time `json:"created_at"`
	// swagger:strfmt date-time
	Updated time.Time `json:"updated_at"`
}

// CreateProjectColumnOption options for creating a project column
type CreateProjectColumnOption struct {
	// required: true
	Title string `json:"title" binding:"Required;MaxSize(255)"`
	Color string `json:"color"`
}

// EditProjectColumnOption options for editing a project column
type EditProjectColumnOption struct {
	Title *string `json:"title"`
	Color *string `json:"color"`
}

// MoveProjectColumnOption options for moving a project column
type MoveProjectColumnOption struct {
	Sorting int64 `json:"sorting"`
}

// ProjectCard represents an issue card in a project column
type ProjectCard struct {
	ID       int64 `json:"id"`
	IssueID  int64 `json:"issue_id"`
	ColumnID int64 `json:"column_id"`
	Sorting  int64 `json:"sorting"`
}

// AddProjectCardOption options for adding a card to a project column
type AddProjectCardOption struct {
	// required: true
	IssueID int64 `json:"issue_id" binding:"Required"`
}

// MoveProjectCardOption options for moving a card between columns
type MoveProjectCardOption struct {
	// required: true
	ColumnID int64 `json:"column_id" binding:"Required"`
	Sorting  int64 `json:"sorting"`
}
