

## Guia per configurar els serveis

### Configuració del servei per a cada entorn

1. **Copiar els executables i arxius de configuració**:

   ```batch
   mkdir C:\gisquick-int
   mkdir C:\gisquick-int\publish
   mkdir C:\gisquick-int\www
   mkdir C:\gisquick-int\www\html
   copy qvistaweb.exe C:\gisquick-int\
   copy config_int.yaml C:\gisquick-int\config.yaml
   ```

2. **Crear el servei de Windows** (per a cada entorn):

   ```batch
   :: Per a integració
   sc create QVistaWebINT binPath= "C:\gisquick-int\qvistaweb.exe serve" DisplayName= "QVistaWeb Integració" start= auto
   sc description QVistaWebINT "Servei QVistaWeb per a l'entorn d'integració"
   
   :: Afegir la variable d'entorn per al servei
   sc failure QVistaWebINT reset= 86400 actions= restart/60000/restart/60000/restart/60000
   reg add "HKLM\SYSTEM\CurrentControlSet\Services\QVistaWebINT\Parameters" /v AppEnvironment /t REG_MULTI_SZ /d "GISQUICK_CONFIG=C:\gisquick-int\config.yaml" /f
   
   :: Per a preproducció i producció (similar)
   ```

3. **Configurar l'Apache** (per a cada entorn):

   Afegir els vhosts corresponents al fitxer de configuració d'Apache:
   
   - Per a integració: Utilitzar el contingut del fitxer vhost-int.md
   - Per a preproducció: Utilitzar el contingut del fitxer vhost-pre.md 
   - Per a producció: Utilitzar el contingut del fitxer vhost-pro.md

4. **Iniciar el servei**:

   ```batch
   net start QVistaWebINT
   ```

### Verificació del funcionament

Per verificar que l'entorn s'ha configurat correctament:

1. Comprovar els logs del servei:
   ```
   type C:\gisquick-int\logs\qvistaweb.log
   ```

2. Comprovar l'accés via navegador:
   - Integració: http://localhost:82/
   - Preproducció: http://localhost:81/
   - Producció: http://localhost/

3. Verificar l'API:
   ```
   curl http://localhost:5000/api/status   // Integració
   curl http://localhost:4000/api/status   // Preproducció
   curl http://localhost:3000/api/status   // Producció
   ```

Amb aquesta configuració, tindreu tres entorns independents (integració, preproducció i producció) funcionant al mateix servidor però amb ports, directoris i configuracions separades.

#### gisquick int
```
<VirtualHost *:82>

# Configuración del proxy inverso para el backend en preproducción
ProxyPass /api/ http://localhost:5000/api/
ProxyPassReverse /api/ http://localhost:5000/api/

ProxyPass /ws/ ws://localhost:5000/ws/
ProxyPassReverse /ws/ ws://localhost:5000/ws/

# Configuración del proxy inverso para /qgis-server (mismo puerto)
<Location /qgis-server>
    ProxyPass fcgi://localhost:8080/
    ProxyPassReverse fcgi://localhost:8080/
</Location>

# Configuración de las rutas y reescrituras del gisquick en preproducción
<Directory C:/gisquick-int/www/html/>
    Require all granted
</Directory>

<Directory C:/gisquick-int/www/html/user>
    Require all granted

    RewriteEngine On
    RewriteCond %{ENV:REDIRECT_STATUS} ^$
    RewriteCond %{REQUEST_URI} ^/user(/.*)?$
    RewriteCond %{REQUEST_URI} !^/user/static/.*
    RewriteRule ^ /user/ [L]
</Directory>

# Alias para archivos específicos en preproducción
Alias /index.html C:/gisquick-int/www/html/map/index.html
Alias /favicon.ico C:/gisquick-int/www/html/map/favicon.ico
Alias /manifest.json C:/gisquick-int/www/html/map/manifest.json
Alias /service-worker.js C:/gisquick-int/www/html/map/service-worker.js
AliasMatch ^/workbox.* C:/gisquick-int/www/html/map/workbox

# Alias para la ruta /map/ en preproducción
Alias /map/ C:/gisquick-int/www/html/map/map/

# Configuración del directorio raíz en preproducción
DocumentRoot C:/gisquick-int/www/html/

<Directory C:/gisquick-int/www/html/map/>
    Options Indexes FollowSymLinks
    AllowOverride None
    Require all granted

    # Configuración de las rutas
    <FilesMatch "\.(html|ico|json|js|map)$">
        Require all granted
    </FilesMatch>
</Directory>

</VirtualHost>
```
