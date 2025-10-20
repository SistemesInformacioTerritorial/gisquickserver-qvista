# 🧪 Guia de Testing - Sistema de Cerca Específica (qV_search)

**Projecte:** Gisquick-QVista  
**Data:** 6 de juny de 2025  
**Desenvolupador:** JFS  

---

## 🎯 **Què hem implementat?**

El sistema de **cerca específica** que permet buscar directament dins dels camps de les capes vectorials. Ja no cal només cercar adreces de Barcelona - ara pots cercar "PARC001" dins la capa de parcel·les!

---

## ⚡ **Funciona sense republicar?**

**SÍ!** El sistema funciona **dinàmicament**. Quan l'usuari carrega un projecte, el backend llegeix les variables del `.qgs` i les integra automàticament al project.json.

```go
// Això es fa cada cop que es carrega un projecte (línia 817-821)
layerVariables := s.extractLayerVariables(projectName, meta.Layers)
if len(layerVariables) > 0 {
    s.integrateVariablesIntoLayers(layers, layerVariables, meta.Layers)
}
```

---

## 🧪 **Com fer el testing**

### **1. Preparar el projecte QGIS**

Al QGIS, afegir variables a una capa:

```_
Propietats de la capa → Variables →
Nom: qV_search
Valor: field="CODIGO" fieldtext="Código Parcela"
```

O variables de projecte:
```
Propietats del Projecte → Variables →
qV_search_global = field="NOM" fieldtext="Nom"
```

### **2. Compilar i executar**

```bash
# Compilar
go build -o qvistaweb.exe -ldflags="-s -w" cmd\main.go

# Executar
qvistaweb.exe serve
```

### **3. Provar l'API**

```bash
# Cridar l'API del projecte
curl http://localhost:port/api/map/config/el-teu-projecte

# O amb navegador
http://localhost:port/api/map/config/el-teu-projecte

#3000 pro, 4000 pre, 5000 int
```

### **4. Què buscar a la resposta**

Si tot va bé, hauràs de veure:

```json
{
  "layers": [
    {
      "name": "parcelas",
      "title": "Parcel·les",
      "qV_search": "field=\"CODIGO\" fieldtext=\"Código Parcela\"",
      "visible": true
    }
  ]
}
```

---

## 🔍 **Casos de prova**

### **✅ TC001: Projecte amb variables qV_search**
- **Pas:** Carregar projecte amb variables configurades
- **Esperat:** Variables apareixen al project.json
- **Com:** Buscar `"qV_search"` a la resposta de l'API

### **✅ TC002: Projecte sense variables**
- **Pas:** Carregar projecte normal (sense variables)
- **Esperat:** Funciona igual que sempre
- **Com:** L'API retorna project.json sense camps qV_search

### **✅ TC003: Variables de projecte**
- **Pas:** Configurar variables a nivell de projecte QGIS
- **Esperat:** Apareixen com a `@project` als logs
- **Com:** Revisar logs del servidor

### **✅ TC004: Grups de capes**
- **Pas:** Projecte amb grups i subgrups
- **Esperat:** Variables s'integren recursivament
- **Com:** Comprovar capes dins de grups

---

## 📋 **Logs per revisar**

El sistema genera logs detallats:

```
[extractLayerVariables] variables encontrades en capa
[integrateVariablesIntoLayers] Variable integrada
[integrateVariablesIntoLayers] Integració completada
```

**Com veure'ls:** Executar el servidor i mirar la consola.

---

## 🚨 **Possibles problemes**

### **Si no apareixen les variables:**
1. Comprovar que les variables estan ben configurades al QGIS
2. Revisar els logs - han de mostrar "variables encontrades"
3. Verificar que el nom de la capa coincideix

### **Si el projecte no carrega:**
1. Comprovar que el projecte funcionava abans
2. Revisar logs d'errors
3. El sistema és retrocompatible - no hauria de trencar res

---

## 🎯 **Resultat esperat**

Si tot va bé:
- ✅ Projectes amb variables: mostren qV_search al JSON
- ✅ Projectes sense variables: funcionen igual que sempre  
- ✅ Logs detallats per debugging
- ✅ Cap projecte existent es trenca

---

## 🛠️ **Consells per al testing**

1. **Comença simple:** Prova amb un projecte petit amb 1-2 capes
2. **Comprova els logs:** Són la millor manera de veure què passa
3. **Prova casos extrems:** Projectes grans, moltes capes, grups complexos
4. **Retrocompatibilitat:** Prova projectes existents per assegurar que no es trenquen

---

**🚀 El sistema està llest per provar!**