package hostdrivers

import (
	"context"
	"fmt"
	"os"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SAzureHostDriver struct {
	SBaseHostDriver
}

func init() {
	driver := SAzureHostDriver{}
	models.RegisterHostDriver(&driver)
}

func (self *SAzureHostDriver) GetHostType() string {
	return models.HOST_TYPE_AZURE
}

func (self *SAzureHostDriver) ValidateUpdateDisk(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	if data.Contains("name") {
		return nil, httperrors.NewInputParameterError("cannot support change azure disk name")
	}
	return data, nil
}

func (self *SAzureHostDriver) CheckAndSetCacheImage(ctx context.Context, host *models.SHost, storageCache *models.SStoragecache, task taskman.ITask) error {
	params := task.GetParams()
	imageId, err := params.GetString("image_id")
	if err != nil {
		return err
	}

	osArch, _ := params.GetString("os_arch")
	osType, _ := params.GetString("os_type")
	osDist, _ := params.GetString("os_distribution")

	isForce := jsonutils.QueryBoolean(params, "is_force", false)
	userCred := task.GetUserCred()
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		lockman.LockRawObject(ctx, "cachedimages", fmt.Sprintf("%s-%s", storageCache.Id, imageId))
		defer lockman.ReleaseRawObject(ctx, "cachedimages", fmt.Sprintf("%s-%s", storageCache.Id, imageId))

		scimg := models.StoragecachedimageManager.Register(ctx, task.GetUserCred(), storageCache.Id, imageId)
		iStorageCache, err := storageCache.GetIStorageCache()
		if err != nil {
			return nil, err
		}

		extImgId, err := iStorageCache.UploadImage(userCred, imageId, osArch, osType, osDist, scimg.ExternalId, isForce)
		if err != nil {
			return nil, err
		} else {
			scimg.SetExternalId(extImgId)

			ret := jsonutils.NewDict()
			ret.Add(jsonutils.NewString(extImgId), "image_id")
			return ret, nil
		}
	})
	return nil
}

func (self *SAzureHostDriver) RequestAllocateDiskOnStorage(ctx context.Context, host *models.SHost, storage *models.SStorage, disk *models.SDisk, task taskman.ITask, content *jsonutils.JSONDict) error {
	if iCloudStorage, err := storage.GetIStorage(); err != nil {
		return err
	} else {
		if size, err := content.Int("size"); err != nil {
			return err
		} else {
			size = size >> 10
			if iDisk, err := iCloudStorage.CreateIDisk(disk.GetName(), int(size), ""); err != nil {
				return err
			} else {
				if _, err := disk.GetModelManager().TableSpec().Update(disk, func() error {
					disk.ExternalId = iDisk.GetGlobalId()

					if metaData := iDisk.GetMetadata(); metaData != nil {
						meta := make(map[string]string)
						if err := metaData.Unmarshal(meta); err != nil {
							log.Errorf("Get disk %s Metadata error: %v", disk.Name, err)
						} else {
							for key, value := range meta {
								if err := disk.SetMetadata(ctx, key, value, task.GetUserCred()); err != nil {
									log.Errorf("set disk %s mata %s => %s error: %v", disk.Name, key, value, err)
								}
							}
						}
					}

					return nil
				}); err != nil {
					log.Errorf("Update disk externalId err: %v", err)
					return err
				}
				data := jsonutils.NewDict()
				data.Add(jsonutils.NewInt(int64(iDisk.GetDiskSizeMB())), "disk_size")
				data.Add(jsonutils.NewString(iDisk.GetDiskFormat()), "disk_format")
				task.ScheduleRun(data)
			}
		}
	}
	return nil
}

func (self *SAzureHostDriver) RequestDeallocateDiskOnHost(ctx context.Context, host *models.SHost, storage *models.SStorage, disk *models.SDisk, task taskman.ITask) error {
	data := jsonutils.NewDict()
	if iCloudStorage, err := storage.GetIStorage(); err != nil {
		return err
	} else if iDisk, err := iCloudStorage.GetIDisk(disk.GetExternalId()); err != nil {
		if err == cloudprovider.ErrNotFound {
			task.ScheduleRun(data)
			return nil
		}
		return err
	} else if err := iDisk.Delete(ctx); err != nil {
		return err
	}
	task.ScheduleRun(data)
	return nil
}

