package commands

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gisquick/gisquick-server/internal/application"
	"github.com/gisquick/gisquick-server/internal/domain"
	"github.com/gisquick/gisquick-server/internal/infrastructure/email"
	"github.com/gisquick/gisquick-server/internal/infrastructure/postgres"
	"github.com/gisquick/gisquick-server/internal/infrastructure/project"
	"github.com/gisquick/gisquick-server/internal/infrastructure/security"
	"github.com/gisquick/gisquick-server/internal/infrastructure/ws"
	"github.com/gisquick/gisquick-server/internal/server"
	"github.com/gisquick/gisquick-server/internal/server/auth"
	"github.com/go-redis/redis/v8"
	"github.com/spf13/viper"
	mail "github.com/xhit/go-simple-mail/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type FileLoggingConfig struct {
	Enabled    bool
	Path       string
	MaxSizeMB  int
	MaxBackups int
	MaxAgeDays int
	Compress   bool
}

func parseLogLevel(value string, fallback zapcore.Level) zapcore.Level {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return fallback
	}
	switch value {
	case "debug":
		return zap.DebugLevel
	case "info":
		return zap.InfoLevel
	case "warn", "warning":
		return zap.WarnLevel
	case "error":
		return zap.ErrorLevel
	default:
		return fallback
	}
}

func parseByteSize(value string) (int64, error) {
	value = strings.TrimSpace(value)
	factor := 1
	if strings.HasSuffix(value, "M") {
		factor = 1024 * 1024
	} else if strings.HasSuffix(value, "G") {
		factor = 1024 * 1024 * 1024
	}
	num, err := strconv.Atoi(strings.TrimRight(value, "MGB"))
	if err != nil {
		return -1, fmt.Errorf("Invalid byte size: %s", value)
	}
	return int64(num * factor), nil
}

type ByteSize int64

// Satisfy the flag package Value interface.
func (b *ByteSize) Set(s string) error {
	bs, err := parseByteSize(s)
	if err != nil {
		return err
	}
	*b = ByteSize(bs)
	return nil
}

// Satisfy the encoding.TextUnmarshaler interface.
func (b *ByteSize) UnmarshalText(text []byte) error {
	return b.Set(string(text))
}

// Funció per comprovar i mostrar canvis respecte als valors per defecte
func checkConfigChanges(log *zap.SugaredLogger, v *viper.Viper) {
	// Mapa de valors per defecte (copiat de les definicions d'abans)
	defaults := map[string]interface{}{
		"gisquick.debug":            true,
		"gisquick.language":         "en-us",
		"gisquick.projectsRoot":     "c:/gisquick/publish",
		"gisquick.mapserverURL":     "http://localhost:8080/cgi-bin/qgis_mapserv.fcgi.exe",
		"auth.sessionExpiration":    "24h",
		"auth.emailTokenExpiration": "72h",
		"auth.secretKey":            "secret-key",
		"web.siteURL":               "http://127.0.0.1",
		"web.apiHost":               "0.0.0.0:3000",
		"postgres.user":             "postgres",
		"postgres.password":         "nexus",
		"postgres.host":             "localhost",
		"postgres.name":             "postgres",
		"postgres.port":             5433,
		"redis.addr":                "localhost:6379",
		"email.host":                "smtp.office365.com",
		"email.port":                587,
		"email.username":            "auth.smtp@nexusgeographics.com",
		"email.encryption":          "STARTTLS",
	}

	// Comprova cada clau
	for key, defaultValue := range defaults {
		if v.IsSet(key) {
			currentValue := v.Get(key)

			// Convertir a string per comparar
			defaultStr := fmt.Sprintf("%v", defaultValue)
			currentStr := fmt.Sprintf("%v", currentValue)

			if defaultStr != currentStr {
				log.Infow("configuració per defecte canviada",
					"paràmetre", key,
					"valor_per_defecte", defaultStr,
					"valor_actual", currentStr)
			}
		}
	}
}

