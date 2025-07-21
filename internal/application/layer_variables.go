package application

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gisquick/gisquick-server/internal/domain"
	"go.uber.org/zap"
)

// LayerVariablesManager gestiona la extracción e integración de variables de capa
type LayerVariablesManager struct {
	log *zap.SugaredLogger
}

// NewLayerVariablesManager crea un nuevo gestor de variables de capa
func NewLayerVariablesManager(log *zap.SugaredLogger) *LayerVariablesManager {
	return &LayerVariablesManager{
		log: log,
	}
}

// ExtractLayerVariables extrae variables de capa desde el repositorio
func (lvm *LayerVariablesManager) ExtractLayerVariables(
	projectName string,
	metaLayers map[string]domain.LayerMeta,
	repo interface{},
) map[string]interface{} {
	lvm.log.Infow("🔍 [ExtractLayerVariables] INICIANDO EXTRACCIÓN",
		"project", projectName,
		"metaLayersCount", len(metaLayers))

	// Mostrar todas las capas del meta para diagnóstico
	lvm.log.Infow("📋 [ExtractLayerVariables] CAPAS EN METADATA:")
	for metaId, metaLayer := range metaLayers {
		lvm.log.Infow("   📌 Meta Capa",
			"metaId", metaId,
			"metaName", metaLayer.Name,
			"metaTitle", metaLayer.Title)
	}

	layerVariables := make(map[string]interface{})

	// Usar el repositorio para obtener variables del QGS
	if repoWithVars, ok := repo.(interface {
		GetLayerVariables(string) (map[string]map[string]string, error)
	}); ok {
		varsFromQgs, err := repoWithVars.GetLayerVariables(projectName)
		if err != nil {
			lvm.log.Errorw("❌ [ExtractLayerVariables] Error obteniendo variables del QGS",
				"project", projectName,
				"error", err)
		} else {
			lvm.log.Infow("📊 [ExtractLayerVariables] Variables extraídas del QGS",
				"project", projectName,
				"qgsLayersWithVars", len(varsFromQgs))

			// Mostrar todas las variables extraídas del QGS
			lvm.log.Infow("🗂️ [ExtractLayerVariables] CAPAS DEL QGS CON VARIABLES:")
			for qgsLayerId, variables := range varsFromQgs {
				lvm.log.Infow("   🎯 QGS Capa",
					"qgsLayerId", qgsLayerId,
					"variables", variables)
			}

			// ✅ MAPEO DIRECTO - EL PARSER YA DEVUELVE LOS IDs CORRECTOS
			lvm.log.Infow("🔄 [ExtractLayerVariables] INICIANDO MAPEO DIRECTO")
			for qgsLayerId, variables := range varsFromQgs {
				// Verificar si existe en metadata
				if _, exists := metaLayers[qgsLayerId]; exists {
					// Filtrar solo las variables que empiezan por qV_search
					filteredVars := make(map[string]string)
					for key, value := range variables {
						if strings.HasPrefix(key, "qV_search") {
							filteredVars[key] = value
						}
					}

					// Solo agregar si hay variables filtradas
					if len(filteredVars) > 0 {
						layerVariables[qgsLayerId] = filteredVars
						lvm.log.Infow("✅ [ExtractLayerVariables] MATCH DIRECTO",
							"qgsLayerId", qgsLayerId,
							"variables", filteredVars)
					}
				} else {
					lvm.log.Warnw("⚠️ [ExtractLayerVariables] Capa QGS no encontrada en metadata",
						"qgsLayerId", qgsLayerId,
						"availableMetaIds", func() []string {
							ids := make([]string, 0, len(metaLayers))
							for id := range metaLayers {
								ids = append(ids, id)
							}
							return ids
						}())
				}
			}
		}
	} else {
		lvm.log.Warnw("⚠️ [ExtractLayerVariables] Repositorio no soporta GetLayerVariables",
			"project", projectName)
	}

	lvm.log.Infow("🏁 [ExtractLayerVariables] EXTRACCIÓN COMPLETADA",
		"project", projectName,
		"totalVariablesIntegradas", len(layerVariables))

	return layerVariables
}

