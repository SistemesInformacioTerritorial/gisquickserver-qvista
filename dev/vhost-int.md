
####### gisquick int

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

