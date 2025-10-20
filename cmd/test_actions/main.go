package main

import (
	"fmt"

	"github.com/gisquick/gisquick-server/internal/infrastructure/project"
	"go.uber.org/zap"
)

func main() {
	// Configurar logger
	logger, _ := zap.NewDevelopment()
	sugar := logger.Sugar()
	defer logger.Sync()

	// Directorio que contiene el archivo
	projectsRoot := "c:\\Users\\jfontan\\OneDrive - CONSULTORIA TECNICA NEXUS GEOGRAFICS SL\\AM DIBC\\2025 AM SIT OF20252510\\3. Tasques\\QVista-Web\\PROVA_ACCIO"

	// Nombre del proyecto y archivo
	projectName := "PlaViure15 - copia"
	projectFile := "PlaViure15.qgs"

	// Crear parser
	parser := project.NewQgisParser(sugar, projectsRoot)

	// Usar el método público
	varsMap, layerActions, err := parser.ExtractLayerVariables(projectName, projectFile)
	if err != nil {
		sugar.Fatalf("Error al extraer variables y acciones: %v", err)
	}

	// Mostrar resultados
	fmt.Printf("Variables encontradas: %d\n", len(varsMap))
	fmt.Printf("Acciones encontradas: %d\n", len(layerActions))

	// Mostrar detalles de cada acción
	for i, action := range layerActions {
		fmt.Printf("\nAcción #%d:\n", i+1)
		fmt.Printf("  ID: %s\n", action.Id)
		fmt.Printf("  Nombre: %s\n", action.Name)
		fmt.Printf("  Texto: %s\n", action.ActionText)
		fmt.Printf("  Capa ID: %s\n", action.LayerId)
	}
}