func Serve() error {
	v := viper.New()

	// Set default values
	v.SetDefault("gisquick.debug", true)
	v.SetDefault("gisquick.language", "en-us")
	v.SetDefault("gisquick.projectsRoot", "c:/gisquick/publish")
	v.SetDefault("gisquick.mapserverURL", "http://localhost:8080/cgi-bin/qgis_mapserv.fcgi.exe")
	v.SetDefault("gisquick.projectSizeLimit", -1)
	v.SetDefault("gisquick.accountStorageLimit", -1)
	v.SetDefault("gisquick.accountProjectsLimit", -1)

	v.SetDefault("auth.sessionExpiration", "24h")
	v.SetDefault("auth.emailTokenExpiration", "72h")
	v.SetDefault("auth.secretKey", "secret-key")

	v.SetDefault("web.readTimeout", "5s")
	v.SetDefault("web.writeTimeout", "10s")
	v.SetDefault("web.idleTimeout", "120s")
	v.SetDefault("web.shutdownTimeout", "20s")
	v.SetDefault("web.siteURL", "http://127.0.0.1")
	v.SetDefault("web.apiHost", "0.0.0.0:3000")

	v.SetDefault("postgres.user", "postgres")
	v.SetDefault("postgres.password", "nexus")
	v.SetDefault("postgres.host", "localhost")
	v.SetDefault("postgres.name", "postgres")
	v.SetDefault("postgres.port", 5433)
	v.SetDefault("postgres.maxIdleConns", 3)
	v.SetDefault("postgres.maxOpenConns", 3)
	v.SetDefault("postgres.sslMode", "disable")
	v.SetDefault("postgres.statementCacheMode", "prepare")

	v.SetDefault("redis.addr", "localhost:6379")
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)

	v.SetDefault("email.host", "smtp.office365.com")
	v.SetDefault("email.port", 587)
	v.SetDefault("email.encryption", "STARTTLS")
	v.SetDefault("email.username", "auth.smtp@nexusgeographics.com")
	v.SetDefault("email.password", "9D1av%JcRVh")
	v.SetDefault("email.sender", "auth.smtp@nexusgeographics.com")
	v.SetDefault("email.activationSubject", "Gisquick Registration")
	v.SetDefault("email.passwordResetSubject", "Gisquick Password Reset")

	// Logging defaults
	v.SetDefault("logging.level", "")
	v.SetDefault("logging.file.enabled", true)
	v.SetDefault("logging.file.path", "c:/gisquick/logs/gisquick-server.log")
	v.SetDefault("logging.file.maxSizeMB", 100)
	v.SetDefault("logging.file.maxBackups", 10)
	v.SetDefault("logging.file.maxAgeDays", 30)
	v.SetDefault("logging.file.compress", false)

	// Configure file search
	configFile := os.Getenv("GISQUICK_CONFIG")
	if configFile != "" {
		v.SetConfigFile(configFile)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
		v.AddConfigPath("/etc/gisquick")
	}

	// Configure environment variables
	v.SetEnvPrefix("GISQUICK")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Attempt to read configuration file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("error reading configuration file: %w", err)
		}
	}

	// Create configuration structure
	cfg := struct {
		Gisquick struct {
			Debug                bool
			Language             string
			ProjectsRoot         string
			MapCacheRoot         string
			MapserverURL         string
			PluginsURL           string
			SignupAPI            bool
			ProjectSizeLimit     ByteSize
			AccountStorageLimit  ByteSize
			AccountProjectsLimit int
			AccountLimiterConfig string
			LandingProject       string
			ProjectCustomization bool
			Extensions           string
		}
		Auth struct {
			SessionExpiration    time.Duration
			EmailTokenExpiration time.Duration
			SecretKey            string
		}
		Web struct {
			ReadTimeout     time.Duration
			WriteTimeout    time.Duration
			IdleTimeout     time.Duration
			ShutdownTimeout time.Duration
			SiteURL         string
			APIHost         string
		}
		Postgres struct {
			User               string
			Password           string
			Host               string
			Name               string
			Port               int
			MaxIdleConns       int
			MaxOpenConns       int
			SSLMode            string
			StatementCacheMode string
		}
		Redis struct {
			Network  string
			Addr     string
			Password string
			DB       int
		}
		Email struct {
			Host                 string
			Port                 int
			Encryption           string
			Username             string
			Password             string
			Sender               string
			ActivationSubject    string
			PasswordResetSubject string
		}
		Logging struct {
			Level string
			File  FileLoggingConfig
		}
	}{}

	// Map Viper configuration to structure
	cfg.Gisquick.Debug = v.GetBool("gisquick.debug")
	cfg.Gisquick.Language = v.GetString("gisquick.language")
	cfg.Gisquick.ProjectsRoot = v.GetString("gisquick.projectsRoot")
	cfg.Gisquick.MapCacheRoot = v.GetString("gisquick.mapCacheRoot")
	cfg.Gisquick.MapserverURL = v.GetString("gisquick.mapserverURL")
	cfg.Gisquick.PluginsURL = v.GetString("gisquick.pluginsURL")
	cfg.Gisquick.SignupAPI = v.GetBool("gisquick.signupAPI")

	if psl := v.GetString("gisquick.projectSizeLimit"); psl != "" {
		var bs ByteSize
		if err := bs.Set(psl); err != nil {
			return fmt.Errorf("invalid projectSizeLimit: %w", err)
		}
		cfg.Gisquick.ProjectSizeLimit = bs
	}

	if asl := v.GetString("gisquick.accountStorageLimit"); asl != "" {
		var bs ByteSize
		if err := bs.Set(asl); err != nil {
			return fmt.Errorf("invalid accountStorageLimit: %w", err)
		}
		cfg.Gisquick.AccountStorageLimit = bs
	}

	cfg.Gisquick.AccountProjectsLimit = v.GetInt("gisquick.accountProjectsLimit")
	cfg.Gisquick.AccountLimiterConfig = v.GetString("gisquick.accountLimiterConfig")
	cfg.Gisquick.LandingProject = v.GetString("gisquick.landingProject")
	cfg.Gisquick.ProjectCustomization = v.GetBool("gisquick.projectCustomization")
	cfg.Gisquick.Extensions = v.GetString("gisquick.extensions")

	cfg.Auth.SessionExpiration = v.GetDuration("auth.sessionExpiration")
	cfg.Auth.EmailTokenExpiration = v.GetDuration("auth.emailTokenExpiration")
	cfg.Auth.SecretKey = v.GetString("auth.secretKey")

	cfg.Web.ReadTimeout = v.GetDuration("web.readTimeout")
	cfg.Web.WriteTimeout = v.GetDuration("web.writeTimeout")
	cfg.Web.IdleTimeout = v.GetDuration("web.idleTimeout")
	cfg.Web.ShutdownTimeout = v.GetDuration("web.shutdownTimeout")
	cfg.Web.SiteURL = v.GetString("web.siteURL")
	cfg.Web.APIHost = v.GetString("web.apiHost")

	cfg.Postgres.User = v.GetString("postgres.user")
	cfg.Postgres.Password = v.GetString("postgres.password")
	cfg.Postgres.Host = v.GetString("postgres.host")
	cfg.Postgres.Name = v.GetString("postgres.name")
	cfg.Postgres.Port = v.GetInt("postgres.port")
	cfg.Postgres.MaxIdleConns = v.GetInt("postgres.maxIdleConns")
	cfg.Postgres.MaxOpenConns = v.GetInt("postgres.maxOpenConns")
	cfg.Postgres.SSLMode = v.GetString("postgres.sslMode")
	cfg.Postgres.StatementCacheMode = v.GetString("postgres.statementCacheMode")

	cfg.Redis.Network = v.GetString("redis.network")
	cfg.Redis.Addr = v.GetString("redis.addr")
	cfg.Redis.Password = v.GetString("redis.password")
	cfg.Redis.DB = v.GetInt("redis.db")

	cfg.Email.Host = v.GetString("email.host")
	cfg.Email.Port = v.GetInt("email.port")
	cfg.Email.Encryption = v.GetString("email.encryption")
	cfg.Email.Username = v.GetString("email.username")
	cfg.Email.Password = v.GetString("email.password")
	cfg.Email.Sender = v.GetString("email.sender")
	cfg.Email.ActivationSubject = v.GetString("email.activationSubject")
	cfg.Email.PasswordResetSubject = v.GetString("email.passwordResetSubject")

	cfg.Logging.Level = v.GetString("logging.level")
	cfg.Logging.File.Enabled = v.GetBool("logging.file.enabled")
	cfg.Logging.File.Path = v.GetString("logging.file.path")
	cfg.Logging.File.MaxSizeMB = v.GetInt("logging.file.maxSizeMB")
	cfg.Logging.File.MaxBackups = v.GetInt("logging.file.maxBackups")
	cfg.Logging.File.MaxAgeDays = v.GetInt("logging.file.maxAgeDays")
	cfg.Logging.File.Compress = v.GetBool("logging.file.compress")

	fallbackLevel := zap.DebugLevel
	if !cfg.Gisquick.Debug {
		fallbackLevel = zap.InfoLevel
	}
	logLevel := parseLogLevel(cfg.Logging.Level, fallbackLevel)

	log, err := createLogger(logLevel, cfg.Logging.File)
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}
	defer log.Sync()

	log.Infow("startup",
		"config_file", v.ConfigFileUsed(),
		"debug", cfg.Gisquick.Debug,
		"projects_root", cfg.Gisquick.ProjectsRoot,
		"mapserver_url", cfg.Gisquick.MapserverURL,
		"site_url", cfg.Web.SiteURL,
		"api_host", cfg.Web.APIHost)

	// Comprovar i mostrar configuracions modificades
	checkConfigChanges(log, v)

	// Database
	dbConn, err := server.OpenDB(server.DBConfig{
		User:               cfg.Postgres.User,
		Password:           cfg.Postgres.Password,
		Host:               cfg.Postgres.Host,
		Name:               cfg.Postgres.Name,
		Port:               cfg.Postgres.Port,
		MaxIdleConns:       cfg.Postgres.MaxIdleConns,
		MaxOpenConns:       cfg.Postgres.MaxOpenConns,
		SSLMode:            cfg.Postgres.SSLMode,
		StatementCacheMode: cfg.Postgres.StatementCacheMode,
	})
	if err != nil {
		return fmt.Errorf("connecting to db: %w", err)
	}
	defer func() {
		log.Infow("shutdown", "status", "stopping database support", "host", cfg.Postgres.Host)
		dbConn.Close()
	}()

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Network:  cfg.Redis.Network,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer rdb.Close()

	var es email.EmailService
	encryptionMap := map[string]mail.Encryption{
		"None":     mail.EncryptionNone,
		"SSL":      mail.EncryptionSSL,
		"TLS":      mail.EncryptionTLS,
		"SSLTLS":   mail.EncryptionSSLTLS,
		"STARTTLS": mail.EncryptionSTARTTLS,
	}
	encryption, ok := encryptionMap[cfg.Email.Encryption]
	if !ok {
		encryption = mail.EncryptionNone
	}
	if cfg.Email.Host != "" {
		es = &email.SmtpEmailService{
			Host:       cfg.Email.Host,
			Port:       cfg.Email.Port,
			Encryption: encryption,
			Username:   cfg.Email.Username,
			Password:   cfg.Email.Password,
		}
	}

	notifications := project.NewRedisNotificationStore(log, rdb)

	conf := server.Config{
		Language:             cfg.Gisquick.Language,
		LandingProject:       cfg.Gisquick.LandingProject,
		MapserverURL:         cfg.Gisquick.MapserverURL,
		MapCacheRoot:         cfg.Gisquick.MapCacheRoot,
		ProjectsRoot:         cfg.Gisquick.ProjectsRoot,
		PluginsURL:           cfg.Gisquick.PluginsURL,
		SignupAPI:            cfg.Gisquick.SignupAPI,
		SiteURL:              cfg.Web.SiteURL,
		MaxProjectSize:       int64(cfg.Gisquick.ProjectSizeLimit),
		ProjectCustomization: cfg.Gisquick.ProjectCustomization,
	}

	accountsRepo := postgres.NewAccountsRepository(dbConn)
	tokenGenerator := security.NewTokenGenerator(cfg.Auth.SecretKey, "signup", cfg.Auth.EmailTokenExpiration)
	emailSender := email.NewAccountsEmailSender(
		es,
		cfg.Email.Sender,
		cfg.Web.SiteURL,
		cfg.Email.ActivationSubject,
		cfg.Email.PasswordResetSubject,
	)
	accountsService := application.NewAccountsService(emailSender, accountsRepo, tokenGenerator)

	sessionStore := auth.NewRedisStore(rdb)
	authServ := auth.NewAuthService(log, cfg.Auth.SessionExpiration, accountsRepo, sessionStore)

	// ✅ PRIMER: Crear projectsServ sense projectsRepo
	defaultAccountConfig := domain.AccountConfig{
		ProjectsCountLimit: cfg.Gisquick.AccountProjectsLimit,
		ProjectSizeLimit:   domain.ByteSize(cfg.Gisquick.ProjectSizeLimit),
		StorageLimit:       domain.ByteSize(cfg.Gisquick.AccountStorageLimit),
	}
	var limiter application.AccountsLimiter
	if cfg.Gisquick.AccountLimiterConfig != "" {
		limiter = project.NewConfigurableProjectsLimiter(log, cfg.Gisquick.AccountLimiterConfig, defaultAccountConfig)
	} else {
		limiter = project.NewSimpleProjectsLimiter(defaultAccountConfig)
	}

	// ✅ CREAR projectsServ sense repo temporalment
	projectsServ := application.NewProjectsService(log, nil, limiter)

	// ✅ CREAR projectsRepo amb projectsServ com a configGenerator
	projectsRepo := project.NewDiskStorage(log, cfg.Gisquick.ProjectsRoot, projectsServ)

	// ✅ INJECTAR repo al projectsServ
	projectsServ.SetRepo(projectsRepo)

	sws := ws.NewSettingsWS(log)
	s := server.NewServer(log, conf, authServ, accountsService, projectsServ, sws, limiter, notifications)

	if cfg.Gisquick.Extensions != "" {
		extensionsList := strings.Split(cfg.Gisquick.Extensions, ",")
		for _, e := range extensionsList {
			if err := s.AddExtension(e); err != nil {
				log.Errorw("adding server extension", "name", e, zap.Error(err))
			}
		}
	}

	go func() {
		if err := s.ListenAndServe(cfg.Web.APIHost); err != nil && err != http.ErrServerClosed {
			log.Fatalf("shutting down the server: %v", err)
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Infof("Received shutdown signal")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}
	return nil
}

