// A internal/infrastructure/project/qgis_parser.go (nuevo archivo)
package project

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

// QgisParser extrae información directamente de archivos QGS/QGZ
type QgisParser struct {
	log          *zap.SugaredLogger
	ProjectsRoot string
}

// NewQgisParser crea un nuevo parser de archivos QGIS
func NewQgisParser(log *zap.SugaredLogger, projectsRoot string) *QgisParser {
	return &QgisParser{
		log:          log,
		ProjectsRoot: projectsRoot,
	}
}

// ExtractLayerVariables obtiene las variables de capas de un archivo QGS/QGZ
func (p *QgisParser) ExtractLayerVariables(projectName, projectFile string) (map[string]map[string]string, error) {
	p.log.Infow("🔍 Extrayendo variables directamente del QGS/QGZ", "project", projectName, "file", projectFile)

	// Ruta completa al archivo
	projectPath := filepath.Join(p.ProjectsRoot, projectName, projectFile)

	var qgsContent []byte
	var err error

	// Determinar si es QGZ o QGS
	if strings.HasSuffix(strings.ToLower(projectFile), ".qgz") {
		// Extraer QGS del archivo QGZ (ZIP)
		qgsContent, err = p.extractQgsFromQgz(projectPath)
	} else {
		// Leer archivo QGS directamente
		qgsContent, err = ioutil.ReadFile(projectPath)
	}

	if err != nil {
		return nil, fmt.Errorf("error leyendo archivo de proyecto: %w", err)
	}

	// Parsear variables del XML
	return p.parseQgsXml(qgsContent, projectName)
}

// extractQgsFromQgz extrae el archivo QGS de un archivo QGZ
func (p *QgisParser) extractQgsFromQgz(qgzPath string) ([]byte, error) {
	zipReader, err := zip.OpenReader(qgzPath)
	if err != nil {
		return nil, err
	}
	defer zipReader.Close()

	// Buscar archivo .qgs dentro del zip
	for _, file := range zipReader.File {
		if strings.HasSuffix(strings.ToLower(file.Name), ".qgs") {
			rc, err := file.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()

			return ioutil.ReadAll(rc)
		}
	}

	return nil, fmt.Errorf("no se encontró archivo QGS dentro del QGZ")
}

