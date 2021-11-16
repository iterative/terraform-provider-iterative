package machine

import (
	"context"
	"bytes"
	"io"

	_ "github.com/rclone/rclone/backend/local"
	_ "github.com/rclone/rclone/backend/azureblob"
	_ "github.com/rclone/rclone/backend/googlecloudstorage"
	_ "github.com/rclone/rclone/backend/s3"

	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/sync"
)

func Logs(ctx context.Context, remote string) ([]string, error) {
	remoteFileSystem, err := fs.NewFs(ctx, remote)
	if err != nil {
		return nil, err
	}

	entries, err := remoteFileSystem.List(ctx, "/log/task")
	if err != nil {
		return nil, err
	}

	var logs []string
	for _, entry := range entries {
		object, err := remoteFileSystem.NewObject(ctx, entry.Remote())
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