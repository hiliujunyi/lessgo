package middleware

import (
	"fmt"
	"net/http"
	"path"

	"github.com/lessgo/lessgo"
)

type (
	// StaticConfig defines the config for static middleware.
	StaticConfig struct {
		// Root is the directory from where the static content is served.
		// Optional with default value as ``.
		Root string `json:"root"`

		// Index is the list of index files to be searched and used when serving
		// a directory.
		// Optional with default value as []string{"index.html"}.
		Index []string `json:"index"`

		// Browse is a flag to enable/disable directory browsing.
		// Optional with default value as false.
		Browse bool `json:"browse"`
	}
)

var (
	// DefaultStaticConfig is the default static middleware config.
	DefaultStaticConfig = StaticConfig{
		Root:   "",
		Index:  []string{"index.html"},
		Browse: false,
	}
)

// Static returns a static middleware to serves static content from the provided
// root directory.
func Static(root string) lessgo.MiddlewareFunc {
	c := DefaultStaticConfig
	c.Root = root
	return StaticFromConfig(c)
}

// StaticFromConfig returns a static middleware from config.
// See `Static()`.
func StaticFromConfig(config StaticConfig) lessgo.MiddlewareFunc {
	// Defaults
	if config.Index == nil {
		config.Index = DefaultStaticConfig.Index
	}

	return func(next lessgo.HandlerFunc) lessgo.HandlerFunc {
		return func(c lessgo.Context) error {
			fs := http.Dir(config.Root)
			p := c.Request().URL().Path()
			if c.P(0) != "" { // If serving from `Group`, e.g. `/static*`
				p = c.P(0)
			}
			file := path.Clean(p)
			f, err := fs.Open(file)
			if err != nil {
				return next(c)
			}
			defer f.Close()

			fi, err := f.Stat()
			if err != nil {
				return err
			}

			if fi.IsDir() {
				/* NOTE:
				Not checking the Last-Modified header as it caches the response `304` when
				changing different directories for the same path.
				*/
				d := f

				// Index file
				// TODO: search all files
				file = path.Join(file, config.Index[0])
				f, err = fs.Open(file)
				if err != nil {
					if config.Browse {
						dirs, err := d.Readdir(-1)
						if err != nil {
							return err
						}

						// Create a directory index
						rs := c.Response()
						rs.Header().Set(lessgo.HeaderContentType, lessgo.MIMETextHTMLCharsetUTF8)
						if _, err = fmt.Fprintf(rs, "<pre>\n"); err != nil {
							return err
						}
						for _, d := range dirs {
							name := d.Name()
							color := "#212121"
							if d.IsDir() {
								color = "#e91e63"
								name += "/"
							}
							if _, err = fmt.Fprintf(rs, "<a href=\"%s\" style=\"color: %s;\">%s</a>\n", name, color, name); err != nil {
								return err
							}
						}
						_, err = fmt.Fprintf(rs, "</pre>\n")
						return err
					}
					return next(c)
				}
				fi, _ = f.Stat() // Index file stat
			}
			return c.ServeContent(f, fi.Name(), fi.ModTime())
		}
	}
}
