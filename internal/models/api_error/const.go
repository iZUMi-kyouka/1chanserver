package api_error

import (
	"errors"
	"net/http"
)

var (
	MissingPageReq = NewC(errors.New("missing page request"), http.StatusBadRequest)
	InvalidPageReq = NewC(errors.New("invalid page request"), http.StatusBadRequest)
)
