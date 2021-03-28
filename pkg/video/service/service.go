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

package service

import (
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"

	api "yunion.io/x/onecloud/pkg/apis/video"
	"yunion.io/x/onecloud/pkg/cloudcommon"
	app_common "yunion.io/x/onecloud/pkg/cloudcommon/app"
	"yunion.io/x/onecloud/pkg/cloudcommon/cronman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/video/models"
	"yunion.io/x/onecloud/pkg/video/options"
	_ "yunion.io/x/onecloud/pkg/video/tasks"
	_ "yunion.io/x/onecloud/pkg/video/videodrivers"
)

func StartService() {
	opts := &options.Options
	//commonOpts := &options.Options.CommonOptions
	baseOpts := &options.Options.BaseOptions
	dbOpts := &options.Options.DBOptions
	common_options.ParseOptions(opts, os.Args, "video.conf", api.SERVICE_TYPE)

	app := app_common.InitApp(baseOpts, true)

	InitHandlers(app)
	db.EnsureAppInitSyncDB(app, dbOpts, models.InitDB)
	defer cloudcommon.CloseDB()

	if !opts.IsSlaveNode {
		cron := cronman.InitCronJobManager(true, options.Options.CronJobWorkerCount)
		cron.AddJobAtIntervalsWithStartRun("SyncVideos", time.Duration(12)*time.Hour, models.VideoSourceManager.SyncVideos, true)
		cron.Start()
		defer cron.Stop()
	}

	app_common.ServeForever(app, baseOpts)
}
