// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"testing"

	project_model "code.gitea.io/gitea/models/project"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// issueProjectTemplate returns a minimal markdown issue template (with YAML
// frontmatter) that references the given project title in its projects: field.
// We use .md rather than .yaml because YAML form templates require a non-empty
// body: field, whereas markdown templates only need name + about.
func issueProjectTemplate(projectTitle string) string {
	return fmt.Sprintf("---\nname: Bug\nabout: Report a bug\nprojects: [%q]\n---\nDescribe the bug.\n", projectTitle)
}

// createRepoProjectForTest inserts a new repository-scoped project and returns it.
func createRepoProjectForTest(t *testing.T, repo *repo_model.Repository, creator *user_model.User, title string) *project_model.Project {
	t.Helper()
	p := &project_model.Project{
		Title:        title,
		RepoID:       repo.ID,
		CreatorID:    creator.ID,
		Type:         project_model.TypeRepository,
		TemplateType: project_model.TemplateTypeNone,
		CardType:     project_model.CardTypeTextOnly,
	}
	require.NoError(t, project_model.NewProject(t.Context(), p))
	return p
}

// projectSelectedNeedle returns the string that appears in the hidden
// project_ids input when exactly project ID n is the (only) selected project.
// The template renders:
//
//	<input class="combo-value" name="project_ids" type="hidden" value="N">
//
// so the value field holds the comma-joined IDs; for a single selection the
// needle is unique and only present when that project is preselected.
func projectSelectedNeedle(id int64) string {
	return `name="project_ids" type="hidden" value="` + strconv.FormatInt(id, 10) + `"`
}

func assertProjectPreselected(t *testing.T, body string, projectID int64) {
	t.Helper()
	assert.Contains(t, body, projectSelectedNeedle(projectID),
		"expected project %d to be preselected in new-issue page", projectID)
}

func assertProjectNotPreselected(t *testing.T, body string, projectID int64) {
	t.Helper()
	assert.NotContains(t, body, projectSelectedNeedle(projectID),
		"did not expect project %d to be preselected", projectID)
}

// TestIssueTemplatePrefillsProject verifies that a YAML issue template with a
// projects: field causes the new-issue page to render that project preselected.
func TestIssueTemplatePrefillsProject(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, _ *url.URL) {
		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "user2"})
		repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{OwnerName: "user2", Name: "repo1"})

		p := createRepoProjectForTest(t, repo, user, "Triage")
		require.NoError(t, createOrReplaceFileInBranch(user, repo,
			".gitea/ISSUE_TEMPLATE/bug.md", repo.DefaultBranch, issueProjectTemplate("Triage")))

		session := loginUser(t, user.Name)
		req := NewRequest(t, "GET", fmt.Sprintf("/%s/issues/new?template=.gitea%%2FISSUE_TEMPLATE%%2Fbug.md",
			repo.FullName()))
		resp := session.MakeRequest(t, req, http.StatusOK)

		assertProjectPreselected(t, resp.Body.String(), p.ID)
	})
}

// TestIssueTemplateProjectTitleUnknownDrops verifies that an issue template
// referencing a project that does not exist on the repo silently drops the
// project (no error banner, form still loads).
func TestIssueTemplateProjectTitleUnknownDrops(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, _ *url.URL) {
		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "user2"})
		repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{OwnerName: "user2", Name: "repo1"})

		require.NoError(t, createOrReplaceFileInBranch(user, repo,
			".gitea/ISSUE_TEMPLATE/bug.md", repo.DefaultBranch, issueProjectTemplate("Nonexistent")))

		session := loginUser(t, user.Name)
		req := NewRequest(t, "GET", fmt.Sprintf("/%s/issues/new?template=.gitea%%2FISSUE_TEMPLATE%%2Fbug.md",
			repo.FullName()))
		resp := session.MakeRequest(t, req, http.StatusOK)

		body := resp.Body.String()
		// No flash error about the template
		assert.NotContains(t, body, "invalid template")
		// The new-issue form rendered successfully
		assert.Contains(t, body, `name="title"`)
		// Hidden input has empty value — no project selected
		assert.Contains(t, body, `name="project_ids" type="hidden" value=""`)
	})
}

// TestPullRequestTemplatePrefillsProject verifies that a YAML PR template with
// a projects: field causes the compare/create-PR page to render the project
// preselected.
func TestPullRequestTemplatePrefillsProject(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, _ *url.URL) {
		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "user2"})
		repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{OwnerName: "user2", Name: "repo1"})

		p := createRepoProjectForTest(t, repo, user, "PR Triage")

		// Use a markdown PR template: YAML PR templates require body: fields,
		// while markdown templates only need name + about in the frontmatter.
		// The auto-detected single-file location is used; directory-based PR
		// templates (PULL_REQUEST_TEMPLATE/<name>.yaml) are not supported.
		prTemplateBody := fmt.Sprintf("---\nname: PR\nabout: Pull request\nprojects: [%q]\n---\nDescribe the change.\n", "PR Triage")
		require.NoError(t, createOrReplaceFileInBranch(user, repo,
			".gitea/PULL_REQUEST_TEMPLATE.md", repo.DefaultBranch, prTemplateBody))

		// A compare page only shows the PR form when head != base commit.
		// Create a feature branch with one additional commit.
		featureBranch := "template-test-pr-branch"
		_, err := createFileInBranch(user, repo, createFileInBranchOptions{
			OldBranch: repo.DefaultBranch,
			NewBranch: featureBranch,
		}, map[string]string{"feature.txt": "hello"})
		require.NoError(t, err)

		session := loginUser(t, user.Name)
		req := NewRequest(t, "GET", fmt.Sprintf("/%s/compare/%s...%s",
			repo.FullName(), repo.DefaultBranch, featureBranch))
		resp := session.MakeRequest(t, req, http.StatusOK)

		assertProjectPreselected(t, resp.Body.String(), p.ID)
	})
}

// TestIssueTemplateProjectVisibilityHonored verifies the happy path: a project
// that is visible to the signed-in user IS preselected from the template.
//
// The structural defense for the negative case (cannot preselect a project the
// actor cannot see) is that retrieveProjectsDataForIssueWriter only loads
// projects the actor is allowed to see, so SetSelectedProjectTitles can never
// resolve a hidden project. A meaningful negative fixture requires a repo where
// the actor can read issues but not a specific project; that is left as a
// follow-up once such a fixture is available.
func TestIssueTemplateProjectVisibilityHonored(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, _ *url.URL) {
		owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "user2"})
		repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{OwnerName: "user2", Name: "repo1"})

		p := createRepoProjectForTest(t, repo, owner, "Triage")
		require.NoError(t, createOrReplaceFileInBranch(owner, repo,
			".gitea/ISSUE_TEMPLATE/bug.md", repo.DefaultBranch, issueProjectTemplate("Triage")))

		session := loginUser(t, owner.Name)
		req := NewRequest(t, "GET", fmt.Sprintf("/%s/issues/new?template=.gitea%%2FISSUE_TEMPLATE%%2Fbug.md",
			repo.FullName()))
		resp := session.MakeRequest(t, req, http.StatusOK)

		assertProjectPreselected(t, resp.Body.String(), p.ID)
		_ = assertProjectNotPreselected // keep helper referenced for future negative cases
	})
}