// parseQgsXml parsea el XML de QGS y extrae las variables de capa
func (p *QgisParser) parseQgsXml(content []byte, projectName string) (map[string]map[string]string, error) {
	p.log.Infow("📋 [parseQgsXml] Iniciando parseo de XML", "project", projectName, "xmlSize", len(content))

	type Option struct {
		Name    string   `xml:"name,attr"`
		Value   string   `xml:"value,attr"`
		Type    string   `xml:"type,attr"`
		Options []Option `xml:"Option"`
	}
	type CustomProperties struct {
		Options []Option `xml:"Option"`
	}
	type MapLayer struct {
		ID               string           `xml:"id,attr"`
		LayerName        string           `xml:"layername"`
		CustomProperties CustomProperties `xml:"customproperties"`
	}
	type ProjectLayers struct {
		Layers []MapLayer `xml:"maplayer"`
	}

	// ✅ NUEVA ESTRUCTURA PARA LAYER TREE
	type LayerTreeLayer struct {
		ID   string `xml:"id,attr"`
		Name string `xml:"name,attr"`
	}
	type LayerTreeGroup struct {
		Layers []LayerTreeLayer `xml:"layer-tree-layer"`
		Groups []LayerTreeGroup `xml:"layer-tree-group"`
	}
	type LayerTree struct {
		Groups []LayerTreeGroup `xml:"layer-tree-group"`
		Layers []LayerTreeLayer `xml:"layer-tree-layer"`
	}

	type Qgis struct {
		ProjectLayers ProjectLayers `xml:"projectlayers"`
		LayerTree     LayerTree     `xml:"layer-tree-group"`
	}

	var doc Qgis
	if err := xml.Unmarshal(content, &doc); err != nil {
		return nil, fmt.Errorf("error parseando XML: %w", err)
	}

	p.log.Infow("🔍 [parseQgsXml] XML parseado correctamente",
		"project", projectName,
		"totalLayers", len(doc.ProjectLayers.Layers))

	// ✅ CREAR MAPEO NOMBRE -> ID DESDE LAYER TREE
	nameToIdMap := make(map[string]string)

	// Procesar capas directas
	for _, layer := range doc.LayerTree.Layers {
		nameToIdMap[layer.Name] = layer.ID
		p.log.Debugw("🗂️ [parseQgsXml] Mapping directo",
			"layerName", layer.Name,
			"layerId", layer.ID)
	}

	// Procesar grupos recursivamente
	var processGroup func(group LayerTreeGroup)
	processGroup = func(group LayerTreeGroup) {
		for _, layer := range group.Layers {
			nameToIdMap[layer.Name] = layer.ID
			p.log.Debugw("🗂️ [parseQgsXml] Mapping desde grupo",
				"layerName", layer.Name,
				"layerId", layer.ID)
		}
		for _, subGroup := range group.Groups {
			processGroup(subGroup)
		}
	}

	for _, group := range doc.LayerTree.Groups {
		processGroup(group)
	}

	varsMap := make(map[string]map[string]string)

	for _, layer := range doc.ProjectLayers.Layers {
		// ✅ RESOLVER ID USANDO LAYER TREE
		layerId := layer.ID
		if layerId == "" {
			if treeId, exists := nameToIdMap[layer.LayerName]; exists {
				layerId = treeId
				p.log.Infow("🔧 [parseQgsXml] ID resuelto desde layer-tree",
					"layerName", layer.LayerName,
					"resolvedId", treeId)
			} else {
				p.log.Warnw("⚠️ [parseQgsXml] No se pudo resolver ID",
					"layerName", layer.LayerName,
					"availableNames", func() []string {
						names := make([]string, 0, len(nameToIdMap))
						for name := range nameToIdMap {
							names = append(names, name)
						}
						return names
					}())
				continue
			}
		}

		p.log.Infow("🗂️ [parseQgsXml] Procesando capa del QGS",
			"project", projectName,
			"layerId", layerId,
			"layerName", layer.LayerName)

		varNames := []string{}
		varValues := []string{}

		// Buscar Option type="Map" (QGIS 3.x)
		for _, opt := range layer.CustomProperties.Options {
			if opt.Type == "Map" {
				for _, subopt := range opt.Options {
					if subopt.Name == "variableNames" {
						for _, o := range subopt.Options {
							varNames = append(varNames, o.Value)
						}
					}
					if subopt.Name == "variableValues" {
						for _, o := range subopt.Options {
							varValues = append(varValues, o.Value)
						}
					}
				}
			}
		}

		p.log.Debugw("📊 [parseQgsXml] Variables encontradas en capa",
			"project", projectName,
			"layerId", layerId,
			"layerName", layer.LayerName,
			"varNames", varNames,
			"varValues", varValues)

		// Procesar variables encontradas
		for i, name := range varNames {
			if name == "qV_search" && i < len(varValues) {
				if varsMap[layerId] == nil {
					varsMap[layerId] = make(map[string]string)
				}
				varsMap[layerId]["qV_search"] = varValues[i]
				varsMap[layerId]["layerName"] = layer.LayerName

				p.log.Infow("✅ [parseQgsXml] Variable qV_search encontrada!",
					"project", projectName,
					"qgsLayerId", layerId,
					"qgsLayerName", layer.LayerName,
					"qV_search", varValues[i])
			}
		}
	}

	p.log.Infow("🏁 [parseQgsXml] Parseo completado",
		"project", projectName,
		"layersWithQVSearch", len(varsMap))

	return varsMap, nil
}
