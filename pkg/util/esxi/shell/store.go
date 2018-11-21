package shell

import (
	"context"
	"os"
	"yunion.io/x/onecloud/pkg/util/esxi"
	"yunion.io/x/onecloud/pkg/util/printutils"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type DatastoreListOptions struct {
		DATACENTER string `help:"List datastores in datacenter"`
	}
	shellutils.R(&DatastoreListOptions{}, "ds-list", "List datastores in datacenter", func(cli *esxi.SESXiClient, args *DatastoreListOptions) error {
		dc, err := cli.FindDatacenterByMoId(args.DATACENTER)
		if err != nil {
			return err
		}
		ds, err := dc.GetIStorages()
		if err != nil {
			return err
		}
		printList(ds, nil)
		return nil
	})

	type DatastoreShowOptions struct {
		DATACENTER string `help:"Datacenter"`
		DSID       string `help:"Datastore ID""`
	}
	shellutils.R(&DatastoreShowOptions{}, "ds-show", "Show details of a datastore", func(cli *esxi.SESXiClient, args *DatastoreShowOptions) error {
		dc, err := cli.FindDatacenterByMoId(args.DATACENTER)
		if err != nil {
			return err
		}
		ds, err := dc.GetIStorageByMoId(args.DSID)
		if err != nil {
			return err
		}
		printObject(ds)
		return nil
	})

	type DatastoreListDirOptions struct {
		DATACENTER string `help:"Datacenter"`
		DSID       string `help:"Datastore ID""`
		DIR        string `help:"directory"`
	}
	shellutils.R(&DatastoreListDirOptions{}, "ds-list-dir", "List directory of a datastore", func(cli *esxi.SESXiClient, args *DatastoreListDirOptions) error {
		dc, err := cli.FindDatacenterByMoId(args.DATACENTER)
		if err != nil {
			return err
		}
		ds, err := dc.GetIStorageByMoId(args.DSID)
		if err != nil {
			return err
		}
		ctx := context.Background()
		dsObj := ds.(*esxi.SDatastore)
		fileList, err := dsObj.ListDir(ctx, args.DIR)
		if err != nil {
			return err
		}
		printutils.PrintInterfaceList(fileList, 0, 0, 0, []string{"Name", "Date", "Size"})
		return nil
	})

	shellutils.R(&DatastoreListDirOptions{}, "ds-check-file", "Check file status in a datastore", func(cli *esxi.SESXiClient, args *DatastoreListDirOptions) error {
		dc, err := cli.FindDatacenterByMoId(args.DATACENTER)
		if err != nil {
			return err
		}
		ds, err := dc.GetIStorageByMoId(args.DSID)
		if err != nil {
			return err
		}
		ctx := context.Background()
		dsObj := ds.(*esxi.SDatastore)
		file, err := dsObj.CheckFile(ctx, args.DIR)
		if err != nil {
			return err
		}
		printutils.PrintInterfaceObject(file)
		return nil
	})

	shellutils.R(&DatastoreListDirOptions{}, "ds-delete-file", "Delete file in a datastore", func(cli *esxi.SESXiClient, args *DatastoreListDirOptions) error {
		dc, err := cli.FindDatacenterByMoId(args.DATACENTER)
		if err != nil {
			return err
		}
		ds, err := dc.GetIStorageByMoId(args.DSID)
		if err != nil {
			return err
		}
		ctx := context.Background()
		dsObj := ds.(*esxi.SDatastore)
		err = dsObj.Delete(ctx, args.DIR)
		if err != nil {
			return err
		}
		return nil
	})

	type DatastoreDownloadOptions struct {
		DATACENTER string `help:"Datacenter"`
		DSID       string `help:"Datastore ID""`
		DIR        string `help:"directory"`
		LOCAL      string `help:"local file"`
	}
	shellutils.R(&DatastoreDownloadOptions{}, "ds-download", "Download file from a datastore", func(cli *esxi.SESXiClient, args *DatastoreDownloadOptions) error {
		dc, err := cli.FindDatacenterByMoId(args.DATACENTER)
		if err != nil {
			return err
		}
		ds, err := dc.GetIStorageByMoId(args.DSID)
		if err != nil {
			return err
		}
		ctx := context.Background()
		dsObj := ds.(*esxi.SDatastore)

		file, err := os.Create(args.LOCAL)
		if err != nil {
			return err
		}
		defer file.Close()

		err = dsObj.Download(ctx, args.DIR, file)
		if err != nil {
			return err
		}

		return nil
	})

	shellutils.R(&DatastoreDownloadOptions{}, "ds-upload", "Upload local file to datastore", func(cli *esxi.SESXiClient, args *DatastoreDownloadOptions) error {
		dc, err := cli.FindDatacenterByMoId(args.DATACENTER)
		if err != nil {
			return err
		}
		ds, err := dc.GetIStorageByMoId(args.DSID)
		if err != nil {
			return err
		}
		ctx := context.Background()
		dsObj := ds.(*esxi.SDatastore)

		file, err := os.Open(args.LOCAL)
		if err != nil {
			return err
		}
		defer file.Close()

		err = dsObj.Upload(ctx, args.DIR, file)
		if err != nil {
			return err
		}

		return nil
	})

}
