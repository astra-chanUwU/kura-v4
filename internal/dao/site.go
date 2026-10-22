package dao

import (
	"context"
	"database/sql"

	"github.com/astra-chanUwU/kura-v4/internal/dao/sqlcgen"
	"github.com/astra-chanUwU/kura-v4/internal/model"
)

type SiteDAO struct{ queries *sqlcgen.Queries }

func NewSite(conn *sql.DB) *SiteDAO { return &SiteDAO{queries: sqlcgen.New(conn)} }
func (d *SiteDAO) Get(ctx context.Context) (model.Site, error) {
	row, err := d.queries.GetSite(ctx)
	if err != nil {
		return model.Site{}, err
	}
	return model.Site{Name: row.Name, Description: row.Description}, nil
}
