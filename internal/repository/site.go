package repository

import (
	"context"

	"github.com/astra-chanUwU/kura-v4/internal/dao"
	"github.com/astra-chanUwU/kura-v4/internal/model"
)

// Site is the concrete store behind site.Repository.
type Site struct {
	dao *dao.SiteDAO
}

func NewSite(d *dao.SiteDAO) *Site { return &Site{dao: d} }

func (r *Site) Get(ctx context.Context) (model.Site, error) {
	return r.dao.Get(ctx)
}
