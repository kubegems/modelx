package registry

import (
	"encoding/json"
	"errors"
	"net/http"

	apierr "kubegems.io/modelx/pkg/errors"
)

func ResponseError(w http.ResponseWriter, err error) {
	info := apierr.ErrorInfo{}
	if !errors.As(err, &info) {
		info = apierr.ErrorInfo{
			HttpStatus: http.StatusBadRequest,
			Code:       apierr.ErrCodeUnknow,
			Message:    err.Error(),
			Detail:     err.Error(),
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(info.HttpStatus)
	json.NewEncoder(w).Encode(info)
}

func ResponseOK(w http.ResponseWriter, data any) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(data)
}
