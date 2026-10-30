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

	"golang.org/x/sync/errgroup"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/compute/options"
)

type regionSyncConcurrencyKeyType struct{}
type regionSyncSlotHoldKeyType struct{}

var (
	regionSyncConcurrencyKey = regionSyncConcurrencyKeyType{}
	regionSyncSlotHoldKey    = regionSyncSlotHoldKeyType{}
)

type regionSyncSlotHold struct {
	sem  chan struct{}
	held bool
}

func (h *regionSyncSlotHold) release() {
	if h == nil || !h.held {
		return
	}
	<-h.sem
	h.held = false
}

func (h *regionSyncSlotHold) acquire(ctx context.Context) error {
	if h == nil || h.held {
		return nil
	}
	select {
	case h.sem <- struct{}{}:
		h.held = true
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func regionSyncConcurrencyLimit() int {
	limit := options.Options.CloudRegionResourceSyncConcurrency
	if limit <= 0 {
		return 1
	}
	return limit
}

// withRegionSyncConcurrency attaches a per-DoSync semaphore so nested
// runSyncTasksParallel calls (wave/zone/host/vm/lb/rds) share one budget of
// CloudRegionResourceSyncConcurrency. Multi-region concurrency remains
// governed by CloudSyncWorkerCount separately.
func withRegionSyncConcurrency(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Value(regionSyncConcurrencyKey).(chan struct{}); ok {
		return ctx
	}
	limit := regionSyncConcurrencyLimit()
	return context.WithValue(ctx, regionSyncConcurrencyKey, make(chan struct{}, limit))
}

func regionSyncConcurrencySem(ctx context.Context) chan struct{} {
	if ctx != nil {
		if sem, ok := ctx.Value(regionSyncConcurrencyKey).(chan struct{}); ok && sem != nil {
			return sem
		}
	}
	return make(chan struct{}, regionSyncConcurrencyLimit())
}

// runSyncTasksParallel 在单 region DoSync 内对无依赖任务做有限并发。
// 使用 DoSync 上下文中的共享 semaphore；嵌套调用时会临时释放父任务占有的槽位，
// 避免 zone×host×vm 各自 SetLimit 叠加成 C³，也避免父子同时占槽死锁。
// task 必须使用入参 ctx（含 slot hold），以便嵌套 parallel 能正确让出槽位。
func runSyncTasksParallel(ctx context.Context, tasks []func(ctx context.Context)) {
	if len(tasks) == 0 {
		return
	}
	sem := regionSyncConcurrencySem(ctx)
	limit := cap(sem)

	// Nested fan-out: free the caller's slot while children compete for the budget.
	if hold, ok := ctx.Value(regionSyncSlotHoldKey).(*regionSyncSlotHold); ok {
		hold.release()
		defer func() {
			if err := hold.acquire(ctx); err != nil {
				log.Infof("runSyncTasksParallel reacquire aborted: %v", err)
			}
		}()
	}

	if limit <= 1 || len(tasks) == 1 {
		for _, task := range tasks {
			if err := CheckCloudaccountSyncCanceled(ctx); err != nil {
				log.Infof("runSyncTasksParallel aborted: %v", err)
				return
			}
			task(ctx)
		}
		return
	}

	g, gctx := errgroup.WithContext(ctx)
	for i := range tasks {
		task := tasks[i]
		g.Go(func() error {
			if err := CheckCloudaccountSyncCanceled(gctx); err != nil {
				return err
			}
			select {
			case sem <- struct{}{}:
			case <-gctx.Done():
				return gctx.Err()
			}
			hold := &regionSyncSlotHold{sem: sem, held: true}
			tctx := context.WithValue(gctx, regionSyncSlotHoldKey, hold)
			defer hold.release()
			task(tctx)
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		log.Infof("runSyncTasksParallel finished with: %v", err)
	}
}

// CheckCloudaccountSyncCanceled 检查同步 ctx 是否已取消。
// 账号级 sync-task 取消树不在本改动范围；此处仅透传 ctx.Err()，供 wave/并行边界提前退出。
func CheckCloudaccountSyncCanceled(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}