// IntegrateVariablesIntoLayers integra las variables directamente en cada capa
func (lvm *LayerVariablesManager) IntegrateVariablesIntoLayers(
	layers []interface{},
	layerVariables map[string]interface{},
	metaLayers map[string]domain.LayerMeta,
) {
	lvm.log.Infow("🔧 [IntegrateVariablesIntoLayers] INICIANDO INTEGRACIÓN",
		"layersCount", len(layers),
		"variablesCount", len(layerVariables))

	// 🔍 DIAGNÓSTICO: Ver qué variables tenemos disponibles
	lvm.log.Infow("📊 [IntegrateVariablesIntoLayers] VARIABLES DISPONIBLES:")
	for varId, variables := range layerVariables {
		lvm.log.Infow("   🎯 Variable para capa",
			"varId", varId,
			"variables", variables)
	}

	// Integrar variables en cada capa
	for i, layer := range layers {
		lvm.log.Debugw("🔍 [IntegrateVariablesIntoLayers] Procesando item",
			"index", i,
			"itemType", fmt.Sprintf("%T", layer))

		switch l := layer.(type) {
		case map[string]interface{}:
			lvm.integrateVariablesInMap(l, layerVariables, i)
		case *OverlayLayer:
			lvm.integrateVariablesInOverlay(l, layerVariables, i)
		case OverlayLayer:
			lvm.integrateVariablesInOverlayValue(&l, layerVariables, i)
			layers[i] = l // Reasignar el valor modificado
		default:
			lvm.log.Debugw("🤷 [IntegrateVariablesIntoLayers] Tipo no soportado",
				"index", i,
				"type", fmt.Sprintf("%T", layer))
		}
	}

	lvm.log.Infow("🏁 [IntegrateVariablesIntoLayers] INTEGRACIÓN COMPLETADA")
}

// integrateVariablesInMap maneja capas como map[string]interface{}
func (lvm *LayerVariablesManager) integrateVariablesInMap(layerMap map[string]interface{}, layerVariables map[string]interface{}, index int) {
	qgisId, _ := layerMap["qgis_id"].(string)
	layerName, _ := layerMap["name"].(string)

	lvm.log.Infow("🗂️ [integrateVariablesInMap] Procesando capa map",
		"index", index,
		"layerName", layerName,
		"qgisId", qgisId)

	// Buscar variables para esta capa por qgis_id
	if variables, exists := layerVariables[qgisId]; exists {
		if varsMap, ok := variables.(map[string]string); ok {
			// Crear mapa para todas las variables qV_*
			qvVariables := make(map[string]string)

			// Procesar todas las variables que empiecen por qV_search
			for key, value := range varsMap {
				if strings.HasPrefix(key, "qV_search") {
					qvVariables[key] = value

					// Mantener compatibilidad con qV_search original
					if key == "qV_search" {
						layerMap["qV_search"] = value
					}
				}
			}

			// Agregar el mapa completo de variables si hay alguna
			if len(qvVariables) > 0 {
				layerMap["variables"] = qvVariables
				lvm.log.Infow("✅ [integrateVariablesInMap] Variables qV_* INTEGRADAS",
					"index", index,
					"layerName", layerName,
					"qgisId", qgisId,
					"variablesCount", len(qvVariables),
					"variables", qvVariables)
			}
		}
	}
}

