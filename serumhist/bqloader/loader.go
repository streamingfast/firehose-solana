package bqloader

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/atomic"

	"github.com/dfuse-io/shutter"

	"cloud.google.com/go/bigquery"
	"github.com/dfuse-io/derr"
	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
	"go.uber.org/zap"
	"google.golang.org/api/iterator"
)

type jobDefinition struct {
	Context context.Context
	Table   string
	URI     string
}

func (j jobDefinition) String() string {
	return fmt.Sprintf("Table: %s | URI: %s", j.Table, j.URI)
}

type BigQueryLoader struct {
	*shutter.Shutter

	wg                sync.WaitGroup
	checkpointContext context.Context
	client            *bigquery.Client
	dataset           *bigquery.Dataset
	checkpointsTable  string

	jobChannel chan jobDefinition

	healthy atomic.Bool
}

func NewBigQueryLoader(dataset *bigquery.Dataset, client *bigquery.Client, checkpointTable string) *BigQueryLoader {
	jobChannel := make(chan jobDefinition, 10)
	bql := &BigQueryLoader{
		Shutter:          shutter.New(),
		dataset:          dataset,
		client:           client,
		checkpointsTable: checkpointTable,
		jobChannel:       jobChannel,
	}

	bql.OnTerminating(func(err error) {
		zlog.Info("waiting for current loader job to end...")
		bql.wg.Wait()
	})

	bql.healthy.Store(true)

	return bql
}

func (bql *BigQueryLoader) IsHealthy() bool {
	return bql.healthy.Load()
}

func (bql *BigQueryLoader) Run(ctx context.Context) {
	go func() {
		for {
			var job jobDefinition
			select {
			case <-ctx.Done():
				return
			case job = <-bql.jobChannel:
			}

			bql.wg.Add(1)

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

			err := derr.RetryContext(job.Context, 3, loadFunc)
			if err != nil {
				zlog.Error("failed to load into big query. stopping", zap.Error(err), zap.Stringer("job", job))
				bql.wg.Done()
				bql.healthy.Store(false)
				return
			}

			jobStatusRow := struct {
				Table    string    `bigquery:"table"`
				Filename string    `bigquery:"file"`
				Time     time.Time `bigquery:"timestamp"`
			}{
				Table:    job.Table,
				Filename: job.URI,
				Time:     time.Now(),
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			err = bql.dataset.Table(processedFiles).Inserter().Put(ctx, jobStatusRow)
			cancel()
			bql.wg.Done()

			if err != nil {
				zlog.Error("could not write checkpoint", zap.String("table", bql.checkpointsTable), zap.Error(err))
				bql.healthy.Store(false)
				return
			}

			zlog.Info("checkpoint written", zap.String("table", bql.checkpointsTable), zap.Stringer("job", job))
		}
	}()
}

func (bql *BigQueryLoader) SubmitJob(ctx context.Context, tableName string, uri string) {
	job := jobDefinition{
		Context: ctx,
		Table:   tableName,
		URI:     uri,
	}

	select {
	case <-ctx.Done():
		return
	case bql.jobChannel <- job:
		zlog.Info("job submitted", zap.Stringer("job", job))
	}
}

func (bql *BigQueryLoader) ReadCheckpoint(ctx context.Context, forTable string) (*pbserumhist.Checkpoint, error) {
	var result *pbserumhist.Checkpoint

	queryFunc := func(ctx context.Context) error {
		queryString := fmt.Sprintf(`SELECT file,timestamp FROM %s WHERE table="%s" ORDER BY timestamp DESC LIMIT 1`, bql.checkpointsTable, forTable)

		q := bql.client.Query(queryString)
		j, err := q.Run(ctx)
		if err != nil {
			return fmt.Errorf("could not run query `%s`: %w", queryString, err)
		}
		it, err := j.Read(ctx)
		if err != nil {
			return fmt.Errorf("could not read query results: %w", err)
		}

		type Row struct {
			File      string    `bigquery:"file"`
			Timestamp time.Time `bigquery:"timestamp"`
		}

		for {
			var row Row
			err := it.Next(&row)
			if err == iterator.Done {
				return nil
			}
			if err != nil {
				return fmt.Errorf("could not read row: %w", err)
			}

			fileInfo, err := parseLatestInfoFromFilePath(row.File)
			if err != nil {
				return fmt.Errorf("could not parse file name: %w", err)
			}

			result = &pbserumhist.Checkpoint{
				LastWrittenSlotNum: fileInfo.LatestSlotNum,
				LastWrittenSlotId:  fileInfo.LatestSlotId,
			}
		}
	}

	err := derr.RetryContext(ctx, 5, queryFunc)
	if err != nil {
		return nil, err
	}

	return result, nil
}