func (self *SAzureHostDriver) RequestResizeDiskOnHostOnline(ctx context.Context, host *models.SHost, storage *models.SStorage, disk *models.SDisk, size int64, task taskman.ITask) error {
	return self.RequestResizeDiskOnHost(ctx, host, storage, disk, size, task)
}

func (self *SAzureHostDriver) RequestResizeDiskOnHost(ctx context.Context, host *models.SHost, storage *models.SStorage, disk *models.SDisk, size int64, task taskman.ITask) error {
	if iCloudStorage, err := storage.GetIStorage(); err != nil {
		return err
	} else if iDisk, err := iCloudStorage.GetIDisk(disk.GetExternalId()); err != nil {
		return err
	} else if err := iDisk.Resize(ctx, size>>10); err != nil {
		return err
	} else {
		task.ScheduleRun(jsonutils.Marshal(map[string]int64{"disk_size": size}))
	}
	return nil
}

func (self *SAzureHostDriver) RequestPrepareSaveDiskOnHost(ctx context.Context, host *models.SHost, disk *models.SDisk, imageId string, task taskman.ITask) error {
	task.ScheduleRun(nil)
	return nil
}

func (self *SAzureHostDriver) RequestSaveUploadImageOnHost(ctx context.Context, host *models.SHost, disk *models.SDisk, imageId string, task taskman.ITask, data jsonutils.JSONObject) error {
	if iDisk, err := disk.GetIDisk(); err != nil {
		return err
	} else if iStorage, err := disk.GetIStorage(); err != nil {
		return err
	} else if iStoragecache := iStorage.GetIStoragecache(); iStoragecache == nil {
		return httperrors.NewResourceNotFoundError("fail to find iStoragecache for storage: %s", iStorage.GetName())
	} else {
		taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
			if snapshot, err := iDisk.CreateISnapshot(ctx, fmt.Sprintf("Snapshot-%s", imageId), "PrepareSaveImage"); err != nil {
				return nil, err
			} else {
				params := task.GetParams()
				osType, _ := params.GetString("properties", "os_type")

				scimg := models.StoragecachedimageManager.Register(ctx, task.GetUserCred(), iStoragecache.GetId(), imageId)
				if scimg.Status != models.CACHED_IMAGE_STATUS_READY {
					scimg.SetStatus(task.GetUserCred(), models.CACHED_IMAGE_STATUS_CACHING, "request_prepare_save_disk_on_host")
				}
				if iImage, err := iStoragecache.CreateIImage(snapshot.GetId(), fmt.Sprintf("Image-%s", imageId), osType, ""); err != nil {
					log.Errorf("fail to create iImage: %v", err)
					scimg.SetStatus(task.GetUserCred(), models.CACHED_IMAGE_STATUS_CACHE_FAILED, err.Error())
					return nil, err
				} else {
					scimg.SetExternalId(iImage.GetId())
					if _, err := os.Stat(options.Options.TempPath); os.IsNotExist(err) {
						if err = os.MkdirAll(options.Options.TempPath, 0755); err != nil {
							return nil, err
						}
					}
					if result, err := iStoragecache.DownloadImage(task.GetUserCred(), imageId, iImage.GetId(), options.Options.TempPath); err != nil {
						scimg.SetStatus(task.GetUserCred(), models.CACHED_IMAGE_STATUS_CACHE_FAILED, err.Error())
						return nil, err
					} else {
						if err := iImage.Delete(); err != nil {
							log.Errorf("Delete iImage %s failed: %v", iImage.GetId(), err)
						}
						if err := snapshot.Delete(); err != nil {
							log.Errorf("Delete snapshot %s failed: %v", snapshot.GetId(), err)
						}
						scimg.SetStatus(task.GetUserCred(), models.CACHED_IMAGE_STATUS_READY, "")
						return result, nil
					}
				}
			}
		})
	}
	return nil
}

func (self *SAzureHostDriver) RequestDeleteSnapshotWithStorage(ctx context.Context, host *models.SHost, snapshot *models.SSnapshot, task taskman.ITask) error {
	return httperrors.NewNotImplementedError("not implement")
}
