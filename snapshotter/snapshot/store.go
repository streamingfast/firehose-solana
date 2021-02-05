package snapshot

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

// listFiles lists objects within specified bucket.
func listFiles(ctx context.Context, client *storage.Client, bucket string, prefix string, fileHandler func(ctx context.Context, client *storage.Client, bucket string, file string) error) ([]string, error) {

	ctx, cancel := context.WithTimeout(ctx, time.Second*200)
	defer cancel()

	it := client.Bucket(bucket).Objects(ctx, &storage.Query{
		Prefix: prefix,
	})
	var files []string
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Bucket(%q).Objects: %v", bucket, err)
		}
		if fileHandler != nil {
			err := fileHandler(ctx, client, bucket, attrs.Name)
			if err != nil {
				return nil, fmt.Errorf("file handler: %w", err)
			}
		}
		files = append(files, attrs.Name)
	}
	return files, nil
}
