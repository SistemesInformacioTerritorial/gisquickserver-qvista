# Manual de configuració del servidor (QVistaWeb / Gisquick Server)

Aquest document descriu com configurar i operar el servidor en entorns Windows (i quines opcions hi ha per desplegar-lo amb Apache com a proxy invers).

## 1) Què és i com s'arrenca

El binari exposa un comandament principal `serve` que aixeca l'API HTTP/WebSocket.

- Compilació (opcional):

  ```powershell
  go mod tidy
  go build -o qvistaweb.exe -ldflags="-s -w" cmd\main.go
  ```

- Arrencada:

  ```powershell
  .\qvistaweb.exe serve
  ```

## 2) Dependències externes

Perquè el servidor funcioni, normalment necessites:

- PostgreSQL (per a usuaris, comptes i metadata)
- Redis (sessions, notificacions)
- QGIS Server / `qgis_mapserv` (per servir mapes). En aquesta adaptació sovint s'utilitza via Apache + `fcgi`.
- (Opcional) SMTP si vols enviament real de correus (activació/restabliment).

## 3) Fitxer de configuració: ubicació i precedència

El servidor carrega la configuració amb Viper, en aquest ordre:

1. Si existeix la variable d'entorn `GISQUICK_CONFIG`, s'utilitza com a ruta explícita cap al YAML.
2. Si no, busca un `config.yaml` a:
   - directori actual (`.`)
   - `./config`
   - `/etc/gisquick` (útil a Linux)

A més, qualsevol clau es pot sobreescriure amb variables d'entorn amb el prefix `GISQUICK_`.

Exemple:

- Clau YAML: `web.apiHost`
- Variable d'entorn equivalent: `GISQUICK_WEB_APIHOST`

### Com definir variables d'entorn a Windows

En PowerShell (només per a la sessió actual):

```powershell
$env:GISQUICK_CONFIG = "C:\gisquick\config.yaml"
$env:GISQUICK_WEB_APIHOST = "0.0.0.0:3000"
```

Persistent (per al sistema):

```powershell
setx GISQUICK_CONFIG "C:\gisquick\config.yaml" /M
```

## 4) Plantilles de configuració per entorn

A l'arrel del repo hi ha exemples a punt per copiar:

- `config_int.yaml` (integració)
- `config_pre.yaml` (preproducció)
- `config_pro.yaml` (producció)
- `config_test.yaml` (tests)

Patró recomanat:

```powershell
New-Item -ItemType Directory -Force C:\gisquick-int\publish | Out-Null
New-Item -ItemType Directory -Force C:\gisquick-int\logs | Out-Null
Copy-Item .\qvistaweb.exe C:\gisquick-int\
Copy-Item .\config_int.yaml C:\gisquick-int\config.yaml
```

I arrencar apuntant a aquest YAML:

```powershell
$env:GISQUICK_CONFIG = "C:\gisquick-int\config.yaml"
C:\gisquick-int\qvistaweb.exe serve
```

## 5) Referència completa de configuració

A continuació es llisten les claus que el comandament `serve` llegeix actualment.

### 5.1) `gisquick.*`

- `gisquick.debug` (bool)
  - Activa logs més detallats i, per defecte, posa el nivell de log a `debug`.
- `gisquick.language` (string)
  - Idioma per defecte (p.ex. `es-es`, `ca-es`, `en-us`).
- `gisquick.projectsRoot` (string)
  - Carpeta arrel on es guarden els projectes en disc.
  - Recomanació: mantenir una carpeta per entorn (int/pre/pro) per evitar barrejar dades.
- `gisquick.mapserverURL` (string)
  - URL base de QGIS MapServer/FCGI (p.ex. `http://localhost:8080/cgi-bin/qgis_mapserv.fcgi.exe`).
- `gisquick.projectSizeLimit` (string mida)
  - Límit per projecte. Admet sufixos `M` o `G` (p.ex. `100M`, `1G`).
- `gisquick.accountStorageLimit` (string mida)
  - Límit total d'emmagatzematge per compte. Admet `M`/`G`.
- `gisquick.accountProjectsLimit` (int)
  - Nombre màxim de projectes per compte.
- `gisquick.accountLimiterConfig` (string ruta a directori)
  - Si es defineix, el servidor permet límits per usuari llegint fitxers JSON des d'aquest directori.
  - Format esperat: un fitxer per usuari, anomenat `<username>.json`.

  Exemple `C:\gisquick\limits\jane.json`:
  ```json
  {
    "projects_limit": 25,
    "project_size_limit": "200M",
    "storage_limit": "2G"
  }
  ```

- `gisquick.landingProject` (string)
  - Projecte que es suggereix com a “landing” en la inicialització de l'app.
- `gisquick.projectCustomization` (bool)
  - Si està actiu, el servidor intenta retornar configuració de customització del projecte quan aplica.
- `gisquick.pluginsURL` (string)
  - Si no és buit, habilita endpoints de repositori de plugins (`/plugins/...`).