func createLogger(level zapcore.Level, fileCfg FileLoggingConfig) (*zap.SugaredLogger, error) {
	if !fileCfg.Enabled {
		config := zap.NewDevelopmentConfig()
		config.OutputPaths = []string{"stdout"}
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		config.DisableStacktrace = true
		config.Level.SetLevel(level)
		logger, err := config.Build()
		if err != nil {
			return nil, err
		}
		_, _ = zap.RedirectStdLogAt(logger, level)
		return logger.Sugar(), nil
	}

	logPath := strings.TrimSpace(fileCfg.Path)
	if logPath == "" {
		return nil, fmt.Errorf("logging.file.path is empty")
	}

	if dir := filepath.Dir(logPath); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("creating log directory %s: %w", dir, err)
		}
	}

	rotator := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    fileCfg.MaxSizeMB,
		MaxBackups: fileCfg.MaxBackups,
		MaxAge:     fileCfg.MaxAgeDays,
		Compress:   fileCfg.Compress,
	}

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderCfg.EncodeLevel = zapcore.CapitalLevelEncoder
	encoder := zapcore.NewConsoleEncoder(encoderCfg)
	ws := zapcore.AddSync(rotator)
	core := zapcore.NewCore(encoder, ws, level)

	logger := zap.New(
		core,
		zap.AddCaller(),
		zap.ErrorOutput(ws),
	)
	_, _ = zap.RedirectStdLogAt(logger, level)
	return logger.Sugar(), nil
}
