// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package storage_test

import (
	"context"

	"github.com/juju/errors"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/rpc/params"
	"github.com/juju/juju/state"
)

type filesystemSuite struct {
	baseStorageSuite
}

var _ = gc.Suite(&filesystemSuite{})

func (s *filesystemSuite) expectedFilesystemDetails() params.FilesystemDetails {
	return params.FilesystemDetails{
		FilesystemTag: s.filesystemTag.String(),
		Life:          "alive",
		Status: params.EntityStatus{
			Status: "attached",
		},
		MachineAttachments: map[string]params.FilesystemAttachmentDetails{
			s.machineTag.String(): {
				Life: "dead",
			},
		},
		UnitAttachments: map[string]params.FilesystemAttachmentDetails{},
		Storage: &params.StorageDetails{
			StorageTag: "storage-data-0",
			OwnerTag:   "unit-mysql-0",
			Kind:       params.StorageKindFilesystem,
			Life:       "dying",
			Status: params.EntityStatus{
				Status: "attached",
			},
			Attachments: map[string]params.StorageAttachmentDetails{
				"unit-mysql-0": {
					StorageTag: "storage-data-0",
					UnitTag:    "unit-mysql-0",
					MachineTag: "machine-66",
					Life:       "alive",
				},
			},
		},
	}
}

func (s *filesystemSuite) TestListFilesystemsEmptyFilter(c *gc.C) {
	defer s.setupMocks(c).Finish()

	found, err := s.api.ListFilesystems(context.Background(), params.FilesystemFilters{
		[]params.FilesystemFilter{{}},
	})
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(found.Results, gc.HasLen, 1)
	c.Assert(found.Results[0].Result, gc.HasLen, 1)
	c.Assert(found.Results[0].Result[0], gc.DeepEquals, s.expectedFilesystemDetails())
}

func (s *filesystemSuite) TestListFilesystemsError(c *gc.C) {
	defer s.setupMocks(c).Finish()

	msg := "inventing error"
	s.storageAccessor.allFilesystems = func() ([]state.Filesystem, error) {
		return nil, errors.New(msg)
	}
	results, err := s.api.ListFilesystems(context.Background(), params.FilesystemFilters{
		[]params.FilesystemFilter{{}},
	})
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(results.Results, gc.HasLen, 1)
	c.Assert(results.Results[0].Error, gc.ErrorMatches, msg)
}

func (s *filesystemSuite) TestListFilesystemsNoFilesystems(c *gc.C) {
	defer s.setupMocks(c).Finish()

	s.storageAccessor.allFilesystems = func() ([]state.Filesystem, error) {
		return nil, nil
	}
	results, err := s.api.ListFilesystems(context.Background(), params.FilesystemFilters{})
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(results.Results, gc.HasLen, 0)
}

func (s *filesystemSuite) TestListFilesystemsFilter(c *gc.C) {
	defer s.setupMocks(c).Finish()

	filters := []params.FilesystemFilter{{
		Machines: []string{s.machineTag.String()},
	}}
	found, err := s.api.ListFilesystems(context.Background(), params.FilesystemFilters{filters})
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(found.Results, gc.HasLen, 1)
	c.Assert(found.Results[0].Result, gc.HasLen, 1)
	c.Assert(found.Results[0].Result[0], jc.DeepEquals, s.expectedFilesystemDetails())
}

func (s *filesystemSuite) TestListFilesystemsFilterNonMatching(c *gc.C) {
	defer s.setupMocks(c).Finish()

	filters := []params.FilesystemFilter{{
		Machines: []string{"machine-42"},
	}}
	found, err := s.api.ListFilesystems(context.Background(), params.FilesystemFilters{filters})
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(found.Results, gc.HasLen, 1)
	c.Assert(found.Results[0].Error, gc.IsNil)
	c.Assert(found.Results[0].Result, gc.HasLen, 0)
}

func (s *filesystemSuite) TestListFilesystemsFilesystemInfo(c *gc.C) {
	defer s.setupMocks(c).Finish()

	s.filesystem.info = &state.FilesystemInfo{
		Size: 123,
	}
	expected := s.expectedFilesystemDetails()
	expected.Info.Size = 123
	found, err := s.api.ListFilesystems(context.Background(), params.FilesystemFilters{
		[]params.FilesystemFilter{{}},
	})
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(found.Results, gc.HasLen, 1)
	c.Assert(found.Results[0].Result, gc.HasLen, 1)
	c.Assert(found.Results[0].Result[0], jc.DeepEquals, expected)
}

func (s *filesystemSuite) TestListFilesystemsAttachmentInfo(c *gc.C) {
	defer s.setupMocks(c).Finish()

	s.filesystemAttachment.info = &state.FilesystemAttachmentInfo{
		MountPoint: "/tmp",
		ReadOnly:   true,
	}
	expected := s.expectedFilesystemDetails()
	expected.MachineAttachments[s.machineTag.String()] = params.FilesystemAttachmentDetails{
		FilesystemAttachmentInfo: params.FilesystemAttachmentInfo{
			MountPoint: "/tmp",
			ReadOnly:   true,
		},
		Life: "dead",
	}
	expectedStorageAttachmentDetails := expected.Storage.Attachments["unit-mysql-0"]
	expectedStorageAttachmentDetails.Location = "/tmp"
	expected.Storage.Attachments["unit-mysql-0"] = expectedStorageAttachmentDetails
	found, err := s.api.ListFilesystems(context.Background(), params.FilesystemFilters{
		[]params.FilesystemFilter{{}},
	})
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(found.Results, gc.HasLen, 1)
	c.Assert(found.Results[0].Result, gc.HasLen, 1)
	c.Assert(found.Results[0].Result[0], jc.DeepEquals, expected)
}

func (s *filesystemSuite) TestListFilesystemsVolumeBacked(c *gc.C) {
	defer s.setupMocks(c).Finish()

	s.filesystem.volume = &s.volumeTag
	expected := s.expectedFilesystemDetails()
	expected.VolumeTag = s.volumeTag.String()
	found, err := s.api.ListFilesystems(context.Background(), params.FilesystemFilters{
		[]params.FilesystemFilter{{}},
	})
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(found.Results, gc.HasLen, 1)
	c.Assert(found.Results[0].Result, gc.HasLen, 1)
	c.Assert(found.Results[0].Result[0], jc.DeepEquals, expected)
}
