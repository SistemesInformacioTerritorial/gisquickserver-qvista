#gisquick
<VirtualHost *:80>
#ProxyPass /api/ http://127.0.0.1:3000/api/
#ProxyPassReverse /api/ http://127.0.0.1:3000/api/


ProxyPass /api/ http://localhost:3000/api/
ProxyPassReverse /api/ http://localhost:3000/api/

ProxyPass /ws/ ws://localhost:3000/ws/
ProxyPassReverse /ws/ ws://localhost:3000/ws/

#ProxyPass /ws/ ws://127.0.0.1:3000/ws/

#ProxyPassReverse /ws/ ws://127.0.0.1:3000/ws/


# Configuración del proxy inverso para /qgis-server
    <Location /qgis-server>
    ProxyPass fcgi://localhost:8080/
    ProxyPassReverse fcgi://localhost:8080/
</Location>
  
  
  
# Configuración de las rutas y reescrituras del gisquick
<Directory C:/gisquick/www/html/>
    Require all granted
</Directory>

<Directory C:/gisquick/www/html/user>
    Require all granted

    RewriteEngine On
    RewriteCond %{ENV:REDIRECT_STATUS} ^$
    RewriteCond %{REQUEST_URI} ^/user(/.*)?$
    RewriteCond %{REQUEST_URI} !^/user/static/.*
    RewriteRule ^ /user/ [L]
</Directory>

	

# Alias para archivos específicos
Alias /index.html C:\gisquick\www/html/map/index.html
Alias /favicon.ico C:\gisquick\www/html/map/favicon.ico
Alias /manifest.json C:\gisquick\www/html/map/manifest.json
Alias /service-worker.js C:\gisquick\www/html/map/service-worker.js
AliasMatch ^/workbox.* C:\gisquick\www/html/map/workbox

# Alias para la ruta /map/
Alias /map/ C:\gisquick\www/html/map/map/

# Configuración del directorio raíz
DocumentRoot C:\gisquick\www/html/

<Directory C:\gisquick\www/html/map/>
    Options Indexes FollowSymLinks
    AllowOverride None
    Require all granted

    # Configuración de las rutas
    <FilesMatch "\.(html|ico|json|js|map)$">
        Require all granted
    </FilesMatch>
</Directory>


</VirtualHost>