// integrateVariablesInOverlay maneja capas OverlayLayer por puntero
func (lvm *LayerVariablesManager) integrateVariablesInOverlay(overlay *OverlayLayer, layerVariables map[string]interface{}, index int) {
	lvm.log.Infow("🗂️ [integrateVariablesInOverlay] Procesando OverlayLayer puntero",
		"index", index,
		"overlayName", overlay.Name,
		"overlayQgisId", overlay.QgisId)

	// Buscar variables para esta capa por QgisId
	if variables, exists := layerVariables[overlay.QgisId]; exists {
		if varsMap, ok := variables.(map[string]string); ok {
			// Crear mapa para todas las variables qV_*
			qvVariables := make(map[string]string)

			// Procesar todas las variables que empiecen por qV_search
			for key, value := range varsMap {
				if strings.HasPrefix(key, "qV_search") {
					qvVariables[key] = value

					// Mantener compatibilidad con qV_search original
					if key == "qV_search" {
						overlay.QVSearch = value
					}
				}
			}

			// Agregar el mapa completo de variables si hay alguna
			if len(qvVariables) > 0 {
				overlay.Variables = qvVariables
				lvm.log.Infow("✅ [integrateVariablesInOverlay] Variables qV_* INTEGRADAS",
					"index", index,
					"overlayName", overlay.Name,
					"overlayQgisId", overlay.QgisId,
					"variablesCount", len(qvVariables),
					"variables", qvVariables)
			}
		}
	}
}

// integrateVariablesInOverlayValue maneja capas OverlayLayer por valor
func (lvm *LayerVariablesManager) integrateVariablesInOverlayValue(overlay *OverlayLayer, layerVariables map[string]interface{}, index int) {
	lvm.log.Infow("🗂️ [integrateVariablesInOverlayValue] Procesando OverlayLayer valor",
		"index", index,
		"overlayName", overlay.Name,
		"overlayQgisId", overlay.QgisId,
		"overlayTitle", overlay.Title)

	actualQgisId := overlay.QgisId

	// ✅ SOLUCIÓN: Buscar el ID correcto por título cuando QgisId está vacío
	if actualQgisId == "" {
		// Recorrer layerVariables buscando coincidencia por título
		for varId, variables := range layerVariables {
			if varsMap, ok := variables.(map[string]string); ok {
				// Si la capa tiene una variable "layerName" que coincide con el título
				if layerName, hasLayerName := varsMap["layerName"]; hasLayerName && layerName == overlay.Title {
					actualQgisId = varId   // Usar este ID
					overlay.QgisId = varId // Y actualizarlo en la estructura
					lvm.log.Infow("🔧 ID encontrado por coincidencia de título",
						"layerTitle", overlay.Title,
						"foundId", varId)
					break
				}
			}
		}

		// Si aún no se encontró, buscar por coincidencia parcial o ID directo
		if actualQgisId == "" {
			for varId := range layerVariables {
				// Si el ID de la variable contiene el título de la capa
				if strings.Contains(varId, overlay.Title) ||
					strings.Contains(varId, strings.ReplaceAll(overlay.Title, " ", "_")) {
					actualQgisId = varId
					overlay.QgisId = varId
					lvm.log.Infow("🔧 ID encontrado por coincidencia parcial",
						"layerTitle", overlay.Title,
						"foundId", varId)
					break
				}
			}
		}
	}

	// Buscar variables para esta capa
	if variables, exists := layerVariables[actualQgisId]; exists {
		if varsMap, ok := variables.(map[string]string); ok {
			// Crear mapa para todas las variables qV_*
			qvVariables := make(map[string]string)

			// Procesar todas las variables que empiecen por qV_search
			for key, value := range varsMap {
				if strings.HasPrefix(key, "qV_search") {
					// ✅ NORMALIZAR VARIABLE PARA COMPATIBILIDAD
					normalizedValue := lvm.normalizeQVSearchValue(value, overlay.Title)
					qvVariables[key] = normalizedValue

					// Mantener compatibilidad con qV_search original
					if key == "qV_search" {
						overlay.QVSearch = normalizedValue
						lvm.log.Infow("✅ [integrateVariablesInOverlayValue] Variable qV_search INTEGRADA",
							"index", index,
							"overlayName", overlay.Name,
							"qgisId", overlay.QgisId,
							"originalValue", value,
							"normalizedValue", normalizedValue)
					}
				}
			}

			// Agregar el mapa completo de variables si hay alguna
			if len(qvVariables) > 0 {
				overlay.Variables = qvVariables
				lvm.log.Infow("✅ [integrateVariablesInOverlayValue] Variables qV_* INTEGRADAS",
					"index", index,
					"overlayName", overlay.Name,
					"overlayQgisId", overlay.QgisId,
					"variablesCount", len(qvVariables),
					"variables", qvVariables)
			}
		}
	}
}

