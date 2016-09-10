package crawl

import (
	"mime"
	"net/http"
	"time"
)

// Response provides additional information about the wrapped response
type response struct {
	*http.Response
	typ     contentType
	lastMod time.Time
}

func wrapResponse(resp *http.Response, state *state) *response {
	wResp := &response{
		Response: resp,
	}

	wResp.updateLastModified(state)
	wResp.updateContentType(state)

	return wResp
}

func (resp *response) updateLastModified(state *state) {
	lastMod := resp.Header.Get("Last-Modified")
	if lastMod == "" {
		return
	}

	t, err := time.Parse(http.TimeFormat, lastMod)
	if err != nil {
		state.Logf("W: [http resp] "+
			"invalid Last-Modified from %s: %v",
			resp.Request.URL, err)
		return
	}

	resp.lastMod = t
}

func (resp *response) updateContentType(state *state) {
	mediaType := resp.Header.Get("Content-Type")

	state.Logf("[resp] %s", mediaType)

	if mediaType != "" {
		var err error
		mediaType, _, err = mime.ParseMediaType(mediaType)
		if err != nil {
			state.Errorf("[http resp] "+
				"invalid content type from %s: %v",
				resp.Request.URL, err)
			return
		}
	}

	resp.typ = contentTypeFromMime(mediaType)
}
