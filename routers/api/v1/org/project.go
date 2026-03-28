// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package org

import (
	"net/http"

	"code.gitea.io/gitea/models/db"
	project_model "code.gitea.io/gitea/models/project"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers/api/v1/utils"
	"code.gitea.io/gitea/routers/common"
	"code.gitea.io/gitea/services/context"
	"code.gitea.io/gitea/services/convert"
	project_service "code.gitea.io/gitea/services/projects"
)

func getOrgProjectByID(ctx *context.APIContext) *project_model.Project {
	p, err := project_model.GetProjectForOrgByID(ctx, ctx.Org.Organization.ID, ctx.PathParamInt64("project_id"))
	if err != nil {
		if project_model.IsErrProjectNotExist(err) {
			ctx.APIErrorNotFound()
		} else {
			ctx.APIErrorInternal(err)
		}
		return nil
	}
	return p
}

func getOrgProjectColumn(ctx *context.APIContext, project *project_model.Project) *project_model.Column {
	col, err := project_model.GetColumnByIDAndProjectID(ctx, ctx.PathParamInt64("column_id"), project.ID)
	if err != nil {
		if project_model.IsErrProjectColumnNotExist(err) {
			ctx.APIErrorNotFound()
		} else {
			ctx.APIErrorInternal(err)
		}
		return nil
	}
	return col
}

// ListProjects list an organization's projects
func ListProjects(ctx *context.APIContext) {
	// swagger:operation GET /orgs/{org}/projects project orgListProjects
	// ---
	// summary: List an organization's projects
	// produces:
	// - application/json
	// parameters:
	// - name: org
	//   in: path
	//   description: name of the organization
	//   type: string
	//   required: true
	// - name: state
	//   in: query
	//   description: Project state, Recognized values are open, closed and all. Defaults to "open"
	//   type: string
	// - name: page
	//   in: query
	//   description: page number of results to return (1-based)
	//   type: integer
	// - name: limit
	//   in: query
	//   description: page size of results
	//   type: integer
	// responses:
	//   "200":
	//     "$ref": "#/responses/ProjectList"
	//   "404":
	//     "$ref": "#/responses/notFound"

	listOptions := utils.GetListOptions(ctx)
	isClosed := common.ParseIssueFilterStateIsClosed(ctx.FormString("state"))

	projects, total, err := db.FindAndCount[project_model.Project](ctx, project_model.SearchOptions{
		ListOptions: listOptions,
		OwnerID:     ctx.Org.Organization.ID,
		IsClosed:    isClosed,
		Type:        project_model.TypeOrganization,
	})
	if err != nil {
		ctx.APIErrorInternal(err)
		return
	}

	apiProjects := make([]*api.Project, len(projects))
	for i := range projects {
		apiProjects[i] = convert.ToAPIProject(projects[i])
	}

	ctx.SetTotalCountHeader(total)
	ctx.JSON(http.StatusOK, &apiProjects)
}

// CreateProject create a project for an organization
func CreateProject(ctx *context.APIContext) {
	// swagger:operation POST /orgs/{org}/projects project orgCreateProject
	// ---
	// summary: Create a project
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: org
	//   in: path
	//   description: name of the organization
	//   type: string
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/CreateProjectOption"
	// responses:
	//   "201":
	//     "$ref": "#/responses/Project"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"
	form := web.GetForm(ctx).(*api.CreateProjectOption)

	project, err := project_service.CreateProject(ctx, ctx.Org.Organization.ID, 0, project_model.TypeOrganization, ctx.Doer, project_service.CreateProjectOptions{
		Title:        form.Title,
		Description:  form.Description,
		TemplateType: project_model.TemplateType(form.TemplateType),
		CardType:     project_model.CardType(form.CardType),
	})
	if err != nil {
		ctx.APIErrorInternal(err)
		return
	}
	ctx.JSON(http.StatusCreated, convert.ToAPIProject(project))
}

