package bqloader

import (
	"context"
	"fmt"
	"sync"

	"github.com/dfuse-io/shutter"

	"cloud.google.com/go/bigquery"
	"github.com/dfuse-io/derr"
	"go.uber.org/zap"
)

type jobDefinition struct {
	Table    string
	URI      string
	Callback func(ctx context.Context) error
}

func (j jobDefinition) String() string {
	return fmt.Sprintf("Table: %s | URI: %s", j.Table, j.URI)
}

type BigQueryLoader struct {
	*shutter.Shutter

	checkpointContext context.Context
	dataset           *bigquery.Dataset

	wg         sync.WaitGroup
	jobChannel chan jobDefinition
}

func NewBigQueryLoader(dataset *bigquery.Dataset, client *bigquery.Client) *BigQueryLoader {
	jobChannel := make(chan jobDefinition, 10)
	bql := &BigQueryLoader{
		Shutter:    shutter.New(),
		dataset:    dataset,
		jobChannel: jobChannel,
	}

	bql.OnTerminating(func(err error) {
		zlog.Info("waiting for current loader job to end...")
	})

	return bql
}

func (bql *BigQueryLoader) Run() {
	go func() {
		for {
			var job jobDefinition
			select {
			case <-bql.Terminating():
				return
			case job = <-bql.jobChannel:
			}

			ref := bigquery.NewGCSReference(job.URI)
			ref.SourceFormat = bigquery.Avro

			loader := bql.dataset.Table(job.Table).LoaderFrom(ref)
			loader.UseAvroLogicalTypes = true

			loadFunc := func(ctx context.Context) error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}

				job, err := loader.Run(ctx)
				if err != nil {
					return err
				}
				jobStatus, err := job.Wait(ctx)
				if err != nil {
					return err
				}
				if jobStatus.Err() != nil {
					return jobStatus.Err()
				}
				return nil
			}

			err := derr.Retry(3, loadFunc)
			if err != nil {
				zlog.Error("failed to load into big query. stopping", zap.Error(err), zap.Stringer("job", job))
				return
			}

			if job.Callback != nil {
				err := job.Callback(context.TODO())
				if err != nil {
					bql.Shutdown(err)
				}
			}
		}
	}()
}

func (bql *BigQueryLoader) SubmitJob(tableName string, uri string, callback func(ctx context.Context) error) {
	if callback == nil {
		callback = func(ctx context.Context) error { return nil }
	}

	job := jobDefinition{
		Table:    tableName,
		URI:      uri,
		Callback: callback,
	}

	select {
	case bql.jobChannel <- job:
		zlog.Info("job submitted", zap.Stringer("job", job))
	}
}