- `gisquick.signupAPI` (bool)
  - Si és `true`, habilita endpoints de registre/activació de comptes (`/api/accounts/signup`, etc.).
- `gisquick.mapCacheRoot` (string)
  - Ruta per a cache de mapes.
  - Nota: al codi actual no hi ha rutes de mapcache actives (estan comentades), per tant aquesta clau pot no tenir efecte.
- `gisquick.extensions` (string)
  - Llista separada per comes amb extensions del servidor.
  - Nota: al codi actual no hi ha extensions registrades, així que normalment es deixa buit.

### 5.2) `auth.*`

- `auth.sessionExpiration` (duration)
  - Durada de sessió (p.ex. `24h`).
- `auth.emailTokenExpiration` (duration)
  - Caducitat del token d'email (registre/activació). P.ex. `72h`.
- `auth.secretKey` (string)
  - Clau secreta per signar tokens.
  - Recomanació: usar un valor llarg, aleatori i no versionar-lo a Git.

### 5.3) `web.*`

- `web.readTimeout`, `web.writeTimeout`, `web.idleTimeout`, `web.shutdownTimeout` (duration)
  - Timeouts del servidor HTTP.
- `web.siteURL` (string)
  - URL pública del lloc, usada per als enllaços dels correus.
- `web.apiHost` (string)
  - Host:port on escolta l'API (p.ex. `0.0.0.0:3000`).

### 5.4) `postgres.*`

- `postgres.user`, `postgres.password`, `postgres.host`, `postgres.name`, `postgres.port`
- `postgres.maxIdleConns`, `postgres.maxOpenConns`
- `postgres.sslMode` (string, p.ex. `disable`, `prefer`, ...)
- `postgres.statementCacheMode` (string, per defecte `prepare`)

### 5.5) `redis.*`

- `redis.network` (string, típic `tcp`)
- `redis.addr` (host:port, p.ex. `localhost:6379`)
- `redis.password` (string)
- `redis.db` (int)

Recomanació per entorns: usar un `redis.db` diferent per a int/pre/pro.

### 5.6) `email.*`

- `email.host`, `email.port`
- `email.encryption` (string)
  - Valors típics: `STARTTLS`, `TLS`, `SSL`, `None`.
- `email.username`, `email.password`, `email.sender`
- `email.activationSubject`, `email.passwordResetSubject`

Si `email.host` és buit, el servidor no configura SMTP i l'enviament real de correu no estarà disponible.

### 5.7) `logging.*`

- `logging.level` (string)
  - `debug`, `info`, `warn`, `error`.
  - Si és buit, es dedueix de `gisquick.debug`.

- `logging.file.*`
  - `logging.file.enabled` (bool)
  - `logging.file.path` (string, ruta del log)
  - `logging.file.maxSizeMB`, `logging.file.maxBackups`, `logging.file.maxAgeDays`, `logging.file.compress`

Si `logging.file.enabled` és `false`, el log s'escriu a `stdout`.

## 6) Configurar Apache com a proxy invers (recomanat)

Si el teu desplegament usa Apache:

- Usa com a referència la guia existent `configuracio_apache.md`.
- Per vhosts per entorn, mira els exemples a `dev/vhost-int.md`, `dev/vhost-pre.md`, `dev/vhost-pro.md`.

El patró habitual és:

- `/api/` i `/ws/` proxificats al backend Go (`web.apiHost`)
- `/qgis-server` proxificat a FCGI de QGIS Server
- `DocumentRoot` apuntant al frontend estàtic (`www/html/...`)

## 7) Executar com a servei a Windows

Hi ha diverses opcions. La més robusta sol ser usar un wrapper (NSSM / WinSW) que permeti fixar variables d'entorn i reinicis.

Si vols usar `sc create`, recorda:

- El servei s'ha d'executar amb un usuari que tingui permisos sobre:
  - `gisquick.projectsRoot`
  - `logging.file.path` (i el seu directori)
- Has de garantir que el procés rebi `GISQUICK_CONFIG` (o que el `config.yaml` estigui al working directory).

Referència: a `configuracio_apache.md` hi ha un exemple de creació de servei i configuració per entorns.

## 8) Migracions de base de dades

El binari inclou el comandament `migrate`:

```powershell
.\qvistaweb.exe migrate up
```

Nota important: el codi actual de migracions està orientat a execució en contenidor (usa `file:///app/migrations`). A Windows local pot ser que no trobi les migracions si no existeix aquesta ruta.

Alternatives:

- Executar migracions des de l'entorn Docker (si uses els Dockerfiles).
- Aplicar manualment els SQL del directori `migrations/`.

## 9) Checklist ràpid de verificació

- API aixecada a `web.apiHost` (port correcte i accessible)
- PostgreSQL accessible (host/port/credencials)
- Redis accessible
- `gisquick.projectsRoot` existeix i té permisos
- `gisquick.mapserverURL` respon (QGIS Server/FCGI accessible)
- Logs generant-se (a fitxer o a stdout segons configuració)
