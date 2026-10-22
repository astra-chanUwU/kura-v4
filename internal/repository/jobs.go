package repository

import "github.com/astra-chanUwU/kura-v4/internal/dao"

type Jobs struct{ *dao.JobDAO }

func NewJobs(d *dao.JobDAO) *Jobs { return &Jobs{JobDAO: d} }
