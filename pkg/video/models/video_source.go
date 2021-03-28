// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package models

import (
	"context"
	"fmt"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"

	api "yunion.io/x/onecloud/pkg/apis/video"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SVideoSourceManager struct {
	db.SEnabledStatusInfrasResourceBaseManager
}

var VideoSourceManager *SVideoSourceManager

func init() {
	VideoSourceManager = &SVideoSourceManager{
		SEnabledStatusInfrasResourceBaseManager: db.NewEnabledStatusInfrasResourceBaseManager(
			SVideoSource{},
			"video_sources_tbl",
			"video_source",
			"video_sources",
		),
	}
	VideoSourceManager.SetVirtualObject(VideoSourceManager)
}

type SVideoSource struct {
	db.SEnabledStatusInfrasResourceBase

	Identity string `width:"128" charset:"ascii" nullable:"false"`
}

func (manager *SVideoSourceManager) InitializeData() error {
	for identity, name := range api.SUPPORTED_VIDEO_SOURCES {
		q := manager.Query().Equals("identity", identity)
		count, err := q.CountWithError()
		if err != nil {
			return errors.Wrapf(err, "CountWithError")
		}
		if count == 0 {
			videoSource := &SVideoSource{}
			videoSource.SetModelManager(manager, videoSource)
			videoSource.Name = name
			videoSource.Identity = identity
			videoSource.Enabled = tristate.True
			err = manager.TableSpec().Insert(context.TODO(), videoSource)
			if err != nil {
				return errors.Wrapf(err, "Insert")
			}
		}
	}
	return nil
}

func (manager *SVideoSourceManager) SyncVideos(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	q := manager.Query().IsTrue("enabled")
	sources := []SVideoSource{}
	err := db.FetchModelObjects(manager, q, &sources)
	if err != nil {
		log.Errorf("db.FetchModelObjects error: %v", err)
		return
	}
	for i := range sources {
		err = sources[i].StartSyncVideosTask(ctx, userCred)
		if err != nil {
			log.Errorf("StartSyncVideosTask for %s error: %v", sources[i].Name, err)
			continue
		}
	}
}

func (self *SVideoSource) StartSyncVideosTask(ctx context.Context, userCred mcclient.TokenCredential) error {
	task, err := taskman.TaskManager.NewTask(ctx, "VideoSyncTask", self, userCred, nil, "", "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (self *SVideoSource) StartSyncCartoonTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "CartoonSyncTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (self *SVideoSource) GetDriver() IVideoDriver {
	driver, ok := videoDrivers[self.Identity]
	if !ok {
		panic(fmt.Sprintf("%s not register", self.Identity))
	}
	return driver
}
