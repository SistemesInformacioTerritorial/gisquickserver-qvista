# Gisquick application server (backend)

```
docker build -t gisquick/server-dev -f ./docker/Dockerfile.dev .
```

```
docker build -t gisquick/server -f ./docker/Dockerfile-alpine .


# Gisquick for Windows

go build -o gisquick_8080_5433_env_nomask_lc_allwin_v2.exe -ldflags="-s -w" cmd\main.go
```
## conf

cfg := struct {
		Gisquick struct {
			//Debug                bool   `conf:"default:false"`
			Debug        bool   `conf:"default:true"`
			Language     string `conf:"default:en-us"`
			ProjectsRoot string `conf:"default:c:/gisquick/publish"`
			MapCacheRoot string
			//MapserverURL string `conf:"default:http://localhost:8080/qgis-server"`

			//jfs gisquick windowns
			MapserverURL string `conf:"default:http://localhost:8080/cgi-bin/qgis_mapserv.fcgi.exe"`

			PluginsURL           string
			SignupAPI            bool
			ProjectSizeLimit     ByteSize `conf:"default:-1"`
			AccountStorageLimit  ByteSize `conf:"default:-1"`
			AccountProjectsLimit int      `conf:"default:-1"`
			AccountLimiterConfig string
			LandingProject       string
			ProjectCustomization bool
			Extensions           string
		}
		Auth struct {
			SessionExpiration    time.Duration `conf:"default:24h"`
			EmailTokenExpiration time.Duration `conf:"default:72h"`
			SecretKey            string        `conf:"default:secret-key,mask"`
		}
		Web struct {
			ReadTimeout     time.Duration `conf:"default:5s"`
			WriteTimeout    time.Duration `conf:"default:10s"`
			IdleTimeout     time.Duration `conf:"default:120s"`
			ShutdownTimeout time.Duration `conf:"default:20s"`
			SiteURL         string        `conf:"default:http://127.0.0.1"`
			APIHost         string        `conf:"default:0.0.0.0:3000"`
		}
		Postgres struct {
			User string `conf:"default:postgres"`
			//Password           string `conf:"default:nexus,mask"`
			Password string `conf:"default:nexus"` // trec la mask perque es pogui veure el password al arrencar
			Host     string `conf:"default:localhost"`
			Name     string `conf:"default:postgres,env:GISQUICK_POSTGRES_DB"`
			//Name               string `conf:"default:postgres"`
			Port               int    `conf:"default:5433"`
			MaxIdleConns       int    `conf:"default:3"`
			MaxOpenConns       int    `conf:"default:3"`
			SSLMode            string `conf:"default:disable"`
			StatementCacheMode string `conf:"default:prepare"`
		}
		Redis struct {
			Network string // "unix"
			//Addr string `conf:"default:redis:6379"` // "/var/run/redis/redis.sock"
			Addr string `conf:"localhost:6379"`
			//Addr string `localhost:6379` // localhost:6379

			Password string `conf:"mask"`
			DB       int    `conf:"default:0"`
		}
		
	}{}

    cfg := struct {
		Gisquick struct {
			//Debug                bool   `conf:"default:false"`
			Debug        bool   `conf:"default:true"`
			Language     string `conf:"default:en-us"`
			ProjectsRoot string `conf:"default:c:/gisquick/publish"`
			MapCacheRoot string
			//MapserverURL string `conf:"default:http://localhost:8080/qgis-server"`

			//jfs gisquick windowns
			MapserverURL string `conf:"default:http://localhost:8080/cgi-bin/qgis_mapserv.fcgi.exe"`

			PluginsURL           string
			SignupAPI            bool
			ProjectSizeLimit     ByteSize `conf:"default:-1"`
			AccountStorageLimit  ByteSize `conf:"default:-1"`
			AccountProjectsLimit int      `conf:"default:-1"`
			AccountLimiterConfig string
			LandingProject       string
			ProjectCustomization bool
			Extensions           string
		}
		Auth struct {
			SessionExpiration    time.Duration `conf:"default:24h"`
			EmailTokenExpiration time.Duration `conf:"default:72h"`
			SecretKey            string        `conf:"default:secret-key,mask"`
		}
		Web struct {
			ReadTimeout     time.Duration `conf:"default:5s"`
			WriteTimeout    time.Duration `conf:"default:10s"`
			IdleTimeout     time.Duration `conf:"default:120s"`
			ShutdownTimeout time.Duration `conf:"default:20s"`
			SiteURL         string        `conf:"default:http://127.0.0.1"`
			APIHost         string        `conf:"default:0.0.0.0:4000"`
		}
		Postgres struct {
			User string `conf:"default:postgres"`
			//Password           string `conf:"default:nexus,mask"`
			Password string `conf:"default:nexus"` // trec la mask perque es pogui veure el password al arrencar
			Host     string `conf:"default:localhost"`
			//Name     string `conf:"default:postgres,env:GISQUICK_POSTGRES_DB"`
			//	Name               string `conf:"default:postgres,env:POSTGRES_DB"`
			Name string `conf:"default:pre,env:POSTGRES_DB"`
			//Name               string `conf:"default:postgres"`
			Port               int    `conf:"default:5433"`
			MaxIdleConns       int    `conf:"default:3"`
			MaxOpenConns       int    `conf:"default:3"`
			SSLMode            string `conf:"default:disable"`
			StatementCacheMode string `conf:"default:prepare"`
		}
		Redis struct {
			Network string // "unix"
			//Addr string `conf:"default:redis:6379"` // "/var/run/redis/redis.sock"
			Addr string `conf:"localhost:6379"`
			//Addr string `localhost:6379` // localhost:6379

			Password string `conf:"mask"`
			DB       int    `conf:"default:0"`
		}