package esxi

import (
	"github.com/vmware/govmomi/vim25/mo"

	"context"
	"fmt"
	"github.com/vmware/govmomi/vim25/types"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/vmdkutils"
)

var DATASTORE_PROPS = []string{"name", "parent", "info", "summary", "host"}

type SDatastore struct {
	SManagedObject

	ihosts []cloudprovider.ICloudHost
}

func NewDatastore(manager *SESXiClient, ds *mo.Datastore, dc *SDatacenter) *SDatastore {
	return &SDatastore{SManagedObject: newManagedObject(manager, ds, dc)}
}

func (self *SDatastore) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SDatastore) getDatastore() *mo.Datastore {
	return self.object.(*mo.Datastore)
}

func (self *SDatastore) GetGlobalId() string {
	volId, err := self.getVolumeId()
	if err != nil {
		log.Fatalf("datastore global ID error %s", err)
	}
	return volId
}

func (self *SDatastore) GetName() string {
	volName, err := self.getVolumeName()
	if err != nil {
		log.Fatalf("datastore get name error %s", err)
	}
	return fmt.Sprintf("%s-%s", self.getVolumeType(), volName)
}

func (self *SDatastore) GetCapacityMB() int {
	moStore := self.getDatastore()
	return int(moStore.Summary.Capacity / 1024 / 1024)
}

func (self *SDatastore) GetEnabled() bool {
	return true
}

func (self *SDatastore) GetStatus() string {
	if self.getDatastore().Summary.Accessible {
		return models.STORAGE_ONLINE
	} else {
		return models.STORAGE_OFFLINE
	}
}

func (self *SDatastore) Refresh() error {
	return cloudprovider.ErrNotImplemented
}

func (self *SDatastore) IsEmulated() bool {
	return false
}

func (self *SDatastore) getVolumeId() (string, error) {
	moStore := self.getDatastore()
	vmfsInfo, ok := moStore.Info.(*types.VmfsDatastoreInfo)
	if ok {
		if vmfsInfo.Vmfs.Local != nil && *vmfsInfo.Vmfs.Local {
			host, err := self.getLocalHost()
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("%s:%s", host.GetAccessIp(), vmfsInfo.Vmfs.Uuid), nil
		} else {
			return vmfsInfo.Vmfs.Uuid, nil
		}
	}
	nasInfo, ok := moStore.Info.(*types.NasDatastoreInfo)
	if ok {
		return fmt.Sprintf("%s:%s", nasInfo.Nas.RemoteHost, nasInfo.Nas.RemotePath), nil
	}
	if moStore.Summary.Type == "vsan" {
		vsanId := moStore.Summary.Url
		vsanId = vsanId[strings.Index(vsanId, "vsan:"):]

		endIdx := len(vsanId)
		for ; vsanId[endIdx-1] == '/'; endIdx -= 1 {
		}

		return vsanId[:endIdx], nil
	}
	log.Fatalf("unsupported volume type %#v", moStore.Info)
	return "", cloudprovider.ErrNotImplemented
}

func (self *SDatastore) getVolumeType() string {
	return self.getDatastore().Summary.Type
}

func (self *SDatastore) getVolumeName() (string, error) {
	moStore := self.getDatastore()

	if self.isLocalVMFS() {
		host, err := self.getLocalHost()
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s-%s", host.GetAccessIp(), moStore.Info.GetDatastoreInfo().Name), nil
	}
	dc, err := self.GetDatacenter()
	if err != nil {
		return "", nil
	}
	return fmt.Sprintf("%s-%s", dc.GetName(), moStore.Info.GetDatastoreInfo().Name), nil
}

func (self *SDatastore) getAttachedHosts() ([]cloudprovider.ICloudHost, error) {
	ihosts := make([]cloudprovider.ICloudHost, 0)

	moStore := self.getDatastore()
	for i := 0; i < len(moStore.Host); i += 1 {
		idstr := moRefId(moStore.Host[i].Key)
		host, err := self.datacenter.GetIHostByMoId(idstr)
		if err != nil {
			return nil, err
		}
		ihosts = append(ihosts, host)
	}

	return ihosts, nil
}

func (self *SDatastore) getCachedAttachedHosts() ([]cloudprovider.ICloudHost, error) {
	if self.ihosts == nil {
		var err error
		self.ihosts, err = self.getAttachedHosts()
		if err != nil {
			return nil, err
		}
	}
	return self.ihosts, nil
}

func (self *SDatastore) GetAttachedHosts() ([]cloudprovider.ICloudHost, error) {
	return self.getCachedAttachedHosts()
}

func (self *SDatastore) getLocalHost() (cloudprovider.ICloudHost, error) {
	hosts, err := self.GetAttachedHosts()
	if err != nil {
		return nil, err
	}
	if len(hosts) == 1 {
		return hosts[0], nil
	}
	return nil, cloudprovider.ErrInvalidStatus
}

