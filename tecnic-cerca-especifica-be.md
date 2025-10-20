# Resumen para Backend - Implementación de Variables qV_search

**Tarea:** Añadir soporte para variables de búsqueda específica en capas vectoriales

---

## 🎯 Objetivo

Modificar el backend (Go) para que lea las variables `qV_search` de los proyectos QGIS y las incluya en el `project.json` generado.

## 📋 Qué Hacer

### 1. Modificar Estructura de Datos

**Archivo:** Estructura `LayerData`

```go
type LayerData struct {
    Name       string      `json:"name"`
    Title      string      `json:"title"`
    Visible    bool        `json:"visible"`
    Queryable  bool        `json:"queryable"`
    Type       string      `json:"type"`
    GeomType   string      `json:"geom_type"`
    Attributes []Attribute `json:"attributes"`
    QVSearch   string      `json:"qV_search,omitempty"` // ← NUEVA LÍNEA
}
```

### 2. Leer Variables del XML

**Ubicación en .qgs:**
```xml
<customproperties>
  <Option name="variableNames" type="StringList">
    <Option value="qV_search" type="QString"/>
  </Option>
  <Option name="variableValues" type="StringList">
    <Option value="field=&quot;CODIGO&quot; fieldtext=&quot;Código&quot; desc=&quot;Introducir código&quot;" type="QString"/>
  </Option>
</customproperties>
```

**Función a implementar:**
```go
func extractQVSearchVariable(customProps *CustomProperties) string {
    variableNames := getStringListOption(customProps, "variableNames")
    variableValues := getStringListOption(customProps, "variableValues")
    
    // Buscar índice de qV_search
    for i, name := range variableNames {
        if name == "qV_search" && i < len(variableValues) {
            return variableValues[i]
        }
    }
    return ""
}
```

### 3. Integrar en Procesamiento

**Modificar función de procesamiento de capas:**
```go
func processLayer(qgisLayer *QGISLayer) *LayerData {
    layerData := &LayerData{
        // ... propiedades existentes
    }
    
    // NUEVA FUNCIONALIDAD
    if customProps := qgisLayer.CustomProperties; customProps != nil {
        qvSearchValue := extractQVSearchVariable(customProps)
        if qvSearchValue != "" {
            layerData.QVSearch = qvSearchValue
        }
    }
    
    return layerData
}
```

## ✅ Resultado Esperado

### Antes (project.json actual):
```json
{
  "layers": [
    {
      "name": "parcelas",
      "title": "Parcelas",
      "visible": true,
      "attributes": [...]
    }
  ]
}
```

### Después (project.json objetivo):
```json
{
  "layers": [
    {
      "name": "parcelas", 
      "title": "Parcelas",
      "visible": true,
      "attributes": [...],
      "qV_search": "field=\"CODIGO\" fieldtext=\"Código Parcela\" desc=\"Introduzca el código\""
    }
  ]
}
```

## 🧪 Testing

### Test Unitario:
```go
func TestExtractQVSearchVariable(t *testing.T) {
    customProps := &CustomProperties{
        VariableNames: []string{"qV_search", "otra_var"},
        VariableValues: []string{"field=\"CODIGO\" fieldtext=\"Código\"", "valor2"}
    }
    
    result := extractQVSearchVariable(customProps)
    expected := "field=\"CODIGO\" fieldtext=\"Código\""
    
    assert.Equal(t, expected, result)
}
```

## ⚠️ Notas Importantes

1. **Solo añadir la propiedad si existe la variable** - No romper capas sin qV_search
2. **Preservar el formato exacto** - No modificar la cadena de la variable
3. **Gestión de errores** - Si hay problemas, simplemente omitir la propiedad
4. **Compatibilidad 100%** - Proyectos existentes deben funcionar igual

## 🎯 Criterios de Aceptación

- [ ] Variables qV_search se leen del XML correctamente
- [ ] Se añaden al project.json cuando existen
- [ ] Capas sin qV_search siguen funcionando normalmente
- [ ] No afecta el rendimiento del procesamiento actual

---

**Estimación:** 2-3 días de desarrollo + testing

**Frontend:** Ya está preparado para recibir estos datos automáticamente