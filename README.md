# Gisquick application server per Windows

El següent producte es una adaptació del producte opensource gisquick-server-next per a windows.

El producte s'enmarca dins de la següent arquitectura i com podem veure es la peça central de l'stack. La seva missió principal es la intercomunicació de tots els subsistemes:

![image](https://github.com/user-attachments/assets/f6a9a6c0-5015-4f64-9416-ae7b74422901)

## Instalació/Configuració del producte del producte

Donat que no volem fer servir el docker, i en ell resideixen tots els subsistemes, caldrà instalarlos per separat en Windows:
D'especial importància es el servidor web/proxy que s'ha substituit des d'un Caddy a un Apache.

### Instal.lació de subsistemes de backend:

1. Cal instalar o tenir accés a un   Postgres v14 o més   per gestionar els usuaris, amb la taula:
<<users.sql>

    i posteriorment configurar el servidor amb els parametres que toqui
    ![image](https://github.com/user-attachments/assets/099973c4-9cca-4020-ad94-a77f8e3b7e41)

als arxius (als tres) : 
![image](https://github.com/user-attachments/assets/8faaf233-3762-4a00-b6dd-01285e4be8c6)

2. Cal també un servidor QGISServer per servir els mapes i configurarlo a serve.go:
![image](https://github.com/user-attachments/assets/81ed8785-d55b-441f-bca3-71c3d6473831)


3. El mateix pel Redis que gestiona les sessions d'usuaris
 ![image](https://github.com/user-attachments/assets/64ebfbb4-1775-46a0-9a39-6519839dad20)

4. Finalment configurem el port del server, aixi com la carpeta on s'allotjaran els projectes:
   ![image](https://github.com/user-attachments/assets/637e62c8-6d14-4ca7-9f40-a0489980c96e)


posteriorment compilarem amb la comanda:
`go build -o gisquick.exe -ldflags="-s -w" cmd\main.go`
 i només caldrà executar l'executable
 

5. Adicionalment es requereix un servidor apache v3.1 amb la configuracio que s'adjunta a httpd.conf que farà de:
   a. tant de webserver per les SPA,
   b. com de de reverse proxy per la comunicació back-front(Http), i Plugin-Server (WebSockets)

 
### Gisquick application server (backend) linux

```
docker build -t gisquick/server-dev -f ./docker/Dockerfile.dev .
```

```
docker build -t gisquick/server -f ./docker/Dockerfile-alpine .
```
