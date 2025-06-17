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
	type Option struct {
		Name    string   `xml:"name,attr"`
		Value   string   `xml:"value,attr"`
		Options []Option `xml:"Option"`
	}
	type CustomProperties struct {
		Options []Option `xml:"Option"`
	}
	type MapLayer struct {
		ID               string           `xml:"id"`
		LayerName        string           `xml:"layername"`
		CustomProperties CustomProperties `xml:"customproperties"`
	}
	type ProjectLayers struct {
		Layers []MapLayer `xml:"maplayer"`
	}
	type Qgis struct {
		ProjectLayers ProjectLayers `xml:"projectlayers"`
	}
	var doc Qgis
	if err := xml.Unmarshal(content, &doc); err != nil {
		return nil, fmt.Errorf("error parseando XML: %w", err)
	}
	varsMap := make(map[string]map[string]string)
	for _, layer := range doc.ProjectLayers.Layers {
		varNames := []string{}
		varValues := []string{}
		for _, opt := range layer.CustomProperties.Options {
			if opt.Name == "variableNames" {
				for _, o := range opt.Options {
					varNames = append(varNames, o.Value)
				}
			}
			if opt.Name == "variableValues" {
				for _, o := range opt.Options {
					varValues = append(varValues, o.Value)
				}
			}
		}
		for i, name := range varNames {
			if name == "qV_search" && i < len(varValues) {
				if varsMap[layer.ID] == nil {
					varsMap[layer.ID] = make(map[string]string)
				}
				varsMap[layer.ID]["qV_search"] = varValues[i]
			}
		}
	}
	return varsMap, nil
}
