package crawl

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/thatguystone/cog/check"
)

func TestServeFileFallback(t *testing.T) {
	c := check.New(t)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	ServeFile(rr, req, "./response_test.go")

	c.Equal(rr.Code, http.StatusOK)
	c.Contains(rr.Body.String(), "TestServeFileFallback")
}
