package server

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/gisquick/gisquick-server/internal/infrastructure/cache"
	"github.com/jellydator/ttlcache/v3"
	"github.com/labstack/echo/v4"
)

type Plugins struct {
	XMLName xml.Name       `xml:"plugins"`
	Plugins []PyQgisPlugin `xml:"pyqgis_plugin"`
}

type PyQgisPlugin struct {
	XMLName        xml.Name `xml:"pyqgis_plugin"`
	Name           string   `xml:"name,attr" json:"name"`
	QgisMinVersion string   `xml:"qgis_minimum_version" json:"qgisMinimumVersion"`
	QgisMaxVersion string   `xml:"qgis_maximum_version,omitempty" json:"qgisMaximumVersion"`
	Description    CDATA    `xml:"description" json:"description"`
	About          CDATA    `xml:"about" json:"about"`
	Version        string   `xml:"version,attr" json:"version"`
	Author         string   `xml:"author_name" json:"author"`
	// Email
	Changelog    CDATA  `xml:"changelog" json:"changelog"`
	Experimental string `xml:"experimental" json:"experimental"`
	Deprecated   string `xml:"deprecated" json:"deprecated"`
	Tags         string `xml:"tags" json:"tags"`
	Homepage     CDATA  `xml:"homepage" json:"homepage"`
	Repository   CDATA  `xml:"repository" json:"repository"`
	Tracker      CDATA  `xml:"tracker" json:"tracker"`
	Icon         string `xml:"icon,omitempty" json:"icon,omitempty"`
	Server       string `xml:"server" json:"server"`
	// Category string `json:"category"` // one of Raster, Vector, Database and Web

	// not part of qgis metadata
	Updated time.Time `xml:"update_date" json:"updated"`

	// generated data
	DownloadURL string `xml:"download_url"`
	FileName    string `xml:"file_name" json:"filename"`
}

// type CDATA struct {
// 	Text string `xml:",cdata"`
// }

type CDATA string

func (c CDATA) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return e.EncodeElement(struct {
		string `xml:",cdata"`
	}{string(c)}, start)
}

func (s *Server) handleDownloadPlugin(rootDir string) func(echo.Context) error {
	return func(c echo.Context) error {
		filename := filepath.Clean(strings.TrimPrefix(c.Param("*"), "/"))
		fpath := filepath.Join(rootDir, filename)
		relPath, err := filepath.Rel(rootDir, fpath)
		if err != nil || strings.HasPrefix(relPath, "..") {
			return echo.ErrForbidden
		}
		return c.File(fpath)
	}
}

func (s *Server) pluginBaseURL(c echo.Context) (*url.URL, error) {
	baseURL := strings.TrimSpace(s.Config.PluginsURL)
	if baseURL == "" {
		scheme := c.Scheme()
		if forwardedProto := c.Request().Header.Get("X-Forwarded-Proto"); forwardedProto != "" {
			scheme = forwardedProto
		}
		baseURL = fmt.Sprintf("%s://%s/plugins", scheme, c.Request().Host)
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing plugins url: %w", err)
	}
	return u, nil
}

func (s *Server) platformPluginRepoHandler1(rootDir string) func(echo.Context) error {
	type PluginKey struct {
		Filename string
		Mtime    int64
	}
	type PluginEntry struct {
		Data  PyQgisPlugin
		Mtime int64
	}
	loader := ttlcache.LoaderFunc[PluginKey, PluginEntry](
		func(c *ttlcache.Cache[PluginKey, PluginEntry], key PluginKey) *ttlcache.Item[PluginKey, PluginEntry] {
			s.log.Infof("platformPluginRepoHandler.LoaderFunc: %s", key)
			// var plugin PyQgisPlugin2
			// f, err := os.Open(filename)
			// if err != nil {
			// 	s.log.Errorw("reading qgis plugin metadata", zap.Error(err))
			// 	return nil
			// }
			// if err := json.NewDecoder(f).Decode(&plugin); err != nil {
			// 	s.log.Errorw("parsing qgis plugin metadata", zap.Error(err))
			// 	return nil
			// }
			// entry = PluginEntry{plugin}
			// item := c.Set(filename, plugin, ttlcache.NoTTL)
			// return item
			return nil
		},
	)
	cache := ttlcache.New(
		ttlcache.WithLoader[PluginKey, PluginEntry](loader),
	)
	cache.Metrics()
	return func(c echo.Context) error {
		return nil
	}
}

func (s *Server) platformPluginRepoHandler(rootDir string) func(echo.Context) error {
	cache := cache.NewDataCache(func(filename string) (PyQgisPlugin, error) {
		var plugin PyQgisPlugin
		s.log.Infow("loading qgis plugin metadata", "file", filename)
		f, err := os.Open(filename)
		if err != nil {
			// s.log.Errorw("reading qgis plugin metadata", zap.Error(err))
			return plugin, fmt.Errorf("reading qgis plugin metadata: %w", err)
		}
		defer f.Close()
		if err := json.NewDecoder(f).Decode(&plugin); err != nil {
			// s.log.Errorw("parsing qgis plugin metadata", zap.Error(err))
			return plugin, fmt.Errorf("parsing qgis plugin metadata: %w", err)
		}
		return plugin, nil
	})

	return func(c echo.Context) error {
		platform := c.Param("platform")
		baseURL, err := s.pluginBaseURL(c)
		if err != nil {
			return fmt.Errorf("building plugins base url: %w", err)
		}

		files, err := filepath.Glob(filepath.Join(rootDir, platform, "*/*.json"))
		if err != nil {
			return fmt.Errorf("listing qgis plugins repo: %w", err)
		}
		plugins := make([]PyQgisPlugin, 0, len(files))
		for _, filename := range files {
			fStat, err := os.Stat(filename)
			if err != nil {
				return fmt.Errorf("listing qgis plugins repo: %w", err)
				// plugin.Updated = fStat.ModTime()
			}
			updated := fStat.ModTime()
			timestamp := updated.Unix()

			plugin, err := cache.Get(filename, timestamp)
			if err != nil {
				return fmt.Errorf("getting qgis plugin metadata: %w", err)
			}
			pluginName := filepath.Base(filepath.Dir(filename))
			plugin.Updated = updated.UTC()

			downloadURL := *baseURL
			downloadURL.Path = path.Join(downloadURL.Path, "download", platform, pluginName, plugin.FileName)
			plugin.DownloadURL = downloadURL.String()

			if plugin.Icon != "" {
				iconURL := *baseURL
				iconURL.Path = path.Join(iconURL.Path, "download", platform, pluginName, plugin.Icon)
				plugin.Icon = iconURL.String()
			}

			// s.log.Infow("plugin metadata", "meta", plugin)
			plugins = append(plugins, plugin)
		}
		return c.XML(http.StatusOK, Plugins{Plugins: plugins})
	}
}
