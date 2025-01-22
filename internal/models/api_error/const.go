package api_error

import (
	"errors"
	"net/http"
)

var (
	MissingPageReq = NewFromErr(errors.New("missing page request"), http.StatusBadRequest)
	InvalidPageReq = NewFromErr(errors.New("invalid page request"), http.StatusBadRequest)
)
