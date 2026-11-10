package repository

import "github.com/astra-chanUwU/kura-v4/internal/dao"

type Reports struct{ *dao.ReportsDAO }

func NewReports(d *dao.ReportsDAO) *Reports { return &Reports{ReportsDAO: d} }