func (self *SDatastore) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return nil
}

func (self *SDatastore) GetIZone() cloudprovider.ICloudZone {
	return nil
}

func (self *SDatastore) GetIDisk(idStr string) (cloudprovider.ICloudDisk, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SDatastore) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SDatastore) isLocalVMFS() bool {
	moStore := self.getDatastore()
	vmfsInfo, ok := moStore.Info.(*types.VmfsDatastoreInfo)
	if ok && vmfsInfo.Vmfs.Local != nil && *vmfsInfo.Vmfs.Local {
		return true
	}
	return false
}

func (self *SDatastore) GetStorageType() string {
	moStore := self.getDatastore()
	switch moStore.Summary.Type {
	case "VMFS":
		if self.isLocalVMFS() {
			return models.STORAGE_LOCAL
		} else {
			return models.STORAGE_NAS
		}
	case "NFS", "NFS41", "CIFS", "vsan":
		return models.STORAGE_NAS
	default:
		log.Fatalf("unsupported datastore type %s", moStore.Summary.Type)
		return ""
	}
}

func (self *SDatastore) GetMediumType() string {
	moStore := self.getDatastore()
	vmfsInfo, ok := moStore.Info.(*types.VmfsDatastoreInfo)
	if ok && vmfsInfo.Vmfs.Ssd != nil && *vmfsInfo.Vmfs.Ssd {
		return models.DISK_TYPE_SSD
	}
	return models.DISK_TYPE_ROTATE
}

func (self *SDatastore) GetStorageConf() jsonutils.JSONObject {
	conf := jsonutils.NewDict()
	conf.Add(jsonutils.NewString(self.GetName()), "name")
	conf.Add(jsonutils.NewString(self.GetGlobalId()), "id")
	conf.Add(jsonutils.NewString(self.GetDatacenterPathString()), "dc_path")
	volId, err := self.getVolumeId()
	if err != nil {
		log.Errorf("getVaolumeId fail %s", err)
	}
	conf.Add(jsonutils.NewString(volId), "volume_id")

	volType := self.getVolumeType()
	conf.Add(jsonutils.NewString(volType), "volume_type")

	volName, err := self.getVolumeName()
	if err != nil {
		log.Errorf("getVaolumeName fail %s", err)
	}
	conf.Add(jsonutils.NewString(volName), "volume_name")
	return conf
}

func (self *SDatastore) GetManagerId() string {
	return self.manager.providerId
}

func (self *SDatastore) GetUrl() string {
	return self.getDatastore().Info.GetDatastoreInfo().Url
}

func (self *SDatastore) cleanPath(remotePath string) string {
	dsName := fmt.Sprintf("[%s]", self.SManagedObject.GetName())
	dsUrl := self.GetUrl()
	if strings.HasPrefix(remotePath, dsName) {
		remotePath = remotePath[len(dsName):]
	} else if strings.HasPrefix(remotePath, dsUrl) {
		remotePath = remotePath[len(dsUrl):]
	}
	return strings.TrimSpace(remotePath)
}

func pathEscape(path string) string {
	segs := strings.Split(path, "/")
	for i := 0; i < len(segs); i += 1 {
		segs[i] = url.PathEscape(segs[i])
	}
	return strings.Join(segs, "/")
}

func (self *SDatastore) GetPathUrl(remotePath string) string {
	remotePath = self.cleanPath(remotePath)
	if len(remotePath) == 0 || remotePath[0] != '/' {
		remotePath = fmt.Sprintf("/%s", remotePath)
	}
	httpUrl := fmt.Sprintf("%s/folder%s", self.getManagerUri(), pathEscape(remotePath))
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(self.SManagedObject.GetName()), "dsName")
	params.Add(jsonutils.NewString(self.GetDatacenterPathString()), "dcPath")

	return fmt.Sprintf("%s?%s", httpUrl, params.QueryString())
}

func (self *SDatastore) getPathString(path string) string {
	for len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}
	return fmt.Sprintf("[%s] %s", self.SManagedObject.GetName(), path)
}

func (self *SDatastore) CreateIDisk(name string, sizeGb int, desc string) (cloudprovider.ICloudDisk, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SDatastore) FileGetContent(ctx context.Context, remotePath string) ([]byte, error) {
	url := self.GetPathUrl(remotePath)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var bytes []byte

	err = self.manager.client.Do(ctx, req, func(resp *http.Response) error {
		if resp.StatusCode >= 400 {
			return fmt.Errorf("%s", resp.Status)
		}
		cont, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		bytes = cont
		return nil
	})

	return bytes, err
}

type SDatastoreFileInfo struct {
	Url      string
	Name     string
	Date     time.Time
	FileType string
	Size     uint64
}

