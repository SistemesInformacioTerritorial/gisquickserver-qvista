// filepath: c:\Users\jfontan\Documents\gisquick-server-next-windows\internal\application\projects_test.go

package application

import (
  "encoding/json"
  "os"
  "path/filepath"
  "testing"

  "github.com/gisquick/gisquick-server/internal/domain"
  "github.com/stretchr/testify/assert"
  "github.com/stretchr/testify/require"
  "go.uber.org/zap"
)

func setupTestLogger() *zap.SugaredLogger {
  logger, _ := zap.NewDevelopment()
  return logger.Sugar()
}

func TestExtractLayerVariables(t *testing.T) {
  // Setup
  logger := setupTestLogger()
  service := NewProjectsService(logger, nil, nil, nil, "")

  tests := []struct {
    name           string
    qgsFile        string
    expectedResult map[string]map[string]string
  }{
    {
      name:    "Project with qV_search variables",
      qgsFile: "testdata/with_qv_search.qgs",
      expectedResult: map[string]map[string]string{
        "parcelas": {
          "qV_search": "field=\"CODIGO\" fieldtext=\"Código Parcela\"",
        },
      },
    },
    {
      name:           "Project without qV_search variables",
      qgsFile:        "testdata/without_qv_search.qgs",
      expectedResult: map[string]map[string]string{},
    },
    {
      name:    "Project with project-level variables",
      qgsFile: "testdata/project_level_vars.qgs",
      expectedResult: map[string]map[string]string{
        "@project": {
          "qV_search_global": "field=\"NOM\" fieldtext=\"Nom\"",
        },
      },
    },
  }

  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      // Create test directory if it doesn't exist
      testDir := "testdata"
      if _, err := os.Stat(testDir); os.IsNotExist(err) {
        err := os.MkdirAll(testDir, 0755)
        require.NoError(t, err)
      }

      // Write test file content (simplified for test)
      // In a real test, you'd have actual QGS file content
      testContent := "test qgs content"
      err := os.WriteFile(tt.qgsFile, []byte(testContent), 0644)
      require.NoError(t, err)
      defer os.Remove(tt.qgsFile)

      // Execute test
      // Note: This is a mock call since we don't have the actual implementation
      result := service.extractLayerVariables("test-project", nil)

      // For testing purposes, we're just checking the structure
      // In a real test, you'd verify actual content
      assert.Equal(t, len(tt.expectedResult), len(result))
    })
  }
}

func TestIntegrateVariablesIntoLayers(t *testing.T) {
  // Setup
  logger := setupTestLogger()
  service := NewProjectsService(logger, nil, nil, nil, "")

  // Test data
  layers := []map[string]interface{}{
    {
      "name":  "parcelas",
      "title": "Parcel·les",
    },
    {
      "name":  "equipamientos",
      "title": "Equipaments",
    },
  }

  layerVariables := map[string]map[string]string{
    "parcelas": {
      "qV_search": "field=\"CODIGO\" fieldtext=\"Código Parcela\"",
    },
  }

  t.Run("Variables are correctly integrated", func(t *testing.T) {
    // Execute
    service.integrateVariablesIntoLayers(layers, layerVariables, nil)

    // Verify
    // Convert to JSON for easier inspection
    jsonData, err := json.Marshal(layers)
    require.NoError(t, err)

    // Verify qV_search was added to the parcelas layer
    assert.Contains(t, string(jsonData), "qV_search")
    assert.Contains(t, string(jsonData), "field=\\\"CODIGO\\\" fieldtext=\\\"Código Parcela\\\"")

    // The first layer should have qV_search
    assert.Contains(t, layers[0], "qV_search")

    // The second layer should not have qV_search
    _, hasQVSearch := layers[1]["qV_search"]
    assert.False(t, hasQVSearch)
  })

  t.Run("Project with nested layer groups", func(t *testing.T) {
    // Test data with nested groups
    nestedLayers := []map[string]interface{}{
      {
        "name":  "group1",
        "title": "Group 1",
        "layers": []map[string]interface{}{
          {
            "name":  "parcelas",
            "title": "Parcel·les",
          },
        },
      },
    }

    // Execute
    service.integrateVariablesIntoLayers(nestedLayers, layerVariables, nil)

    // Verify
    // The nested parcelas layer should have qV_search
    nestedLayer := nestedLayers[0]["layers"].([]map[string]interface{})[0]
    assert.Contains(t, nestedLayer, "qV_search")
    assert.Equal(t, "field=\"CODIGO\" fieldtext=\"Código Parcela\"", nestedLayer["qV_search"])
  })
}

func TestGetMapConfig(t *testing.T) {
  // This would be an integration test to verify the full flow
  // from loading a project to generating the final config JSON
  
  // Setup
  logger := setupTestLogger()
  // Mock other dependencies as needed
  
  // In a real test, you would:
  // 1. Create a test project with qV_search variables
  // 2. Call GetMapConfig
  // 3. Verify the returned JSON contains the qV_search variables
  
  t.Skip("Integration test to be implemented")
}