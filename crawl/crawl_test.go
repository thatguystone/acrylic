package crawl

import (
	"net/http"
	"testing"

	"github.com/thatguystone/cog/check"
)

func TestCrawlInvalidURLs(t *testing.T) {
	c := check.New(t)

	ns := newTestNS(c, nil)
	defer ns.clean()

	tests := []struct {
		name    string
		errPath string
		cfg     Config
	}{
		{
			name:    "InvalidEntry",
			errPath: "/",
			cfg: Config{
				Handler: http.HandlerFunc(http.NotFound),
				Entries: []string{"://"},
				Output:  ns.path("/public"),
			},
		},
		{
			name:    "InvalidHref",
			errPath: "/",
			cfg: Config{
				Handler: mux(map[string]http.Handler{
					"/": stringHandler{
						contType: htmlType,
						body:     `<a href="://"></a>`,
					},
				}),
				Entries: []string{"/"},
				Output:  ns.path("/public"),
			},
		},
	}

	for _, test := range tests {
		test := test

		c.Run(test.name, func(c *check.C) {
			_, err := Crawl(test.cfg)
			c.Log(err)
			c.Contains(err, test.errPath)
		})
	}
}
