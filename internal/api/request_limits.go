package api

import (
	"errors"
	"mime/multipart"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/photoprism/photoprism/pkg/i18n"
)

const (
	maxSessionRequestBytes       int64 = 64 * 1024
	maxUploadOptionsRequestBytes int64 = 256 * 1024
	maxMultipartOverheadBytes    int64 = 1024 * 1024
	maxAvatarUploadBytes         int64 = 20000000 + maxMultipartOverheadBytes
)

// limitRequestBodyBytes caps the readable request body size for the current handler.
func limitRequestBodyBytes(c *gin.Context, limit int64) {
	if c == nil || c.Request == nil || limit <= 0 {
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
}

// isRequestBodyTooLarge reports whether the parsing error was caused by a body-size limit.
func isRequestBodyTooLarge(err error) bool {
	if err == nil {
		return false
	}

	var maxBytesErr *http.MaxBytesError

	return errors.As(err, &maxBytesErr) || errors.Is(err, multipart.ErrMessageTooLarge)
}

// abortRequestTooLarge writes a localized 413 response and stops the current request.
func abortRequestTooLarge(c *gin.Context, id i18n.Message) {
	Abort(c, http.StatusRequestEntityTooLarge, id)
}
