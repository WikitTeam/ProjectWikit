package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/url"
	"path"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type S3Config struct {
	Endpoint  string
	Region    string
	Bucket    string
	Prefix    string
	PathStyle bool
	AccessKey string
	SecretKey string
}

type S3 struct {
	client *minio.Client
	bucket string
	prefix string
}

var _ Store = (*S3)(nil)

func NewS3(cfg S3Config) (*S3, error) {
	u, err := url.Parse(cfg.Endpoint)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("storage: endpoint %q is not a URL such as https://s3.example.com", cfg.Endpoint)
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return nil, fmt.Errorf("storage: endpoint %q must start with https:// or http://", cfg.Endpoint)
	}
	if cfg.Bucket == "" {
		return nil, errors.New("storage: no bucket is set")
	}
	if cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, errors.New("storage: the access key or the secret key is missing")
	}
	lookup := minio.BucketLookupDNS
	if cfg.PathStyle {
		lookup = minio.BucketLookupPath
	}
	client, err := minio.New(u.Host, &minio.Options{
		Creds:        credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure:       u.Scheme == "https",
		Region:       cfg.Region,
		BucketLookup: lookup,
	})
	if err != nil {
		return nil, fmt.Errorf("storage: %w", err)
	}
	prefix := strings.Trim(cfg.Prefix, "/")
	if prefix != "" {
		prefix += "/"
	}
	return &S3{client: client, bucket: cfg.Bucket, prefix: prefix}, nil
}

func (s *S3) object(key string) (string, error) {
	if err := CheckKey(key); err != nil {
		return "", err
	}
	return s.prefix + key, nil
}

func missing(err error) bool {
	code := minio.ToErrorResponse(err).Code
	return code == "NoSuchKey" || code == "NotFound"
}

func wrap(err error) error {
	if err != nil && missing(err) {
		return ErrNotExist
	}
	return err
}

type s3File struct {
	*minio.Object
	info Info
}

func (f s3File) Info() Info { return f.info }

func (s *S3) Open(ctx context.Context, key string) (File, error) {
	name, err := s.object(key)
	if err != nil {
		return nil, err
	}
	obj, err := s.client.GetObject(ctx, s.bucket, name, minio.GetObjectOptions{})
	if err != nil {
		return nil, wrap(err)
	}
	st, err := obj.Stat()
	if err != nil {
		obj.Close()
		return nil, wrap(err)
	}
	return s3File{Object: obj, info: Info{Size: st.Size, ModTime: st.LastModified}}, nil
}

func (s *S3) Stat(ctx context.Context, key string) (Info, error) {
	name, err := s.object(key)
	if err != nil {
		return Info{}, err
	}
	st, err := s.client.StatObject(ctx, s.bucket, name, minio.StatObjectOptions{})
	if err != nil {
		return Info{}, wrap(err)
	}
	return Info{Size: st.Size, ModTime: st.LastModified}, nil
}

// A bucket served straight to readers answers with the type stored here, so it
// is set from the name the way the files directory would have been read.
func (s *S3) Put(ctx context.Context, key string, r io.Reader, size int64) error {
	name, err := s.object(key)
	if err != nil {
		return err
	}
	kind := mime.TypeByExtension(strings.ToLower(path.Ext(key)))
	if kind == "" {
		kind = "application/octet-stream"
	}
	_, err = s.client.PutObject(ctx, s.bucket, name, r, size, minio.PutObjectOptions{ContentType: kind})
	return err
}

func (s *S3) Remove(ctx context.Context, key string) error {
	name, err := s.object(key)
	if err != nil {
		return err
	}
	if err := s.client.RemoveObject(ctx, s.bucket, name, minio.RemoveObjectOptions{}); err != nil && !missing(err) {
		return err
	}
	return nil
}

func (s *S3) RemoveAll(ctx context.Context, prefix string) error {
	prefix = strings.TrimSuffix(prefix, "/")
	if err := CheckKey(prefix); err != nil {
		return err
	}
	objects := make(chan minio.ObjectInfo)
	go func() {
		defer close(objects)
		for obj := range s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{Prefix: s.prefix + prefix + "/", Recursive: true}) {
			if obj.Err != nil {
				return
			}
			objects <- obj
		}
	}()
	for result := range s.client.RemoveObjects(ctx, s.bucket, objects, minio.RemoveObjectsOptions{}) {
		if result.Err != nil && !missing(result.Err) {
			return result.Err
		}
	}
	return nil
}

func (s *S3) Walk(ctx context.Context, prefix string, fn func(key string, info Info) error) error {
	from := s.prefix
	if prefix = strings.TrimSuffix(prefix, "/"); prefix != "" {
		if err := CheckKey(prefix); err != nil {
			return err
		}
		from += prefix + "/"
	}
	walkCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	for obj := range s.client.ListObjects(walkCtx, s.bucket, minio.ListObjectsOptions{Prefix: from, Recursive: true}) {
		if obj.Err != nil {
			return obj.Err
		}
		if err := fn(strings.TrimPrefix(obj.Key, s.prefix), Info{Size: obj.Size, ModTime: obj.LastModified}); err != nil {
			return err
		}
	}
	return nil
}