// GetProject get an organization's project
func GetProject(ctx *context.APIContext) {
	// swagger:operation GET /orgs/{org}/projects/{project_id} project orgGetProject
	// ---
	// summary: Get a project
	// produces:
	// - application/json
	// parameters:
	// - name: org
	//   in: path
	//   description: name of the organization
	//   type: string
	//   required: true
	// - name: project_id
	//   in: path
	//   description: id of the project
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/Project"
	//   "404":
	//     "$ref": "#/responses/notFound"

	project := getOrgProjectByID(ctx)
	if ctx.Written() {
		return
	}

	ctx.JSON(http.StatusOK, convert.ToAPIProject(project))
}

// EditProject edit an organization's project
func EditProject(ctx *context.APIContext) {
	// swagger:operation PATCH /orgs/{org}/projects/{project_id} project orgEditProject
	// ---
	// summary: Update a project
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: org
	//   in: path
	//   description: name of the organization
	//   type: string
	//   required: true
	// - name: project_id
	//   in: path
	//   description: id of the project
	//   type: integer
	//   format: int64
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/EditProjectOption"
	// responses:
	//   "200":
	//     "$ref": "#/responses/Project"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"
	form := web.GetForm(ctx).(*api.EditProjectOption)

	project := getOrgProjectByID(ctx)
	if ctx.Written() {
		return
	}

	if form.State != nil {
		isClosed := *form.State == "closed"
		if err := project_service.ChangeProjectStatus(ctx, project, isClosed); err != nil {
			ctx.APIErrorInternal(err)
			return
		}
	}

	var cardType *project_model.CardType
	if form.CardType != nil {
		ct := project_model.CardType(*form.CardType)
		cardType = &ct
	}
	if err := project_service.UpdateProject(ctx, project, project_service.UpdateProjectOptions{
		Title:       form.Title,
		Description: form.Description,
		CardType:    cardType,
	}); err != nil {
		ctx.APIErrorInternal(err)
		return
	}

	project, err := project_model.GetProjectForOrgByID(ctx, ctx.Org.Organization.ID, project.ID)
	if err != nil {
		ctx.APIErrorInternal(err)
		return
	}
	ctx.JSON(http.StatusOK, convert.ToAPIProject(project))
}

