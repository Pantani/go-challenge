package mock_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/gloveboxhq/glovebox-go-code-challenge/filestore"
	"github.com/gloveboxhq/glovebox-go-code-challenge/filestore/mock"
)

func concurrentRound(writer, reader *mock.Client, name string) error {
	ctx := context.Background()
	if err := writer.Set(ctx, name, []byte(name), "text/plain"); err != nil {
		return err
	}
	got, err := tryReadFile(reader, name)
	if err != nil {
		return err
	}
	if got != name {
		return fmt.Errorf("content = %q, want %q", got, name)
	}
	return concurrentTransfers(writer, reader, name)
}

func concurrentTransfers(writer, reader *mock.Client, name string) error {
	ctx := context.Background()
	if err := reader.Copy(ctx, name, name+"-copy"); err != nil {
		return err
	}
	if err := writer.Move(ctx, name+"-copy", name); !errors.Is(err, filestore.ErrFileExists) {
		return fmt.Errorf("move conflict = %v, want ErrFileExists", err)
	}
	if _, err := reader.GetPresignedURL(ctx, name, time.Minute); err != nil {
		return err
	}
	if err := writer.Move(ctx, name+"-copy", name+"-moved"); err != nil {
		return err
	}
	return errors.Join(writer.Purge(ctx, name), reader.Purge(ctx, name+"-moved"))
}

func runConcurrentWorker(writer, reader *mock.Client, name string) error {
	for range 50 {
		if err := concurrentRound(writer, reader, name); err != nil {
			return err
		}
	}
	return nil
}

func TestConcurrentSharedClients(t *testing.T) {
	bucket := mock.NewSharedBucket(mock.Bucket{})
	writer := mock.NewClientFromSharedBucket(bucket, "")
	reader := mock.NewClientFromSharedBucket(bucket, "")
	results := make(chan error, 8)
	var workers sync.WaitGroup
	for worker := range 8 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			results <- runConcurrentWorker(writer, reader, fmt.Sprintf("worker-%d", worker))
		}()
	}
	workers.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Error(err)
		}
	}
}
