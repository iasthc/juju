// Copyright 2025 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package model_test

import (
	"context"

	"github.com/juju/errors"
	jc "github.com/juju/testing/checkers"
	"go.uber.org/mock/gomock"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/apiserver/authentication"
	"github.com/juju/juju/apiserver/common/model"
	"github.com/juju/juju/apiserver/facade/mocks"
	"github.com/juju/juju/core/permission"
	"github.com/juju/juju/internal/testing"
)

type PermissionSuite struct {
	testing.BaseSuite
}

func (r *PermissionSuite) TestHasModelAdminSuperUser(c *gc.C) {
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()

	auth := mocks.NewMockAuthorizer(ctrl)
	auth.EXPECT().HasPermission(gomock.Any(), permission.SuperuserAccess, testing.ControllerTag).Return(nil)

	has, err := model.HasModelAdmin(context.Background(), auth, testing.ControllerTag, testing.ModelTag)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(has, jc.IsTrue)
}

func (r *PermissionSuite) TestHasModelAdminYes(c *gc.C) {
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()

	auth := mocks.NewMockAuthorizer(ctrl)
	auth.EXPECT().HasPermission(gomock.Any(), permission.SuperuserAccess, testing.ControllerTag).Return(authentication.ErrorEntityMissingPermission)
	auth.EXPECT().HasPermission(gomock.Any(), permission.AdminAccess, testing.ModelTag).Return(nil)

	has, err := model.HasModelAdmin(context.Background(), auth, testing.ControllerTag, testing.ModelTag)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(has, jc.IsTrue)
}

func (r *PermissionSuite) TestHasModelAdminNo(c *gc.C) {
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()

	auth := mocks.NewMockAuthorizer(ctrl)
	auth.EXPECT().HasPermission(gomock.Any(), permission.SuperuserAccess, testing.ControllerTag).Return(authentication.ErrorEntityMissingPermission)
	auth.EXPECT().HasPermission(gomock.Any(), permission.AdminAccess, testing.ModelTag).Return(authentication.ErrorEntityMissingPermission)

	has, err := model.HasModelAdmin(context.Background(), auth, testing.ControllerTag, testing.ModelTag)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(has, jc.IsFalse)
}

func (r *PermissionSuite) TestHasModelAdminError(c *gc.C) {
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()

	auth := mocks.NewMockAuthorizer(ctrl)
	auth.EXPECT().HasPermission(gomock.Any(), permission.SuperuserAccess, testing.ControllerTag).Return(authentication.ErrorEntityMissingPermission)
	someError := errors.New("error")
	auth.EXPECT().HasPermission(gomock.Any(), permission.AdminAccess, testing.ModelTag).Return(someError)

	has, err := model.HasModelAdmin(context.Background(), auth, testing.ControllerTag, testing.ModelTag)
	c.Assert(err, jc.ErrorIs, someError)
	c.Assert(has, jc.IsFalse)
}
