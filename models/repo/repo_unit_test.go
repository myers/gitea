// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"strings"
	"testing"

	"code.gitea.io/gitea/models/perm"
	"code.gitea.io/gitea/models/unit"

	"github.com/stretchr/testify/assert"
)

func TestProjectsConfigSerialization(t *testing.T) {
	cfg := &ProjectsConfig{
		ProjectsMode:     ProjectsModeAll,
		DefaultProjectID: 42,
		AutoAssignIssues: true,
		AutoAssignPRs:    false,
	}

	data, err := cfg.ToDB()
	assert.NoError(t, err)

	cfg2 := &ProjectsConfig{}
	err = cfg2.FromDB(data)
	assert.NoError(t, err)
	assert.Equal(t, ProjectsModeAll, cfg2.GetProjectsMode())
	assert.Equal(t, int64(42), cfg2.GetDefaultProjectID())
	assert.True(t, cfg2.AutoAssignIssues)
	assert.False(t, cfg2.AutoAssignPRs)
}

func TestProjectsConfigDefaultValues(t *testing.T) {
	cfg := &ProjectsConfig{}
	err := cfg.FromDB([]byte("{}"))
	assert.NoError(t, err)
	assert.Equal(t, int64(0), cfg.GetDefaultProjectID())
	assert.False(t, cfg.AutoAssignIssues)
	assert.False(t, cfg.AutoAssignPRs)
}

func TestActionsConfig(t *testing.T) {
	cfg := &ActionsConfig{}
	cfg.DisableWorkflow("test1.yaml")
	assert.Equal(t, []string{"test1.yaml"}, cfg.DisabledWorkflows)

	cfg.DisableWorkflow("test1.yaml")
	assert.Equal(t, []string{"test1.yaml"}, cfg.DisabledWorkflows)

	cfg.EnableWorkflow("test1.yaml")
	assert.Equal(t, []string{}, cfg.DisabledWorkflows)

	cfg.EnableWorkflow("test1.yaml")
	assert.Equal(t, []string{}, cfg.DisabledWorkflows)

	cfg.DisableWorkflow("test1.yaml")
	cfg.DisableWorkflow("test2.yaml")
	cfg.DisableWorkflow("test3.yaml")
	assert.Equal(t, "test1.yaml,test2.yaml,test3.yaml", strings.Join(cfg.DisabledWorkflows, ","))
}

func TestActionsConfigTokenPermissions(t *testing.T) {
	t.Run("Default Permission Mode", func(t *testing.T) {
		cfg := &ActionsConfig{TokenPermissionMode: "invalid-value"}
		_ = cfg.FromDB(nil)
		assert.Equal(t, ActionsTokenPermissionModePermissive, cfg.TokenPermissionMode)
		assert.Equal(t, perm.AccessModeWrite, cfg.GetDefaultTokenPermissions().UnitAccessModes[unit.TypeCode])
	})

	t.Run("Explicit Permission Mode", func(t *testing.T) {
		cfg := &ActionsConfig{
			TokenPermissionMode: ActionsTokenPermissionModeRestricted,
		}
		assert.Equal(t, ActionsTokenPermissionModeRestricted, cfg.TokenPermissionMode)
	})

	t.Run("Effective Permissions - Permissive Mode", func(t *testing.T) {
		cfg := &ActionsConfig{
			TokenPermissionMode: ActionsTokenPermissionModePermissive,
		}
		defaultPerms := cfg.GetDefaultTokenPermissions()
		perms := cfg.ClampPermissions(defaultPerms)
		assert.Equal(t, perm.AccessModeWrite, perms.UnitAccessModes[unit.TypeCode])
		assert.Equal(t, perm.AccessModeWrite, perms.UnitAccessModes[unit.TypeIssues])
		assert.Equal(t, perm.AccessModeWrite, perms.UnitAccessModes[unit.TypePackages])
	})

	t.Run("Effective Permissions - Restricted Mode", func(t *testing.T) {
		cfg := &ActionsConfig{
			TokenPermissionMode: ActionsTokenPermissionModeRestricted,
		}
		defaultPerms := cfg.GetDefaultTokenPermissions()
		perms := cfg.ClampPermissions(defaultPerms)
		assert.Equal(t, perm.AccessModeRead, perms.UnitAccessModes[unit.TypeCode])
		assert.Equal(t, perm.AccessModeNone, perms.UnitAccessModes[unit.TypeIssues])
		assert.Equal(t, perm.AccessModeRead, perms.UnitAccessModes[unit.TypePackages])
	})

	t.Run("Clamp Permissions", func(t *testing.T) {
		cfg := &ActionsConfig{
			MaxTokenPermissions: &ActionsTokenPermissions{
				UnitAccessModes: map[unit.Type]perm.AccessMode{
					unit.TypeCode:         perm.AccessModeRead,
					unit.TypeIssues:       perm.AccessModeWrite,
					unit.TypePullRequests: perm.AccessModeRead,
					unit.TypePackages:     perm.AccessModeRead,
					unit.TypeActions:      perm.AccessModeNone,
					unit.TypeWiki:         perm.AccessModeWrite,
				},
			},
		}
		input := ActionsTokenPermissions{
			UnitAccessModes: map[unit.Type]perm.AccessMode{
				unit.TypeCode:         perm.AccessModeWrite, // Should be clamped to Read
				unit.TypeIssues:       perm.AccessModeWrite, // Should stay Write
				unit.TypePullRequests: perm.AccessModeWrite, // Should be clamped to Read
				unit.TypePackages:     perm.AccessModeWrite, // Should be clamped to Read
				unit.TypeActions:      perm.AccessModeRead,  // Should be clamped to None
				unit.TypeWiki:         perm.AccessModeRead,  // Should stay Read
			},
		}
		clamped := cfg.ClampPermissions(input)
		assert.Equal(t, perm.AccessModeRead, clamped.UnitAccessModes[unit.TypeCode])
		assert.Equal(t, perm.AccessModeWrite, clamped.UnitAccessModes[unit.TypeIssues])
		assert.Equal(t, perm.AccessModeRead, clamped.UnitAccessModes[unit.TypePullRequests])
		assert.Equal(t, perm.AccessModeRead, clamped.UnitAccessModes[unit.TypePackages])
		assert.Equal(t, perm.AccessModeNone, clamped.UnitAccessModes[unit.TypeActions])
		assert.Equal(t, perm.AccessModeRead, clamped.UnitAccessModes[unit.TypeWiki])
	})
}
