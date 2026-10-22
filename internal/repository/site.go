package repository

import "github.com/astra-chanUwU/kura-v4/internal/dao"

// Promoted DAO methods avoid forwarding boilerplate. Composite operations and
// transactions belong here when a feature needs them.
type Site struct{ *dao.SiteDAO }

func NewSite(d *dao.SiteDAO) *Site { return &Site{SiteDAO: d} }
