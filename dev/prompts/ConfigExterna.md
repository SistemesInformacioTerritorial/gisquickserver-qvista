
# Configuración Externa para Gisquick

Puedes implementar un sistema de configuración externa para Gisquick que permita modificar parámetros sin necesidad de recompilar la aplicación. Hay varias formas de hacerlo:

## Opción 1: Usar un archivo de configuración YAML/JSON

Puedes modificar tu aplicación para que lea un archivo de configuración externo. La solución más directa sería:

```go
func Serve() error {
    // Primero determinar qué archivo de configuración usar
    configFile := os.Getenv("GISQUICK_CONFIG")
    if configFile == "" {
        // Valor por defecto
        configFile = "config.yaml" // o .json, .toml, etc.
    }

    // Definir tu estructura de configuración como ya lo haces
    cfg := struct {
        Gisquick struct {
            // ...todos tus campos actuales...
        }
        // ...resto de tu configuración...
    }{}

    // Cargar configuración con prioridades:
    // 1. Valores por defecto (los que tienes declarados con `conf:"default:..."`)
    // 2. Archivo de configuración
    // 3. Variables de entorno (que sobreescriben el archivo)
    
    // Cargar desde archivo si existe
    if _, err := os.Stat(configFile); err == nil {
        data, err := os.ReadFile(configFile)
        if err != nil {
            return fmt.Errorf("error leyendo archivo de configuración: %w", err)
        }
        
        // Dependiendo del formato del archivo
        if strings.HasSuffix(configFile, ".json") {
            if err := json.Unmarshal(data, &cfg); err != nil {
                return fmt.Errorf("error parseando config JSON: %w", err)
            }
        } else if strings.HasSuffix(configFile, ".yaml") || strings.HasSuffix(configFile, ".yml") {
            // Necesitarías importar "gopkg.in/yaml.v3"
            if err := yaml.Unmarshal(data, &cfg); err != nil {
                return fmt.Errorf("error parseando config YAML: %w", err)
            }
        }
        // Podrías agregar más formatos según necesites
    }

    // Luego aplicar configuración de variables de entorno
    // (esto sobreescribirá lo que venga del archivo)
    prefix := ""
    help, err := conf.Parse(prefix, &cfg)
    if err != nil {
        // ... tu código de manejo de errores
    }
    
    // Continúa con el resto de tu código...
}
```

## Opción 2: Usar Viper (solución completa)

Una solución más elegante sería usar [Viper](https://github.com/spf13/viper), que está diseñado específicamente para este propósito y maneja mejor las prioridades:

```go
package commands

import (
    // Tus imports actuales
    "github.com/spf13/viper"
)

func Serve() error {
    // Configurar Viper
    v := viper.New()
    
    // Establecer valores por defecto
    v.SetDefault("gisquick.debug", true)
    v.SetDefault("gisquick.language", "en-us")
    v.SetDefault("gisquick.projectsRoot", "c:/gisquick/publish")
    // ... configurar todos los valores por defecto
    
    // Buscar archivo de configuración
    v.SetConfigName("config") // nombre del archivo sin extensión
    v.SetConfigType("yaml")   // tipo de archivo (también soporta json, toml, etc.)
    v.AddConfigPath(".")      // buscar en directorio actual
    v.AddConfigPath("/etc/gisquick/") // buscar en directorio de configuración
    
    // Leer variables de entorno con prefijo GISQUICK
    v.SetEnvPrefix("GISQUICK")
    v.AutomaticEnv()
    v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
    
    // Leer archivo de configuración
    err := v.ReadInConfig()
    if err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
            // No es un error si el archivo no existe, pero sí si hay problema al leerlo
            return fmt.Errorf("error leyendo archivo de configuración: %w", err)
        }
    }
    
    // Ahora puedes leer valores de configuración directamente:
    debug := v.GetBool("gisquick.debug")
    projectsRoot := v.GetString("gisquick.projectsRoot")
    
    // O puedes cargar toda la configuración en tu estructura:
    var cfg YourConfigStruct
    if err := v.Unmarshal(&cfg); err != nil {
        return fmt.Errorf("error unmarshal de la configuración: %w", err)
    }
    
    // Continúa con tu código usando cfg...
}
```

## Ejemplo de archivo de configuración (YAML)

Un archivo de configuración `config.yaml` podría verse así:

```yaml
gisquick:
  debug: true
  language: es-es
  projectsRoot: c:/gisquick/publish
  mapserverURL: http://localhost:8080/cgi-bin/qgis_mapserv.fcgi.exe
  projectSizeLimit: 100M
  accountStorageLimit: 1G

auth:
  sessionExpiration: 24h
  secretKey: mi-clave-secreta-personalizada

web:
  apiHost: 0.0.0.0:3000
  siteURL: http://mi-gisquick.local

postgres:
  user: postgres
  password: nexus
  host: localhost
  port: 5433
  name: postgres

redis:
  addr: localhost:6379
```

## Implementación recomendada

Te recomendaría la opción 2 con Viper, ya que:

1. Maneja múltiples formatos de archivo (YAML, JSON, TOML)
2. Gestiona correctamente la prioridad entre valores por defecto, archivo y variables de entorno
3. Ofrece validación y opciones más avanzadas
4. Es una biblioteca muy usada y probada en el ecosistema Go

Si prefieres una solución más ligera y continuar con lo que ya tienes, la opción 1 es viable pero requerirá más código manual para manejar todas las situaciones.