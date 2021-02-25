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
	URI        string
	Table      string
	Dataset    *bigquery.Dataset
	DataFormat bigquery.DataFormat
	Callback   func(ctx context.Context) error
}

func (j jobDefinition) String() string {
	return fmt.Sprintf("Table: %s | URI: %s", j.Table, j.URI)
}

type BigQueryInjector struct {
	*shutter.Shutter

	checkpointContext context.Context

	wg         sync.WaitGroup
	jobChannel chan jobDefinition
}

func NewBigQueryInjector() *BigQueryInjector {
	jobChannel := make(chan jobDefinition, 10)
	injector := &BigQueryInjector{
		Shutter:    shutter.New(),
		jobChannel: jobChannel,
	}

	injector.OnTerminating(func(err error) {
		zlog.Info("waiting for current loader jobs to end...")
		injector.wg.Wait()
		zlog.Info("all loader jobs completed.")
	})

	return injector
}

func (inj *BigQueryInjector) Run() {
	go func() {
		for {
			var job jobDefinition
			select {
			case <-inj.Terminating():
				return
			case job = <-inj.jobChannel:
				inj.wg.Add(1)
			}

			ref := bigquery.NewGCSReference(job.URI)
			ref.SourceFormat = job.DataFormat

			loader := job.Dataset.Table(job.Table).LoaderFrom(ref)
			loader.UseAvroLogicalTypes = true

			if err := derr.Retry(3, func(ctx context.Context) error {
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
			}); err != nil {
				inj.wg.Done()
				inj.Shutdown(err)
				return
			}

			if err := derr.Retry(3, job.Callback); err != nil {
				inj.wg.Done()
				inj.Shutdown(err)
				return
			}
		}
	}()
}

func (inj *BigQueryInjector) SubmitJob(uri string, tableName string, dataset *bigquery.Dataset, format bigquery.DataFormat, callback func(ctx context.Context) error) {
	if callback == nil {
		//noop
		callback = func(ctx context.Context) error { return nil }
	}

	job := jobDefinition{
		URI:        uri,
		Table:      tableName,
		Dataset:    dataset,
		DataFormat: format,
		Callback:   callback,
	}

	select {
	case inj.jobChannel <- job:
		zlog.Info("job submitted", zap.Stringer("job", job))
	}
}