const (
	fileListPattern = `<tr><td><a href="(?P<url>[\w\d:#@%/;$()~_?\+-=\\\.&]+)">(?P<name>[^<]+)<\/a></td><td align="right">(?P<date>[^<]+)</td><td align="right">(?P<size>[^<]+)</td></tr>`
	fileDateFormat  = "02-Jan-2006 15:04"
	fileDateFormat2 = "Mon, 2 Jan 2006 15:04:05 GMT"
)

var (
	fileListRegexp = regexp.MustCompile(fileListPattern)
)

func (self *SDatastore) ListDir(ctx context.Context, remotePath string) ([]SDatastoreFileInfo, error) {
	listContent, err := self.FileGetContent(ctx, remotePath)
	if err != nil {
		return nil, err
	}
	ret := make([]SDatastoreFileInfo, 0)
	matches := fileListRegexp.FindAllStringSubmatch(string(listContent), -1)
	for r := 0; r < len(matches); r += 1 {
		url := strings.TrimSpace(matches[r][1])
		name := strings.TrimSpace(matches[r][2])
		dateStr := strings.TrimSpace(matches[r][3])
		sizeStr := strings.TrimSpace(matches[r][4])
		var ftype string
		var size uint64
		if sizeStr == "-" {
			ftype = "dir"
			size = 0
		} else {
			ftype = "file"
			size, _ = strconv.ParseUint(sizeStr, 10, 64)
		}
		date, err := time.Parse(fileDateFormat, dateStr)
		if err != nil {
			return nil, err
		}
		info := SDatastoreFileInfo{
			Url:      url,
			Name:     name,
			FileType: ftype,
			Size:     size,
			Date:     date,
		}
		ret = append(ret, info)
	}

	return ret, nil
}

func (self *SDatastore) CheckFile(ctx context.Context, remotePath string) (*SDatastoreFileInfo, error) {
	url := self.GetPathUrl(remotePath)

	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return nil, err
	}

	var size uint64
	var date time.Time

	err = self.manager.client.Do(ctx, req, func(resp *http.Response) error {
		if resp.StatusCode >= 400 {
			return fmt.Errorf("%s", resp.Status)
		}
		sizeStr := resp.Header.Get("Content-Length")
		size, _ = strconv.ParseUint(sizeStr, 10, 64)

		dateStr := resp.Header.Get("Date")
		date, _ = time.Parse(fileDateFormat2, dateStr)
		return nil
	})

	if err != nil {
		return nil, err
	}
	return &SDatastoreFileInfo{Date: date, Size: size}, nil
}

func (self *SDatastore) Download(ctx context.Context, remotePath string, writer io.Writer) error {
	url := self.GetPathUrl(remotePath)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	err = self.manager.client.Do(ctx, req, func(resp *http.Response) error {
		if resp.StatusCode >= 400 {
			return fmt.Errorf("%s", resp.Status)
		}
		buffer := make([]byte, 4096)
		for {
			rn, re := resp.Body.Read(buffer)
			if rn > 0 {
				wo := 0
				for wo < rn {
					wn, we := writer.Write(buffer[wo:rn])
					if we != nil {
						return we
					}
					wo += wn
				}
			}
			if re != nil {
				if re != io.EOF {
					return re
				} else {
					break
				}
			}
		}
		return nil
	})

	return err
}

func (self *SDatastore) Upload(ctx context.Context, remotePath string, body io.Reader) error {
	url := self.GetPathUrl(remotePath)

	req, err := http.NewRequest("PUT", url, body)
	if err != nil {
		return err
	}

	err = self.manager.client.Do(ctx, req, func(resp *http.Response) error {
		if resp.StatusCode >= 400 {
			return fmt.Errorf("%s", resp.Status)
		}
		_, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		// log.Debugf("upload respose %s", buffer)
		return nil
	})

	return err
}

func (self *SDatastore) FilePutContent(ctx context.Context, remotePath string, content string) error {
	return self.Upload(ctx, remotePath, strings.NewReader(content))
}

func (self *SDatastore) Delete(ctx context.Context, remotePath string) error {
	url := self.GetPathUrl(remotePath)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	err = self.manager.client.Do(ctx, req, func(resp *http.Response) error {
		if resp.StatusCode >= 400 {
			return fmt.Errorf("%s", resp.Status)
		}
		_, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		// log.Debugf("delete respose %s", buffer)
		return nil
	})

	return err
}

func (self *SDatastore) DeleteVmdk(ctx context.Context, remotePath string) error {
	vmdkContent, err := self.FileGetContent(ctx, remotePath)
	if err != nil {
		return err
	}
	vmdkInfo, err := vmdkutils.Parse(string(vmdkContent))
	if err != nil {
		return err
	}
	err = self.Delete(ctx, remotePath)
	if err != nil {
		return err
	}
	if len(vmdkInfo.ExtentFile) > 0 {
		err = self.Delete(ctx, path.Join(remotePath, vmdkInfo.ExtentFile))
		if err != nil {
			return err
		}
	}
	return nil
}