// DeleteProject delete an organization's project
func DeleteProject(ctx *context.APIContext) {
	// swagger:operation DELETE /orgs/{org}/projects/{project_id} project orgDeleteProject
	// ---
	// summary: Delete a project
	// parameters:
	// - name: org
	//   in: path
	//   description: name of the organization
	//   type: string
	//   required: true
	// - name: project_id
	//   in: path
	//   description: id of the project
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"

	project := getOrgProjectByID(ctx)
	if ctx.Written() {
		return
	}

	if err := project_service.DeleteProject(ctx, project); err != nil {
		ctx.APIErrorInternal(err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

// ListProjectColumns list a project's columns
func ListProjectColumns(ctx *context.APIContext) {
	// swagger:operation GET /orgs/{org}/projects/{project_id}/columns project orgListProjectColumns
	// ---
	// summary: List a project's columns
	// produces:
	// - application/json
	// parameters:
	// - name: org
	//   in: path
	//   description: name of the organization
	//   type: string
	//   required: true
	// - name: project_id
	//   in: path
	//   description: id of the project
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/ProjectColumnList"
	//   "404":
	//     "$ref": "#/responses/notFound"

	project := getOrgProjectByID(ctx)
	if ctx.Written() {
		return
	}

	columns, err := project.GetColumns(ctx)
	if err != nil {
		ctx.APIErrorInternal(err)
		return
	}

	apiColumns := make([]*api.ProjectColumn, len(columns))
	for i := range columns {
		apiColumns[i] = convert.ToAPIProjectColumn(columns[i])
	}
	ctx.JSON(http.StatusOK, &apiColumns)
}

// CreateProjectColumn create a column for a project
func CreateProjectColumn(ctx *context.APIContext) {
	// swagger:operation POST /orgs/{org}/projects/{project_id}/columns project orgCreateProjectColumn
	// ---
	// summary: Create a project column
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: org
	//   in: path
	//   description: name of the organization
	//   type: string
	//   required: true
	// - name: project_id
	//   in: path
	//   description: id of the project
	//   type: integer
	//   format: int64
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/CreateProjectColumnOption"
	// responses:
	//   "201":
	//     "$ref": "#/responses/ProjectColumn"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"
	form := web.GetForm(ctx).(*api.CreateProjectColumnOption)

	project := getOrgProjectByID(ctx)
	if ctx.Written() {
		return
	}

	column, err := project_service.CreateColumn(ctx, project, ctx.Doer, project_service.CreateColumnOptions{
		Title: form.Title,
		Color: form.Color,
	})
	if err != nil {
		ctx.APIErrorInternal(err)
		return
	}
	ctx.JSON(http.StatusCreated, convert.ToAPIProjectColumn(column))
}

// GetProjectColumn get a project column
func GetProjectColumn(ctx *context.APIContext) {
	// swagger:operation GET /orgs/{org}/projects/{project_id}/columns/{column_id} project orgGetProjectColumn
	// ---
	// summary: Get a project column
	// produces:
	// - application/json
	// parameters:
	// - name: org
	//   in: path
	//   description: name of the organization
	//   type: string
	//   required: true
	// - name: project_id
	//   in: path
	//   description: id of the project
	//   type: integer
	//   format: int64
	//   required: true
	// - name: column_id
	//   in: path
	//   description: id of the column
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/ProjectColumn"
	//   "404":
	//     "$ref": "#/responses/notFound"

	project := getOrgProjectByID(ctx)
	if ctx.Written() {
		return
	}

	col := getOrgProjectColumn(ctx, project)
	if ctx.Written() {
		return
	}

	ctx.JSON(http.StatusOK, convert.ToAPIProjectColumn(col))
}

// EditProjectColumn edit a project column
func EditProjectColumn(ctx *context.APIContext) {
	// swagger:operation PATCH /orgs/{org}/projects/{project_id}/columns/{column_id} project orgEditProjectColumn
	// ---
	// summary: Update a project column
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: org
	//   in: path
	//   description: name of the organization
	//   type: string
	//   required: true
	// - name: project_id
	//   in: path
	//   description: id of the project
	//   type: integer
	//   format: int64
	//   required: true
	// - name: column_id
	//   in: path
	//   description: id of the column
	//   type: integer
	//   format: int64
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/EditProjectColumnOption"
	// responses:
	//   "200":
	//     "$ref": "#/responses/ProjectColumn"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"
	form := web.GetForm(ctx).(*api.EditProjectColumnOption)

	project := getOrgProjectByID(ctx)
	if ctx.Written() {
		return
	}

	col := getOrgProjectColumn(ctx, project)
	if ctx.Written() {
		return
	}

	if err := project_service.UpdateColumn(ctx, col, project_service.UpdateColumnOptions{
		Title: form.Title,
		Color: form.Color,
	}); err != nil {
		ctx.APIErrorInternal(err)
		return
	}

	col, err := project_model.GetColumn(ctx, col.ID)
	if err != nil {
		ctx.APIErrorInternal(err)
		return
	}
	ctx.JSON(http.StatusOK, convert.ToAPIProjectColumn(col))
}

// DeleteProjectColumn delete a project column
func DeleteProjectColumn(ctx *context.APIContext) {
	// swagger:operation DELETE /orgs/{org}/projects/{project_id}/columns/{column_id} project orgDeleteProjectColumn
	// ---
	// summary: Delete a project column
	// parameters:
	// - name: org
	//   in: path
	//   description: name of the organization
	//   type: string
	//   required: true
	// - name: project_id
	//   in: path
	//   description: id of the project
	//   type: integer
	//   format: int64
	//   required: true
	// - name: column_id
	//   in: path
	//   description: id of the column
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"

	project := getOrgProjectByID(ctx)
	if ctx.Written() {
		return
	}

	col := getOrgProjectColumn(ctx, project)
	if ctx.Written() {
		return
	}

	if col.Default {
		ctx.APIError(http.StatusForbidden, "cannot delete the default column")
		return
	}

	if err := project_service.DeleteColumn(ctx, col); err != nil {
		ctx.APIErrorInternal(err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

// MoveProjectColumn move a project column
func MoveProjectColumn(ctx *context.APIContext) {
	// swagger:operation POST /orgs/{org}/projects/{project_id}/columns/{column_id}/move project orgMoveProjectColumn
	// ---
	// summary: Move a project column
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: org
	//   in: path
	//   description: name of the organization
	//   type: string
	//   required: true
	// - name: project_id
	//   in: path
	//   description: id of the project
	//   type: integer
	//   format: int64
	//   required: true
	// - name: column_id
	//   in: path
	//   description: id of the column
	//   type: integer
	//   format: int64
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/MoveProjectColumnOption"
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"
	form := web.GetForm(ctx).(*api.MoveProjectColumnOption)

	project := getOrgProjectByID(ctx)
	if ctx.Written() {
		return
	}

	col := getOrgProjectColumn(ctx, project)
	if ctx.Written() {
		return
	}

	if err := project_model.MoveColumnToPosition(ctx, project.ID, col.ID, int8(form.Sorting)); err != nil {
		ctx.APIErrorInternal(err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

// ListProjectCards list cards in a project column
func ListProjectCards(ctx *context.APIContext) {
	// swagger:operation GET /orgs/{org}/projects/{project_id}/columns/{column_id}/cards project orgListProjectCards
	// ---
	// summary: List cards in a project column
	// produces:
	// - application/json
	// parameters:
	// - name: org
	//   in: path
	//   description: name of the organization
	//   type: string
	//   required: true
	// - name: project_id
	//   in: path
	//   description: id of the project
	//   type: integer
	//   format: int64
	//   required: true
	// - name: column_id
	//   in: path
	//   description: id of the column
	//   type: integer
	//   format: int64
	//   required: true
	// - name: page
	//   in: query
	//   description: page number of results to return (1-based)
	//   type: integer
	// - name: limit
	//   in: query
	//   description: page size of results
	//   type: integer
	// responses:
	//   "200":
	//     "$ref": "#/responses/ProjectCardList"
	//   "404":
	//     "$ref": "#/responses/notFound"

	project := getOrgProjectByID(ctx)
	if ctx.Written() {
		return
	}

	col := getOrgProjectColumn(ctx, project)
	if ctx.Written() {
		return
	}

	listOptions := utils.GetListOptions(ctx)
	cards, total, err := project_model.GetColumnCards(ctx, col.ID, listOptions)
	if err != nil {
		ctx.APIErrorInternal(err)
		return
	}

	apiCards := make([]*api.ProjectCard, len(cards))
	for i := range cards {
		apiCards[i] = convert.ToAPIProjectCard(cards[i])
	}
	ctx.SetTotalCountHeader(total)
	ctx.JSON(http.StatusOK, &apiCards)
}

// AddProjectCard add a card to a project column
func AddProjectCard(ctx *context.APIContext) {
	// swagger:operation POST /orgs/{org}/projects/{project_id}/columns/{column_id}/cards project orgAddProjectCard
	// ---
	// summary: Add a card to a project column
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: org
	//   in: path
	//   description: name of the organization
	//   type: string
	//   required: true
	// - name: project_id
	//   in: path
	//   description: id of the project
	//   type: integer
	//   format: int64
	//   required: true
	// - name: column_id
	//   in: path
	//   description: id of the column
	//   type: integer
	//   format: int64
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/AddProjectCardOption"
	// responses:
	//   "201":
	//     "$ref": "#/responses/ProjectCard"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "422":
	//     "$ref": "#/responses/validationError"
	form := web.GetForm(ctx).(*api.AddProjectCardOption)

	project := getOrgProjectByID(ctx)
	if ctx.Written() {
		return
	}

	col := getOrgProjectColumn(ctx, project)
	if ctx.Written() {
		return
	}

	card, err := project_service.AddCardToColumn(ctx, project, col, form.IssueID, -1)
	if err != nil {
		if project_model.IsErrCardAlreadyInProject(err) {
			ctx.APIError(http.StatusUnprocessableEntity, "issue already in project")
			return
		}
		ctx.APIErrorInternal(err)
		return
	}
	ctx.JSON(http.StatusCreated, convert.ToAPIProjectCard(card))
}

// DeleteProjectCard delete a card from a project
func DeleteProjectCard(ctx *context.APIContext) {
	// swagger:operation DELETE /orgs/{org}/projects/{project_id}/columns/{column_id}/cards/{card_id} project orgDeleteProjectCard
	// ---
	// summary: Delete a project card
	// parameters:
	// - name: org
	//   in: path
	//   description: name of the organization
	//   type: string
	//   required: true
	// - name: project_id
	//   in: path
	//   description: id of the project
	//   type: integer
	//   format: int64
	//   required: true
	// - name: column_id
	//   in: path
	//   description: id of the column
	//   type: integer
	//   format: int64
	//   required: true
	// - name: card_id
	//   in: path
	//   description: id of the card
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"

	project := getOrgProjectByID(ctx)
	if ctx.Written() {
		return
	}

	_ = getOrgProjectColumn(ctx, project)
	if ctx.Written() {
		return
	}

	card, err := project_model.GetProjectIssueByID(ctx, ctx.PathParamInt64("card_id"))
	if err != nil {
		if project_model.IsErrProjectCardNotExist(err) {
			ctx.APIErrorNotFound()
		} else {
			ctx.APIErrorInternal(err)
		}
		return
	}

	if err := project_service.RemoveCardFromProject(ctx, project, card.IssueID); err != nil {
		ctx.APIErrorInternal(err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

// MoveProjectCard move a card between columns
func MoveProjectCard(ctx *context.APIContext) {
	// swagger:operation POST /orgs/{org}/projects/{project_id}/columns/{column_id}/cards/{card_id}/move project orgMoveProjectCard
	// ---
	// summary: Move a project card to a column
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: org
	//   in: path
	//   description: name of the organization
	//   type: string
	//   required: true
	// - name: project_id
	//   in: path
	//   description: id of the project
	//   type: integer
	//   format: int64
	//   required: true
	// - name: column_id
	//   in: path
	//   description: id of the column
	//   type: integer
	//   format: int64
	//   required: true
	// - name: card_id
	//   in: path
	//   description: id of the card
	//   type: integer
	//   format: int64
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/MoveProjectCardOption"
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"
	form := web.GetForm(ctx).(*api.MoveProjectCardOption)

	project := getOrgProjectByID(ctx)
	if ctx.Written() {
		return
	}

	_ = getOrgProjectColumn(ctx, project)
	if ctx.Written() {
		return
	}

	card, err := project_model.GetProjectIssueByID(ctx, ctx.PathParamInt64("card_id"))
	if err != nil {
		if project_model.IsErrProjectCardNotExist(err) {
			ctx.APIErrorNotFound()
		} else {
			ctx.APIErrorInternal(err)
		}
		return
	}

	targetColumn, err := project_model.GetColumnByIDAndProjectID(ctx, form.ColumnID, project.ID)
	if err != nil {
		if project_model.IsErrProjectColumnNotExist(err) {
			ctx.APIErrorNotFound()
		} else {
			ctx.APIErrorInternal(err)
		}
		return
	}

	if err := project_service.MoveCard(ctx, ctx.Doer, project, card, targetColumn, form.Sorting); err != nil {
		ctx.APIErrorInternal(err)
		return
	}
	ctx.Status(http.StatusNoContent)
}