// ✅ NUEVA FUNCIÓN: Normaliza la variable qV_search para compatibilidad con el frontend
func (lvm *LayerVariablesManager) normalizeQVSearchValue(value string, layerTitle string) string {
	lvm.log.Debugw("🔄 [normalizeQVSearch] Procesando valor",
		"originalValue", value,
		"layerTitle", layerTitle)

	// Si está vacío, usar valor por defecto
	if strings.TrimSpace(value) == "" {
		defaultValue := fmt.Sprintf("field=\"NAME\" fieldText=\"%s\" desc=\"Introduïu %s\"", layerTitle, layerTitle)
		lvm.log.Warnw("⚠️ [normalizeQVSearch] Valor vacío, usando valor por defecto",
			"defaultValue", defaultValue)
		return defaultValue
	}

	// Comprobar si tiene field (obligatorio)
	hasField := strings.Contains(value, "field=\"")
	if !hasField {
		// No tiene field, devolver valor por defecto
		defaultValue := fmt.Sprintf("field=\"NAME\" fieldText=\"%s\" desc=\"Introduïu %s\"", layerTitle, layerTitle)
		lvm.log.Warnw("⚠️ [normalizeQVSearch] No tiene campo field, usando valor por defecto",
			"originalValue", value,
			"defaultValue", defaultValue)
		return defaultValue
	}

	result := value

	// Convertir fieldtext a fieldText si existe
	if strings.Contains(result, "fieldtext=\"") {
		oldValue := result
		result = strings.Replace(result, "fieldtext=\"", "fieldText=\"", -1)
		lvm.log.Infow("🔄 [normalizeQVSearch] Convertido fieldtext a fieldText",
			"before", oldValue,
			"after", result)
	}

	// Si no tiene fieldText, extraer field y agregarlo
	if !strings.Contains(result, "fieldText=\"") {
		// Extraer el valor del campo field
		fieldRegex := regexp.MustCompile(`field="([^"]+)"`)
		fieldMatch := fieldRegex.FindStringSubmatch(result)

		fieldValue := ""
		if len(fieldMatch) >= 2 {
			fieldValue = fieldMatch[1]
		}

		// Usar field como fieldText o el título de la capa si está vacío
		fieldTextValue := fieldValue
		if strings.TrimSpace(fieldTextValue) == "" {
			fieldTextValue = layerTitle
		}

		oldValue := result
		result = result + fmt.Sprintf(" fieldText=\"%s\"", fieldTextValue)
		lvm.log.Infow("➕ [normalizeQVSearch] Agregado fieldText que faltaba",
			"before", oldValue,
			"after", result,
			"fieldValue", fieldValue,
			"fieldTextValue", fieldTextValue)
	}

	// Si no tiene desc, extraer fieldText y agregarlo
	if !strings.Contains(result, "desc=\"") {
		// Extraer fieldText para usarlo en la descripción
		fieldTextRegex := regexp.MustCompile(`fieldText="([^"]+)"`)
		fieldTextMatch := fieldTextRegex.FindStringSubmatch(result)

		descValue := layerTitle
		if len(fieldTextMatch) >= 2 && fieldTextMatch[1] != "" {
			descValue = fieldTextMatch[1]
		}

		oldValue := result
		result = result + fmt.Sprintf(" desc=\"Introduïu %s\"", descValue)
		lvm.log.Infow("➕ [normalizeQVSearch] Agregada descripción que faltaba",
			"before", oldValue,
			"after", result,
			"descValue", fmt.Sprintf("Introduïu %s", descValue))
	}

	lvm.log.Infow("✅ [normalizeQVSearch] Normalización completada",
		"originalValue", value,
		"normalizedValue", result)

	return result
}
