# QVistaWeb - Adaptació de Gisquick per a Windows

Aquest projecte és una adaptació de Gisquick específicament dissenyada per a entorns Windows.

## Compilació

Per compilar l'executable:

```bash
# Actualitzar dependències
go mod tidy

# Compilar versió optimitzada
go build -o qvistaweb.exe -ldflags="-s -w" cmd\main.go
```

## Execució

Per iniciar el servidor:

```bash
qvistaweb.exe serve
```

## Manual de configuració / configuració del servidor

Vegeu [MANUAL_CONFIGURACION_SERVER.md](MANUAL_CONFIGURACION_SERVER.md).

Comandes disponibles:

- `serve`: Inicia el servidor web
- `adduser`: Afegeix un usuari nou
- `addsuperuser`: Afegeix un usuari amb privilegis d'administrador
- `deleteuser`: Elimina un usuari existent
- `dumpusers`: Exporta llista d'usuaris
- `loadusers`: Importa llista d'usuaris

## Configuració externa

QVistaWeb utilitza un arxiu de configuració YAML per definir el seu comportament. Aquest arxiu ha d'ubicar-se en algun d'aquests llocs:

- Al directori actual: config.yaml
- Al directori de configuració: `./config/config.yaml`
- A la ruta: `/etc/gisquick/config.yaml`
- O especificat mitjançant la variable d'entorn: `GISQUICK_CONFIG`

### Exemple d'arxiu de configuració (config.yaml)

```yaml
gisquick:
  debug: true
  language: ca-es
  projectsRoot: c:/gisquick/publish
  mapserverURL: http://localhost:8080/cgi-bin/qgis_mapserv.fcgi.exe
  projectSizeLimit: 100M
  accountStorageLimit: 1G
  projectCustomization: true

auth:
  sessionExpiration: 24h
  emailTokenExpiration: 72h
  secretKey: la-teva-clau-secreta

web:
  readTimeout: 5s
  writeTimeout: 10s
  idleTimeout: 120s
  shutdownTimeout: 20s
  siteURL: http://127.0.0.1
  apiHost: 0.0.0.0:3000

postgres:
  user: postgres
  password: ********
  host: localhost
  name: postgres
  port: 5433
  maxIdleConns: 3
  maxOpenConns: 3
  sslMode: disable
  statementCacheMode: prepare

redis:
  addr: localhost:6379
  password: ""
  db: 0

email:
  host: smtp.example.com
  port: 587
  encryption: STARTTLS
  username: user@example.com
  password: password
  sender: user@example.com
  activationSubject: QVistaWeb Registre
  passwordResetSubject: QVistaWeb Restabliment de Contrasenya
```

### Sobreescriure la configuració amb variables d'entorn

Totes les configuracions poden ser sobreescrites mitjançant variables d'entorn amb el prefix `GISQUICK_`:

```
set GISQUICK_POSTGRES_PASSWORD=altra-contrasenya
set GISQUICK_WEB_APIHOST=0.0.0.0:4000
```

## Repositori del projecte

El codi font es troba disponible a:
https://github.com/SistemesInformacioTerritorial/gisquickserver-qvista.git