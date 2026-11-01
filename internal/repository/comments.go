package repository

import "github.com/astra-chanUwU/kura-v4/internal/dao"

type Comments struct{ *dao.CommentsDAO }

func NewComments(d *dao.CommentsDAO) *Comments { return &Comments{CommentsDAO: d} }
