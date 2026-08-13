package backup

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"os"
	"path/filepath"
	"strings"
)

// ObjectPutter is the narrow S3-compatible client surface used by S3Sink.
type ObjectPutter interface {
	PutObject(ctx context.Context, bucket, key string, body io.Reader, size int64, contentType string) error
}

// S3Sink implements Sink by uploading each file under localDir with PutObject.
type S3Sink struct {
	Client ObjectPutter
	Bucket string
}

// NewS3Sink returns a directory-tree Sink. bucket must be non-empty at use time.
func NewS3Sink(client ObjectPutter, bucket string) *S3Sink {
	return &S3Sink{Client: client, Bucket: bucket}
}

func (s *S3Sink) Upload(ctx context.Context, localDir, objectPrefix string) error {
	if s == nil || s.Client == nil {
		return fmt.Errorf("s3 sink not configured")
	}
	if s.Bucket == "" {
		return fmt.Errorf("s3 bucket empty")
	}
	prefix := strings.TrimSuffix(objectPrefix, "/")
	return filepath.WalkDir(localDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(localDir, path)
		if err != nil {
			return err
		}
		key := prefix + "/" + filepath.ToSlash(rel)
		info, err := d.Info()
		if err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		ct := contentTypeFor(rel)
		return s.Client.PutObject(ctx, s.Bucket, key, f, info.Size(), ct)
	})
}

func contentTypeFor(rel string) string {
	ext := filepath.Ext(rel)
	if ext == "" {
		return "application/octet-stream"
	}
	if t := mime.TypeByExtension(ext); t != "" {
		return t
	}
	return "application/octet-stream"
}
