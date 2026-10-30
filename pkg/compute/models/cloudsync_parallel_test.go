// Copyright 2019 Yunion
package models

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"yunion.io/x/onecloud/pkg/compute/options"
)

func TestSSyncResultSetConcurrentAdd(t *testing.T) {
	set := NewSSyncResultSet()
	const n = 100
	done := make(chan struct{}, n)
	for i := 0; i < n; i++ {
		go func() {
			unlock := set.lock()
			res := set.getOrCreate("tests")
			res.AddCnt++
			unlock()
			done <- struct{}{}
		}()
	}
	for i := 0; i < n; i++ {
		<-done
	}
	unlock := set.lock()
	res := set.data["tests"]
	unlock()
	if res == nil || res.AddCnt != n {
		t.Fatalf("expected AddCnt=%d got %#v", n, res)
	}
}

func TestRunSyncTasksParallelCancel(t *testing.T) {
	old := options.Options.CloudRegionResourceSyncConcurrency
	options.Options.CloudRegionResourceSyncConcurrency = 3
	defer func() { options.Options.CloudRegionResourceSyncConcurrency = old }()

	ctx, cancel := context.WithCancel(context.Background())
	ctx = withRegionSyncConcurrency(ctx)
	var ran int32
	tasks := make([]func(context.Context), 0, 10)
	for i := 0; i < 10; i++ {
		tasks = append(tasks, func(context.Context) {
			atomic.AddInt32(&ran, 1)
			time.Sleep(50 * time.Millisecond)
		})
	}
	cancel()
	runSyncTasksParallel(ctx, tasks)
	if atomic.LoadInt32(&ran) > 10 {
		t.Fatalf("unexpected ran=%d", ran)
	}
}

func TestRunSyncTasksParallelLimit(t *testing.T) {
	old := options.Options.CloudRegionResourceSyncConcurrency
	options.Options.CloudRegionResourceSyncConcurrency = 2
	defer func() { options.Options.CloudRegionResourceSyncConcurrency = old }()

	ctx := withRegionSyncConcurrency(context.Background())
	var peak, cur int32
	tasks := make([]func(context.Context), 0, 6)
	for i := 0; i < 6; i++ {
		tasks = append(tasks, func(context.Context) {
			v := atomic.AddInt32(&cur, 1)
			for {
				p := atomic.LoadInt32(&peak)
				if v <= p || atomic.CompareAndSwapInt32(&peak, p, v) {
					break
				}
			}
			time.Sleep(30 * time.Millisecond)
			atomic.AddInt32(&cur, -1)
		})
	}
	runSyncTasksParallel(ctx, tasks)
	if peak > 2 {
		t.Fatalf("peak concurrency %d > limit 2", peak)
	}
	if peak < 1 {
		t.Fatalf("expected some concurrency, peak=%d", peak)
	}
}

// Nested wave/zone/host style calls must share one DoSync budget (not multiply / deadlock).
// Peak is measured by semaphore occupancy (parents release slots while waiting on children).
func TestRunSyncTasksParallelNestedSharedLimit(t *testing.T) {
	old := options.Options.CloudRegionResourceSyncConcurrency
	options.Options.CloudRegionResourceSyncConcurrency = 2
	defer func() { options.Options.CloudRegionResourceSyncConcurrency = old }()

	ctx := withRegionSyncConcurrency(context.Background())
	sem := regionSyncConcurrencySem(ctx)
	var peak int32
	stop := make(chan struct{})
	var monitorDone sync.WaitGroup
	monitorDone.Add(1)
	go func() {
		defer monitorDone.Done()
		for {
			select {
			case <-stop:
				return
			default:
				n := int32(len(sem))
				for {
					p := atomic.LoadInt32(&peak)
					if n <= p || atomic.CompareAndSwapInt32(&peak, p, n) {
						break
					}
				}
				time.Sleep(time.Millisecond)
			}
		}
	}()

	done := make(chan struct{})
	go func() {
		outer := make([]func(context.Context), 0, 3)
		for i := 0; i < 3; i++ {
			outer = append(outer, func(ctx context.Context) {
				time.Sleep(5 * time.Millisecond)
				inner := make([]func(context.Context), 0, 3)
				for j := 0; j < 3; j++ {
					inner = append(inner, func(context.Context) {
						time.Sleep(20 * time.Millisecond)
					})
				}
				runSyncTasksParallel(ctx, inner)
			})
		}
		runSyncTasksParallel(ctx, outer)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("nested parallel deadlocked")
	}
	close(stop)
	monitorDone.Wait()
	if peak > 2 {
		t.Fatalf("nested peak semaphore occupancy %d > shared limit 2", peak)
	}
	if peak < 1 {
		t.Fatalf("expected some semaphore use, peak=%d", peak)
	}
}

func TestStorageCachePairsConcurrentMerge(t *testing.T) {
	old := options.Options.CloudRegionResourceSyncConcurrency
	options.Options.CloudRegionResourceSyncConcurrency = 3
	defer func() { options.Options.CloudRegionResourceSyncConcurrency = old }()

	ctx := withRegionSyncConcurrency(context.Background())
	var pairsMu sync.Mutex
	storageCachePairs := make([]sStoragecacheSyncPair, 0)
	tasks := make([]func(context.Context), 0, 6)
	for i := 0; i < 6; i++ {
		tasks = append(tasks, func(context.Context) {
			pairsMu.Lock()
			_ = append([]sStoragecacheSyncPair(nil), storageCachePairs...)
			pairsMu.Unlock()
			time.Sleep(5 * time.Millisecond)
			pairsMu.Lock()
			storageCachePairs = append(storageCachePairs, sStoragecacheSyncPair{isNew: true})
			pairsMu.Unlock()
		})
	}
	runSyncTasksParallel(ctx, tasks)
	if len(storageCachePairs) != 6 {
		t.Fatalf("expected 6 pairs, got %d", len(storageCachePairs))
	}
}

func TestSSyncResultSetMarshalJSON(t *testing.T) {
	set := NewSSyncResultSet()
	unlock := set.lock()
	res := set.getOrCreate("compute")
	res.AddCnt = 3
	unlock()
	b, err := set.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if len(b) < 10 || string(b) == "{}" {
		t.Fatalf("unexpected marshal %s", b)
	}
}

func TestSyncResultGroupByKeyword(t *testing.T) {
	cases := map[string]string{
		"hosts":                   "compute",
		"disks":                   "compute",
		"hoststorages":            "compute",
		"servers":                 "compute",
		"vpcs":                    "network",
		"networks":                "network",
		"secgroups":               "network",
		"eips":                    "network",
		"buckets":                 "objectstore",
		"loadbalancers":           "loadbalancer",
		"loadbalanceracls":        "loadbalancer",
		"dbinstances":             "rds",
		"dbinstancebackups":       "rds",
		"elasticcaches":           "cache",
		"file_systems":            "nas",
		"access_groups":           "nas",
		"waf_instances":           "waf",
		"cloud_kube_clusters":     "container",
		"cachedimages":            "image",
		"cloudproviderquotas":     "quota",
		"zones":                   "zone",
		"unknown_custom_resource": "unknown_custom_resource",
	}
	for kw, want := range cases {
		if got := syncResultGroupByKeyword(kw); got != want {
			t.Fatalf("%s: got %s want %s", kw, got, want)
		}
	}
}
