package errors

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/opencontainers/go-digest"
)

// https://github.com/opencontainers/distribution-spec/blob/main/spec.md#error-codes
// | ID      | Code                    | Description                                                |
// |-------- | ------------------------|------------------------------------------------------------|
// | code-1  | `BLOB_UNKNOWN`          | blob unknown to registry                                   |
// | code-2  | `BLOB_UPLOAD_INVALID`   | blob upload invalid                                        |
// | code-3  | `BLOB_UPLOAD_UNKNOWN`   | blob upload unknown to registry                            |
// | code-4  | `DIGEST_INVALID`        | provided digest did not match uploaded content             |
// | code-5  | `MANIFEST_BLOB_UNKNOWN` | manifest references a manifest or blob unknown to registry |
// | code-6  | `MANIFEST_INVALID`      | manifest invalid                                           |
// | code-7  | `MANIFEST_UNKNOWN`      | manifest unknown to registry                               |
// | code-8  | `NAME_INVALID`          | invalid repository name                                    |
// | code-9  | `NAME_UNKNOWN`          | repository name not known to registry                      |
// | code-10 | `SIZE_INVALID`          | provided length did not match content length               |
// | code-12 | `UNAUTHORIZED`          | authentication required                                    |
// | code-13 | `DENIED`                | requested access to the resource is denied                 |
// | code-14 | `UNSUPPORTED`           | the operation is unsupported                               |
// | code-15 | `TOOMANYREQUESTS`       | too many requests                                          |
const (
	ErrCodeBlobUnknown         ErrCode = "BLOB_UNKNOWN"
	ErrCodeBlobUploadInvalid   ErrCode = "BLOB_UPLOAD_INVALID"
	ErrCodeBlobUploadUnknown   ErrCode = "BLOB_UPLOAD_UNKNOWN"
	ErrCodeDigestInvalid       ErrCode = "DIGEST_INVALID"
	ErrCodeManifestBlobUnknown ErrCode = "MANIFEST_BLOB_UNKNOWN"
	ErrCodeManifestInvalid     ErrCode = "MANIFEST_INVALID"
	ErrCodeManifestUnknown     ErrCode = "MANIFEST_UNKNOWN"
	ErrCodeNameInvalid         ErrCode = "NAME_INVALID"
	ErrCodeNameUnknown         ErrCode = "NAME_UNKNOWN"
	ErrCodeSizeInvalid         ErrCode = "SIZE_INVALID"
	ErrCodeUnauthorized        ErrCode = "UNAUTHORIZED"
	ErrCodeDenied              ErrCode = "DENIED"
	ErrCodeUnsupported         ErrCode = "UNSUPPORTED"
	ErrCodeTooManyRequests     ErrCode = "TOOMANYREQUESTS"
	ErrCodeConfigInvalid       ErrCode = "CONFIG_INVALID"
	ErrCodeInvalidParameter    ErrCode = "INVALID_PARAMETER"
	ErrCodeIndexUnknown        ErrCode = "INDEX_UNKNOWN"
	ErrCodeUnknow              ErrCode = "UNKNOWN"
	ErrCodeInternal            ErrCode = "INTERNAL"
)

type ErrCode string

type ErrorInfo struct {
	HttpStatus int     `json:"-"`
	Code       ErrCode `json:"code"`
	Message    string  `json:"message"`
	Detail     string  `json:"detail"`
}

func (e ErrorInfo) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func IsErrCode(err error, code ErrCode) bool {
	if err == nil {
		return false
	}
	info := ErrorInfo{}
	if errors.As(err, &info) {
		return info.Code == code
	}
	return false
}

func NewAuthFailedError(err error) ErrorInfo {
	return ErrorInfo{
		HttpStatus: http.StatusUnauthorized,
		Code:       ErrCodeUnauthorized,
		Message:    err.Error(),
	}
}

func NewUnsupportedError(msg string) ErrorInfo {
	return ErrorInfo{HttpStatus: http.StatusNotImplemented, Code: ErrCodeUnsupported, Message: msg}
}

func NewInternalError(err error) ErrorInfo {
	return ErrorInfo{HttpStatus: http.StatusInternalServerError, Code: ErrCodeInternal, Message: err.Error()}
}

func NewDigestInvalidError(got string) ErrorInfo {
	return ErrorInfo{HttpStatus: http.StatusBadRequest, Code: ErrCodeDigestInvalid, Message: fmt.Sprintf("digest invalid: %s", got)}
}

func NewIndexUnknownError(repository string) ErrorInfo {
	return ErrorInfo{HttpStatus: http.StatusNotFound, Code: ErrCodeIndexUnknown, Message: fmt.Sprintf("index: %s not found", repository)}
}

func NewBlobUnknownError(digest digest.Digest) ErrorInfo {
	return ErrorInfo{HttpStatus: http.StatusNotFound, Code: ErrCodeBlobUnknown, Message: fmt.Sprintf("blob: %s not found", digest.String())}
}

func NewManifestUnknownError(reference string) ErrorInfo {
	return ErrorInfo{HttpStatus: http.StatusNotFound, Code: ErrCodeManifestUnknown, Message: fmt.Sprintf("manifest: %s not found", reference)}
}

func NewManifestInvalidError(err error) ErrorInfo {
	return ErrorInfo{HttpStatus: http.StatusBadRequest, Code: ErrCodeManifestInvalid, Message: err.Error()}
}

func NewContentTypeInvalidError(got string) ErrorInfo {
	return ErrorInfo{HttpStatus: http.StatusBadRequest, Code: ErrCodeInvalidParameter, Message: fmt.Sprintf("content type invalid: %s", got)}
}

func NewContentRangeInvalidError(msg string) ErrorInfo {
	return ErrorInfo{HttpStatus: http.StatusBadRequest, Code: ErrCodeSizeInvalid, Message: fmt.Sprintf("content range: %s", msg)}
}

func NewContentLengthInvalidError(msg string) ErrorInfo {
	return ErrorInfo{HttpStatus: http.StatusBadRequest, Code: ErrCodeSizeInvalid, Message: fmt.Sprintf("content length: %s", msg)}
}

func NewConfigInvalidError(msg string) ErrorInfo {
	return ErrorInfo{HttpStatus: http.StatusBadRequest, Code: ErrCodeConfigInvalid, Message: msg}
}

func NewParameterInvalidError(msg string) ErrorInfo {
	return ErrorInfo{HttpStatus: http.StatusBadRequest, Code: ErrCodeInvalidParameter, Message: msg}
}
