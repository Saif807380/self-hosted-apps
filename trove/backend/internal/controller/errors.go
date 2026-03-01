package controller

import (
	"errors"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
)

func connectErr(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}
