package machine

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"path/filepath"
	"strings"

	_ "github.com/rclone/rclone/backend/azureblob"
	_ "github.com/rclone/rclone/backend/googlecloudstorage"
	_ "github.com/rclone/rclone/backend/local"
	_ "github.com/rclone/rclone/backend/s3"

	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/sync"

	"terraform-provider-iterative/task/common"
)

type StatusReport struct {
	Result string
	Status string
	Code   string
}

func Reports(ctx context.Context, remote, prefix string) ([]string, error) {
	remoteFileSystem, err := fs.NewFs(ctx, remote)
	if err != nil {
		return nil, err
	}

	entries, err := remoteFileSystem.List(ctx, "/reports")
	if err != nil {
		return nil, err
	}

	var logs []string
	for _, entry := range entries {
		path := entry.Remote()
		if base := filepath.Base(path); !strings.HasPrefix(base, prefix+"-") {
			continue
		}

		object, err := remoteFileSystem.NewObject(ctx, path)
		if err != nil {
			return nil, err
		}
		reader, err := object.Open(ctx)
		if err != nil {
			return nil, err
		}
		buffer := new(bytes.Buffer)
		if _, err := io.Copy(buffer, reader); err != nil {
			return nil, err
		}
		logs = append(logs, buffer.String())
		reader.Close()
	}

	return logs, nil
}

func Logs(ctx context.Context, remote string) ([]string, error) {
	return Reports(ctx, remote, "task")
}

func Status(ctx context.Context, remote string, initialStatus common.Status) (common.Status, error) {
	reports, err := Reports(ctx, remote, "status")
	if err != nil {
		return initialStatus, err
	}

	for _, report := range reports {
		var statusReport StatusReport
		if err := json.Unmarshal([]byte(report), &statusReport); err != nil {
			return initialStatus, err
		}
		if statusReport.Code != "" {
			if statusReport.Code == "0" {
				initialStatus[common.StatusCodeSucceeded] += 1
			} else {
				initialStatus[common.StatusCodeFailed] += 1
			}
		}
	}
	return initialStatus, nil
}

func Transfer(ctx context.Context, source, destination string) error {
	sourceFileSystem, err := fs.NewFs(ctx, source)
	if err != nil {
		return err
	}

	destinationFileSystem, err := fs.NewFs(ctx, destination)
	if err != nil {
		return err
	}

	if err := sync.CopyDir(ctx, destinationFileSystem, sourceFileSystem, true); err != nil {
		return err
	}

	return nil
}
