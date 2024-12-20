package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/G-Research/unicorn-history-server/internal/database/sql"
	"github.com/G-Research/unicorn-history-server/internal/model"
)

type HistoryFilters struct {
	TimestampStart *time.Time
	TimestampEnd   *time.Time
	Offset         *int
	Limit          *int
}

func applyHistoryFilters(builder *sql.Builder, filters HistoryFilters) {
	if filters.TimestampStart != nil {
		builder.Conditionp("timestamp", ">=", filters.TimestampStart.UnixNano())
	}
	if filters.TimestampEnd != nil {
		builder.Conditionp("timestamp", "<=", filters.TimestampEnd.UnixNano())
	}
	applyLimitAndOffset(builder, filters.Limit, filters.Offset)
}

func (r *PostgresRepository) InsertAppHistory(ctx context.Context, appHistory *model.AppHistory) error {
	const appHistoryType = "application"
	const q = `
INSERT INTO history (
	 id, 
	 created_at_nano,
	 deleted_at_nano,
	 history_type, 
	 total_number, 
	 timestamp
) VALUES (
	@id,
	@created_at_nano,
	@deleted_at_nano,
	@history_type,
	@total_number,
	@timestamp
)`

	_, err := r.dbpool.Exec(ctx, q,
		pgx.NamedArgs{
			"id":              appHistory.ID,
			"created_at_nano": appHistory.CreatedAtNano,
			"deleted_at_nano": appHistory.DeletedAtNano,
			"history_type":    appHistoryType,
			"total_number":    appHistory.TotalApplications,
			"timestamp":       appHistory.Timestamp,
		})
	if err != nil {
		return fmt.Errorf("could not create application history into DB: %v", err)
	}
	return nil
}

func (r *PostgresRepository) InsertContainerHistory(ctx context.Context, containerHistory *model.ContainerHistory) error {
	const containerHistoryType = "container"
	const q = `
INSERT INTO history (
	id,
	 created_at_nano,
	 deleted_at_nano,
	history_type,
	total_number,
	timestamp
) VALUES (
	 @id,
	 @created_at_nano,
	 @deleted_at_nano,
	 @history_type,
	 @total_number,
	 @timestamp
)`

	_, err := r.dbpool.Exec(ctx, q,
		pgx.NamedArgs{
			"id":              containerHistory.ID,
			"created_at_nano": containerHistory.CreatedAtNano,
			"deleted_at_nano": containerHistory.DeletedAtNano,
			"history_type":    containerHistoryType,
			"total_number":    containerHistory.TotalContainers,
			"timestamp":       containerHistory.Timestamp,
		})
	if err != nil {
		return fmt.Errorf("could not create container history into DB: %v", err)
	}
	return nil

}

func (r *PostgresRepository) GetApplicationsHistory(ctx context.Context, filters HistoryFilters) ([]*model.AppHistory, error) {
	queryBuilder := sql.NewBuilder().
		SelectAll("history", "").
		Conditionp("history_type", "=", "application").
		OrderBy("timestamp", sql.OrderByDescending)
	applyHistoryFilters(queryBuilder, filters)

	var apps []*model.AppHistory

	query := queryBuilder.Query()
	args := queryBuilder.Args()
	rows, err := r.dbpool.Query(ctx, query, args...)

	if err != nil {
		return nil, fmt.Errorf("could not get applications history from DB: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var app model.AppHistory
		err := rows.Scan(&app.ID, &app.CreatedAtNano, &app.DeletedAtNano, nil, &app.TotalApplications, &app.Timestamp)
		if err != nil {
			return nil, fmt.Errorf("could not scan applications history from DB: %v", err)
		}
		apps = append(apps, &app)
	}
	return apps, nil
}

func (r *PostgresRepository) GetContainersHistory(ctx context.Context, filters HistoryFilters) ([]*model.ContainerHistory, error) {
	queryBuilder := sql.NewBuilder().
		SelectAll("history", "").
		Conditionp("history_type", "=", "container").
		OrderBy("timestamp", sql.OrderByDescending)
	applyHistoryFilters(queryBuilder, filters)

	var containers []*model.ContainerHistory

	query := queryBuilder.Query()
	args := queryBuilder.Args()
	rows, err := r.dbpool.Query(ctx, query, args...)

	if err != nil {
		return nil, fmt.Errorf("could not get container history from DB: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var container model.ContainerHistory
		err := rows.Scan(&container.ID, &container.CreatedAtNano, &container.DeletedAtNano, nil, &container.TotalContainers, &container.Timestamp)
		if err != nil {
			return nil, fmt.Errorf("could not scan contaienrs history from DB: %v", err)
		}
		containers = append(containers, &container)
	}
	return containers, nil
}
