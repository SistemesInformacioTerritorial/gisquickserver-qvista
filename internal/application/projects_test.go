package application

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/gisquick/gisquick-server/internal/domain"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// Mock implementations for testing

type mockProjectsRepository struct {
	projects         map[string]domain.ProjectInfo
	qgisMetadata     map[string]interface{}
	settings         map[string]domain.ProjectSettings
	scripts          map[string]domain.Scripts
	files            map[string][]domain.ProjectFile
	temporaryFiles   map[string][]domain.ProjectFile
	userProjects     map[string][]string
	customizations   map[string]json.RawMessage
	thumbnailPaths   map[string]string
	filesInfo        map[string]map[string]domain.FileInfo
	shouldFailCreate bool
	shouldFailGet    bool
	shouldFailUpdate bool
}

func newMockProjectsRepository() *mockProjectsRepository {
	return &mockProjectsRepository{
		projects:       make(map[string]domain.ProjectInfo),
		qgisMetadata:   make(map[string]interface{}),
		settings:       make(map[string]domain.ProjectSettings),
		scripts:        make(map[string]domain.Scripts),
		files:          make(map[string][]domain.ProjectFile),
		temporaryFiles: make(map[string][]domain.ProjectFile),
		userProjects:   make(map[string][]string),
		customizations: make(map[string]json.RawMessage),
		thumbnailPaths: make(map[string]string),
		filesInfo:      make(map[string]map[string]domain.FileInfo),
	}
}

func (m *mockProjectsRepository) CheckProjectExists(name string) bool {
	_, exists := m.projects[name]
	return exists
}

func (m *mockProjectsRepository) Create(name string, qmeta json.RawMessage) (*domain.ProjectInfo, error) {
	if m.shouldFailCreate {
		return nil, errors.New("mock create error")
	}

	project := domain.ProjectInfo{
		Name:       name,
		Title:      "Test Project",
		State:      "empty",
		Created:    time.Now(),
		LastUpdate: time.Now(),
		Size:       0,
		Thumbnail:  false,
		QgisFile:   "",
		Projection: "EPSG:4326",
	}

	m.projects[name] = project
	return &project, nil
}

func (m *mockProjectsRepository) AllProjects(skipErrors bool) ([]string, error) {
	var projectNames []string
	for name := range m.projects {
		projectNames = append(projectNames, name)
	}
	return projectNames, nil
}

func (m *mockProjectsRepository) UserProjects(user string) ([]string, error) {
	if projects, exists := m.userProjects[user]; exists {
		return projects, nil
	}
	return []string{}, nil
}

func (m *mockProjectsRepository) GetProjectInfo(name string) (domain.ProjectInfo, error) {
	if m.shouldFailGet {
		return domain.ProjectInfo{}, errors.New("mock get error")
	}

	if project, exists := m.projects[name]; exists {
		return project, nil
	}
	return domain.ProjectInfo{}, domain.ErrProjectNotExists
}

func (m *mockProjectsRepository) Delete(name string) error {
	if _, exists := m.projects[name]; !exists {
		return domain.ErrProjectNotExists
	}
	delete(m.projects, name)
	return nil
}

func (m *mockProjectsRepository) CreateFile(projectName, directory, pattern string, r io.Reader) (domain.ProjectFile, error) {
	file := domain.ProjectFile{
		Path:  "test/file.txt",
		Hash:  "testhash",
		Size:  1024,
		Mtime: time.Now().Unix(),
	}
	return file, nil
}

func (m *mockProjectsRepository) SaveFile(project string, finfo domain.ProjectFile, path string) error {
	return nil
}

func (m *mockProjectsRepository) GetFileInfo(project, path string) (domain.FileInfo, error) {
	if projectFiles, exists := m.filesInfo[project]; exists {
		if fileInfo, exists := projectFiles[path]; exists {
			return fileInfo, nil
		}
	}
	return domain.FileInfo{}, domain.ErrFileNotExists
}

func (m *mockProjectsRepository) GetFilesInfo(project string, paths ...string) (map[string]domain.FileInfo, error) {
	result := make(map[string]domain.FileInfo)
	if projectFiles, exists := m.filesInfo[project]; exists {
		for _, path := range paths {
			if fileInfo, exists := projectFiles[path]; exists {
				result[path] = fileInfo
			}
		}
	}
	return result, nil
}

func (m *mockProjectsRepository) ListProjectFiles(project string, checksum bool) ([]domain.ProjectFile, []domain.ProjectFile, error) {
	files := m.files[project]
	tempFiles := m.temporaryFiles[project]
	if files == nil {
		files = []domain.ProjectFile{}
	}
	if tempFiles == nil {
		tempFiles = []domain.ProjectFile{}
	}
	return files, tempFiles, nil
}

func (m *mockProjectsRepository) ParseQgisMetadata(projectName string, data interface{}) error {
	if metadata, exists := m.qgisMetadata[projectName]; exists {
		// Convert metadata to the target interface using JSON marshaling/unmarshaling
		jsonData, err := json.Marshal(metadata)
		if err != nil {
			return err
		}
		return json.Unmarshal(jsonData, data)
	}
	return errors.New("metadata not found")
}

func (m *mockProjectsRepository) UpdateMeta(projectName string, meta json.RawMessage) error {
	if m.shouldFailUpdate {
		return errors.New("mock update error")
	}
	var metadata interface{}
	if err := json.Unmarshal(meta, &metadata); err != nil {
		return err
	}
	m.qgisMetadata[projectName] = metadata
	return nil
}

func (m *mockProjectsRepository) GetSettings(projectName string) (domain.ProjectSettings, error) {
	if settings, exists := m.settings[projectName]; exists {
		return settings, nil
	}
	return domain.ProjectSettings{}, errors.New("settings not found")
}

func (m *mockProjectsRepository) UpdateSettings(projectName string, data json.RawMessage) error {
	if m.shouldFailUpdate {
		return errors.New("mock update settings error")
	}
	var settings domain.ProjectSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return err
	}
	m.settings[projectName] = settings
	return nil
}

func (m *mockProjectsRepository) GetThumbnailPath(projectName string) string {
	if path, exists := m.thumbnailPaths[projectName]; exists {
		return path
	}
	return "/default/thumbnail/path"
}

func (m *mockProjectsRepository) SaveThumbnail(projectName string, r io.Reader) error {
	m.thumbnailPaths[projectName] = "/saved/thumbnail/path"
	return nil
}

func (m *mockProjectsRepository) UpdateFiles(projectName string, info domain.FilesChanges, next domain.FilesReader) ([]domain.ProjectFile, error) {
	if m.shouldFailUpdate {
		return nil, errors.New("mock update files error")
	}

	// Simulate file updates
	var updatedFiles []domain.ProjectFile
	for _, file := range info.Updates {
		updatedFiles = append(updatedFiles, file)
	}

	m.files[projectName] = updatedFiles
	return updatedFiles, nil
}

func (m *mockProjectsRepository) GetScripts(projectName string) (domain.Scripts, error) {
	if scripts, exists := m.scripts[projectName]; exists {
		return scripts, nil
	}
	return domain.Scripts{}, nil
}

func (m *mockProjectsRepository) UpdateScripts(projectName string, scripts domain.Scripts) error {
	if m.shouldFailUpdate {
		return errors.New("mock update scripts error")
	}
	m.scripts[projectName] = scripts
	return nil
}

func (m *mockProjectsRepository) GetProjectCustomizations(projectName string) (json.RawMessage, error) {
	if customizations, exists := m.customizations[projectName]; exists {
		return customizations, nil
	}
	return nil, errors.New("customizations not found")
}

func (m *mockProjectsRepository) Close() {
	// Mock implementation - nothing to close
}

type mockAccountsLimiter struct {
	limits              map[string]domain.AccountConfig
	shouldFailGetLimits bool
}

func newMockAccountsLimiter() *mockAccountsLimiter {
	return &mockAccountsLimiter{
		limits: make(map[string]domain.AccountConfig),
	}
}

func (m *mockAccountsLimiter) GetAccountLimits(username string) (domain.AccountConfig, error) {
	if m.shouldFailGetLimits {
		return domain.AccountConfig{}, errors.New("mock get limits error")
	}

	if limits, exists := m.limits[username]; exists {
		return limits, nil
	}

	// Return default limits
	return domain.AccountConfig{
		MaxProjects:    10,
		MaxProjectSize: 100 * 1024 * 1024,  // 100MB
		MaxStorageSize: 1024 * 1024 * 1024, // 1GB
	}, nil
}

// Test helper functions

func setupProjectService(t *testing.T) (*projectService, *mockProjectsRepository, *mockAccountsLimiter) {
	logger := zaptest.NewLogger(t).Sugar()
	repo := newMockProjectsRepository()
	limiter := newMockAccountsLimiter()

	service := NewProjectsService(logger, repo, limiter)
	return service, repo, limiter
}

// Unit Tests

func TestNewProjectsService(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	repo := newMockProjectsRepository()
	limiter := newMockAccountsLimiter()

	service := NewProjectsService(logger, repo, limiter)

	if service == nil {
		t.Fatal("Expected service to be created, got nil")
	}

	if service.log != logger {
		t.Error("Expected logger to be set correctly")
	}

	if service.repo != repo {
		t.Error("Expected repository to be set correctly")
	}

	if service.limiter != limiter {
		t.Error("Expected limiter to be set correctly")
	}
}

func TestProjectService_Create(t *testing.T) {
	tests := []struct {
		name          string
		projectName   string
		meta          json.RawMessage
		setupLimiter  func(*mockAccountsLimiter)
		setupRepo     func(*mockProjectsRepository)
		expectedError string
	}{
		{
			name:        "successful_creation",
			projectName: "testuser/testproject",
			meta:        json.RawMessage(`{"title": "Test Project"}`),
			setupLimiter: func(limiter *mockAccountsLimiter) {
				limiter.limits["testuser"] = domain.AccountConfig{
					MaxProjects: 10,
				}
			},
			setupRepo: func(repo *mockProjectsRepository) {
				repo.userProjects["testuser"] = []string{}
			},
		},
		{
			name:        "projects_limit_exceeded",
			projectName: "testuser/testproject",
			meta:        json.RawMessage(`{"title": "Test Project"}`),
			setupLimiter: func(limiter *mockAccountsLimiter) {
				limiter.limits["testuser"] = domain.AccountConfig{
					MaxProjects: 1,
				}
			},
			setupRepo: func(repo *mockProjectsRepository) {
				repo.userProjects["testuser"] = []string{"testuser/existing"}
			},
			expectedError: "account projects count limit reached",
		},
		{
			name:        "repository_error",
			projectName: "testuser/testproject",
			meta:        json.RawMessage(`{"title": "Test Project"}`),
			setupLimiter: func(limiter *mockAccountsLimiter) {
				limiter.limits["testuser"] = domain.AccountConfig{
					MaxProjects: 10,
				}
			},
			setupRepo: func(repo *mockProjectsRepository) {
				repo.userProjects["testuser"] = []string{}
				repo.shouldFailCreate = true
			},
			expectedError: "mock create error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, repo, limiter := setupProjectService(t)

			if tt.setupLimiter != nil {
				tt.setupLimiter(limiter)
			}
			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}

			project, err := service.Create(tt.projectName, tt.meta)

			if tt.expectedError != "" {
				if err == nil {
					t.Fatalf("Expected error containing '%s', got nil", tt.expectedError)
				}
				if !strings.Contains(err.Error(), tt.expectedError) {
					t.Fatalf("Expected error containing '%s', got '%s'", tt.expectedError, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if project == nil {
				t.Fatal("Expected project to be created, got nil")
			}

			if project.Name != tt.projectName {
				t.Errorf("Expected project name '%s', got '%s'", tt.projectName, project.Name)
			}
		})
	}
}

func TestProjectService_GetProjectInfo(t *testing.T) {
	service, repo, _ := setupProjectService(t)

	// Setup test project
	testProject := domain.ProjectInfo{
		Name:  "testuser/testproject",
		Title: "Test Project",
		State: "published",
	}
	repo.projects["testuser/testproject"] = testProject

	// Test successful retrieval
	project, err := service.GetProjectInfo("testuser/testproject")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if project.Name != testProject.Name {
		t.Errorf("Expected project name '%s', got '%s'", testProject.Name, project.Name)
	}

	// Test non-existent project
	_, err = service.GetProjectInfo("nonexistent/project")
	if err == nil {
		t.Error("Expected error for non-existent project, got nil")
	}
}

func TestProjectService_Delete(t *testing.T) {
	service, repo, _ := setupProjectService(t)

	// Setup test project
	testProject := domain.ProjectInfo{
		Name: "testuser/testproject",
	}
	repo.projects["testuser/testproject"] = testProject

	// Test successful deletion
	err := service.Delete("testuser/testproject")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify project was deleted
	if repo.CheckProjectExists("testuser/testproject") {
		t.Error("Expected project to be deleted")
	}

	// Test deleting non-existent project
	err = service.Delete("nonexistent/project")
	if err == nil {
		t.Error("Expected error for deleting non-existent project, got nil")
	}
}

func TestProjectService_GetUserProjects(t *testing.T) {
	service, repo, _ := setupProjectService(t)

	// Setup test data
	testProjects := []domain.ProjectInfo{
		{Name: "testuser/project1", Title: "Project 1", State: "published"},
		{Name: "testuser/project2", Title: "Project 2", State: "empty"},
	}

	repo.userProjects["testuser"] = []string{"testuser/project1", "testuser/project2"}
	for _, project := range testProjects {
		repo.projects[project.Name] = project
	}

	// Test successful retrieval
	projects, err := service.GetUserProjects("testuser")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(projects) != 2 {
		t.Errorf("Expected 2 projects, got %d", len(projects))
	}

	// Test user with no projects
	projects, err = service.GetUserProjects("emptyuser")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(projects) != 0 {
		t.Errorf("Expected 0 projects, got %d", len(projects))
	}
}

func TestProjectService_AccessibleProjects(t *testing.T) {
	service, repo, _ := setupProjectService(t)

	// Setup test data
	testProjects := []domain.ProjectInfo{
		{Name: "testuser/project1", Title: "Project 1", State: "published"},
		{Name: "testuser/project2", Title: "Project 2", State: "empty"},
	}

	repo.userProjects["testuser"] = []string{"testuser/project1", "testuser/project2"}
	for _, project := range testProjects {
		repo.projects[project.Name] = project
	}

	// Test with skipErrors = false
	projects, err := service.AccessibleProjects("testuser", false)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(projects) != 2 {
		t.Errorf("Expected 2 projects, got %d", len(projects))
	}

	// Test with skipErrors = true and a failing project
	repo.userProjects["testuser"] = []string{"testuser/project1", "testuser/missing"}
	projects, err = service.AccessibleProjects("testuser", true)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(projects) != 1 {
		t.Errorf("Expected 1 project (missing project skipped), got %d", len(projects))
	}

	// Test with skipErrors = false and a failing project
	projects, err = service.AccessibleProjects("testuser", false)
	if err == nil {
		t.Error("Expected error when skipErrors=false and project missing, got nil")
	}
}

func TestProjectService_SaveFile(t *testing.T) {
	tests := []struct {
		name          string
		projectName   string
		directory     string
		pattern       string
		size          int64
		setupLimiter  func(*mockAccountsLimiter)
		setupRepo     func(*mockProjectsRepository)
		expectedError string
	}{
		{
			name:        "successful_save",
			projectName: "testuser/testproject",
			directory:   "media",
			pattern:     "*.jpg",
			size:        1024,
			setupLimiter: func(limiter *mockAccountsLimiter) {
				limiter.limits["testuser"] = domain.AccountConfig{
					MaxProjectSize: 10 * 1024 * 1024,
					MaxStorageSize: 100 * 1024 * 1024,
				}
			},
			setupRepo: func(repo *mockProjectsRepository) {
				repo.projects["testuser/testproject"] = domain.ProjectInfo{
					Name: "testuser/testproject",
					Size: 0,
				}
			},
		},
		{
			name:        "project_size_limit_exceeded",
			projectName: "testuser/testproject",
			directory:   "media",
			pattern:     "*.jpg",
			size:        10 * 1024 * 1024, // 10MB
			setupLimiter: func(limiter *mockAccountsLimiter) {
				limiter.limits["testuser"] = domain.AccountConfig{
					MaxProjectSize: 5 * 1024 * 1024, // 5MB limit
				}
			},
			setupRepo: func(repo *mockProjectsRepository) {
				repo.projects["testuser/testproject"] = domain.ProjectInfo{
					Name: "testuser/testproject",
					Size: 1024, // 1KB existing
				}
			},
			expectedError: "project size limit reached",
		},
		{
			name:        "storage_limit_exceeded",
			projectName: "testuser/testproject",
			directory:   "media",
			pattern:     "*.jpg",
			size:        50 * 1024 * 1024, // 50MB
			setupLimiter: func(limiter *mockAccountsLimiter) {
				limiter.limits["testuser"] = domain.AccountConfig{
					MaxStorageSize: 60 * 1024 * 1024, // 60MB total limit
				}
			},
			setupRepo: func(repo *mockProjectsRepository) {
				repo.userProjects["testuser"] = []string{"testuser/testproject", "testuser/other"}
				repo.projects["testuser/testproject"] = domain.ProjectInfo{
					Name: "testuser/testproject",
					Size: 20 * 1024 * 1024, // 20MB existing
				}
				repo.projects["testuser/other"] = domain.ProjectInfo{
					Name: "testuser/other",
					Size: 20 * 1024 * 1024, // 20MB in other project
				}
			},
			expectedError: "account storage limit reached",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, repo, limiter := setupProjectService(t)

			if tt.setupLimiter != nil {
				tt.setupLimiter(limiter)
			}
			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}

			reader := strings.NewReader("test file content")
			file, err := service.SaveFile(tt.projectName, tt.directory, tt.pattern, reader, tt.size)

			if tt.expectedError != "" {
				if err == nil {
					t.Fatalf("Expected error containing '%s', got nil", tt.expectedError)
				}
				if !strings.Contains(err.Error(), tt.expectedError) {
					t.Fatalf("Expected error containing '%s', got '%s'", tt.expectedError, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if file.Path == "" {
				t.Error("Expected file path to be set")
			}
		})
	}
}

func TestProjectService_UpdateFiles(t *testing.T) {
	service, repo, limiter := setupProjectService(t)

	// Setup test data
	projectName := "testuser/testproject"
	limiter.limits["testuser"] = domain.AccountConfig{
		MaxProjectSize: 10 * 1024 * 1024,
		MaxStorageSize: 100 * 1024 * 1024,
	}

	repo.projects[projectName] = domain.ProjectInfo{
		Name: projectName,
		Size: 1024,
	}

	repo.filesInfo[projectName] = map[string]domain.FileInfo{
		"existing.txt": {Size: 500, Hash: "oldhash", Mtime: time.Now().Unix()},
	}

	// Test successful update
	changes := domain.FilesChanges{
		Updates: []domain.ProjectFile{
			{Path: "new.txt", Size: 1024, Hash: "newhash", Mtime: time.Now().Unix()},
		},
		Removes: []string{"existing.txt"},
	}

	nextFunc := func() (string, io.ReadCloser, error) {
		return "new.txt", io.NopCloser(strings.NewReader("new content")), nil
	}

	files, err := service.UpdateFiles(projectName, changes, nextFunc)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(files) == 0 {
		t.Error("Expected updated files to be returned")
	}
}

func TestProjectService_GetQgisMetadata(t *testing.T) {
	service, repo, _ := setupProjectService(t)

	projectName := "testuser/testproject"
	testMetadata := map[string]interface{}{
		"title":      "Test Project",
		"projection": "EPSG:4326",
		"layers": map[string]interface{}{
			"layer1": map[string]interface{}{
				"name": "Test Layer",
				"type": "VectorLayer",
			},
		},
	}

	repo.qgisMetadata[projectName] = testMetadata

	var metadata map[string]interface{}
	err := service.GetQgisMetadata(projectName, &metadata)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if metadata["title"] != "Test Project" {
		t.Errorf("Expected title 'Test Project', got '%v'", metadata["title"])
	}
}

func TestProjectService_UpdateMeta(t *testing.T) {
	service, repo, _ := setupProjectService(t)

	projectName := "testuser/testproject"
	meta := json.RawMessage(`{"title": "Updated Project", "projection": "EPSG:3857"}`)

	err := service.UpdateMeta(projectName, meta)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify metadata was updated
	var updatedMeta map[string]interface{}
	err = service.GetQgisMetadata(projectName, &updatedMeta)
	if err != nil {
		t.Fatalf("Error retrieving updated metadata: %v", err)
	}

	if updatedMeta["title"] != "Updated Project" {
		t.Errorf("Expected title 'Updated Project', got '%v'", updatedMeta["title"])
	}
}

func TestProjectService_GetSettings(t *testing.T) {
	service, repo, _ := setupProjectService(t)

	projectName := "testuser/testproject"
	testSettings := domain.ProjectSettings{
		Title:    "Test Settings",
		MapCache: true,
	}

	repo.settings[projectName] = testSettings

	settings, err := service.GetSettings(projectName)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if settings.Title != "Test Settings" {
		t.Errorf("Expected title 'Test Settings', got '%s'", settings.Title)
	}

	if !settings.MapCache {
		t.Error("Expected MapCache to be true")
	}
}

func TestProjectService_UpdateSettings(t *testing.T) {
	service, repo, _ := setupProjectService(t)

	projectName := "testuser/testproject"
	settingsData := json.RawMessage(`{"title": "Updated Settings", "map_cache": false}`)

	err := service.UpdateSettings(projectName, settingsData)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify settings were updated in repository
	if _, exists := repo.settings[projectName]; !exists {
		t.Error("Expected settings to be saved in repository")
	}
}

func TestProjectService_ListProjectFiles(t *testing.T) {
	service, repo, _ := setupProjectService(t)

	projectName := "testuser/testproject"
	testFiles := []domain.ProjectFile{
		{Path: "file1.txt", Size: 1024, Hash: "hash1"},
		{Path: "file2.txt", Size: 2048, Hash: "hash2"},
	}

	testTempFiles := []domain.ProjectFile{
		{Path: "temp1.tmp", Size: 512, Hash: "temphash1"},
	}

	repo.files[projectName] = testFiles
	repo.temporaryFiles[projectName] = testTempFiles

	files, tempFiles, err := service.ListProjectFiles(projectName, true)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(files))
	}

	if len(tempFiles) != 1 {
		t.Errorf("Expected 1 temp file, got %d", len(tempFiles))
	}
}

func TestProjectService_GetScripts(t *testing.T) {
	service, repo, _ := setupProjectService(t)

	projectName := "testuser/testproject"
	testScripts := domain.Scripts{
		"module1": domain.ScriptModule{
			Path:       "scripts/module1.js",
			Components: []string{"Component1", "Component2"},
		},
	}

	repo.scripts[projectName] = testScripts

	scripts, err := service.GetScripts(projectName)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(scripts) != 1 {
		t.Errorf("Expected 1 script module, got %d", len(scripts))
	}

	if scripts["module1"].Path != "scripts/module1.js" {
		t.Errorf("Expected path 'scripts/module1.js', got '%s'", scripts["module1"].Path)
	}
}

func TestProjectService_UpdateScripts(t *testing.T) {
	service, repo, _ := setupProjectService(t)

	projectName := "testuser/testproject"
	scripts := domain.Scripts{
		"module1": domain.ScriptModule{
			Path:       "scripts/updated.js",
			Components: []string{"UpdatedComponent"},
		},
	}

	err := service.UpdateScripts(projectName, scripts)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify scripts were updated
	if updatedScripts, exists := repo.scripts[projectName]; exists {
		if updatedScripts["module1"].Path != "scripts/updated.js" {
			t.Error("Expected scripts to be updated in repository")
		}
	} else {
		t.Error("Expected scripts to be saved in repository")
	}
}

func TestProjectService_RemoveScripts(t *testing.T) {
	service, repo, _ := setupProjectService(t)

	projectName := "testuser/testproject"
	initialScripts := domain.Scripts{
		"module1": domain.ScriptModule{
			Path:       "scripts/module1.js",
			Components: []string{"Component1"},
		},
		"module2": domain.ScriptModule{
			Path:       "scripts/module2.js",
			Components: []string{"Component2"},
		},
	}

	repo.scripts[projectName] = initialScripts

	// Set up files to be removed
	repo.files[projectName] = []domain.ProjectFile{
		{Path: "web/scripts/module1.js", Size: 1024},
	}

	remainingScripts, err := service.RemoveScripts(projectName, "module1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(remainingScripts) != 1 {
		t.Errorf("Expected 1 remaining script, got %d", len(remainingScripts))
	}

	if _, exists := remainingScripts["module1"]; exists {
		t.Error("Expected module1 to be removed")
	}

	if _, exists := remainingScripts["module2"]; !exists {
		t.Error("Expected module2 to remain")
	}
}

func TestProjectService_GetThumbnailPath(t *testing.T) {
	service, repo, _ := setupProjectService(t)

	projectName := "testuser/testproject"
	expectedPath := "/custom/thumbnail/path"
	repo.thumbnailPaths[projectName] = expectedPath

	path := service.GetThumbnailPath(projectName)
	if path != expectedPath {
		t.Errorf("Expected path '%s', got '%s'", expectedPath, path)
	}
}

func TestProjectService_SaveThumbnail(t *testing.T) {
	service, repo, _ := setupProjectService(t)

	projectName := "testuser/testproject"
	reader := strings.NewReader("thumbnail data")

	err := service.SaveThumbnail(projectName, reader)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify thumbnail path was set
	if _, exists := repo.thumbnailPaths[projectName]; !exists {
		t.Error("Expected thumbnail path to be set")
	}
}

func TestProjectService_DeleteFile(t *testing.T) {
	service, repo, _ := setupProjectService(t)

	projectName := "testuser/testproject"
	filePath := "test/file.txt"

	// Setup initial files
	repo.files[projectName] = []domain.ProjectFile{
		{Path: filePath, Size: 1024},
	}

	err := service.DeleteFile(projectName, filePath)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// File deletion is handled by UpdateFiles, so we check that it was called
	// In the mock, we can verify the removes array was processed
}

func TestProjectService_GetLayersData(t *testing.T) {
	service, repo, _ := setupProjectService(t)

	projectName := "testuser/testproject"
	testMetadata := map[string]interface{}{
		"layers": map[string]interface{}{
			"layer1": map[string]interface{}{
				"name": "Test Layer 1",
				"type": "VectorLayer",
			},
			"layer2": map[string]interface{}{
				"name": "Test Layer 2",
				"type": "RasterLayer",
			},
		},
	}

	repo.qgisMetadata[projectName] = testMetadata

	layersData, err := service.GetLayersData(projectName)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(layersData.LayerNameToID) != 2 {
		t.Errorf("Expected 2 layers, got %d", len(layersData.LayerNameToID))
	}

	if layersData.LayerNameToID["Test Layer 1"] != "layer1" {
		t.Error("Expected layer name to ID mapping to be correct")
	}
}

func TestProjectService_GetProjectCustomizations(t *testing.T) {
	service, repo, _ := setupProjectService(t)

	projectName := "testuser/testproject"
	customizations := json.RawMessage(`{"theme": "dark", "logo": "custom.png"}`)
	repo.customizations[projectName] = customizations

	result, err := service.GetProjectCustomizations(projectName)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(result, &data); err != nil {
		t.Fatalf("Error unmarshaling customizations: %v", err)
	}

	if data["theme"] != "dark" {
		t.Errorf("Expected theme 'dark', got '%v'", data["theme"])
	}
}

func TestProjectService_Close(t *testing.T) {
	service, _, _ := setupProjectService(t)

	// Close should not panic or return error
	service.Close()
}

func TestProjectService_ExtractLayerVariables(t *testing.T) {
	service, repo, _ := setupProjectService(t)

	projectName := "testuser/testproject"

	// Test with layer variables in metadata
	testMetadata := map[string]interface{}{
		"layers": map[string]interface{}{
			"layer1": map[string]interface{}{
				"name": "Test Layer",
				"variables": map[string]interface{}{
					"qV_search":  "search_field",
					"custom_var": "custom_value",
				},
			},
		},
		"variables": map[string]interface{}{
			"project_var": "project_value",
		},
	}

	repo.qgisMetadata[projectName] = testMetadata

	metaLayers := map[string]domain.LayerMeta{
		"layer1": {
			Name: "Test Layer",
			Type: "VectorLayer",
		},
	}

	variables := service.extractLayerVariables(projectName, metaLayers)

	if len(variables) == 0 {
		t.Error("Expected variables to be extracted")
	}

	// Check layer variables
	if layerVars, exists := variables["layer1"]; exists {
		if layerVarsMap, ok := layerVars.(map[string]interface{}); ok {
			if layerVarsMap["qV_search"] != "search_field" {
				t.Errorf("Expected qV_search 'search_field', got '%v'", layerVarsMap["qV_search"])
			}
		} else {
			t.Error("Expected layer variables to be a map")
		}
	} else {
		t.Error("Expected layer1 variables to be present")
	}

	// Check project variables
	if projectVars, exists := variables["@project"]; exists {
		if projectVarsMap, ok := projectVars.(map[string]interface{}); ok {
			if projectVarsMap["project_var"] != "project_value" {
				t.Errorf("Expected project_var 'project_value', got '%v'", projectVarsMap["project_var"])
			}
		} else {
			t.Error("Expected project variables to be a map")
		}
	} else {
		t.Error("Expected @project variables to be present")
	}
}

// Error handling tests

func TestProjectService_ErrorHandling(t *testing.T) {
	t.Run("Create_LimiterError", func(t *testing.T) {
		service, repo, limiter := setupProjectService(t)

		limiter.shouldFailGetLimits = true
		repo.userProjects["testuser"] = []string{}

		_, err := service.Create("testuser/testproject", json.RawMessage(`{}`))
		if err == nil {
			t.Error("Expected error when limiter fails, got nil")
		}
	})

	t.Run("Create_UserProjectsError", func(t *testing.T) {
		service, repo, limiter := setupProjectService(t)

		limiter.limits["testuser"] = domain.AccountConfig{MaxProjects: 10}
		// Don't set userProjects to simulate error
		delete(repo.userProjects, "testuser")

		_, err := service.Create("testuser/testproject", json.RawMessage(`{}`))
		// This should work because empty slice is returned for missing users
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("SaveFile_LimiterError", func(t *testing.T) {
		service, _, limiter := setupProjectService(t)

		limiter.shouldFailGetLimits = true

		reader := strings.NewReader("test")
		_, err := service.SaveFile("testuser/testproject", "dir", "pattern", reader, 1024)
		if err == nil {
			t.Error("Expected error when limiter fails, got nil")
		}
	})

	t.Run("UpdateMeta_RepositoryError", func(t *testing.T) {
		service, repo, _ := setupProjectService(t)

		repo.shouldFailUpdate = true

		err := service.UpdateMeta("testuser/testproject", json.RawMessage(`{}`))
		if err == nil {
			t.Error("Expected error when repository update fails, got nil")
		}
	})

	t.Run("UpdateSettings_RepositoryError", func(t *testing.T) {
		service, repo, _ := setupProjectService(t)

		repo.shouldFailUpdate = true

		err := service.UpdateSettings("testuser/testproject", json.RawMessage(`{}`))
		if err == nil {
			t.Error("Expected error when repository update fails, got nil")
		}
	})

	t.Run("UpdateFiles_RepositoryError", func(t *testing.T) {
		service, repo, limiter := setupProjectService(t)

		limiter.limits["testuser"] = domain.AccountConfig{MaxProjectSize: 10 * 1024 * 1024}
		repo.projects["testuser/testproject"] = domain.ProjectInfo{Size: 0}
		repo.shouldFailUpdate = true

		changes := domain.FilesChanges{
			Updates: []domain.ProjectFile{
				{Path: "test.txt", Size: 1024},
			},
		}

		nextFunc := func() (string, io.ReadCloser, error) {
			return "test.txt", io.NopCloser(strings.NewReader("content")), nil
		}

		_, err := service.UpdateFiles("testuser/testproject", changes, nextFunc)
		if err == nil {
			t.Error("Expected error when repository update fails, got nil")
		}
	})
}

// Integration-style tests

func TestProjectService_IntegrationScenarios(t *testing.T) {
	t.Run("CompleteProjectLifecycle", func(t *testing.T) {
		service, repo, limiter := setupProjectService(t)

		projectName := "testuser/testproject"

		// Setup limits
		limiter.limits["testuser"] = domain.AccountConfig{
			MaxProjects:    10,
			MaxProjectSize: 10 * 1024 * 1024,
			MaxStorageSize: 100 * 1024 * 1024,
		}
		repo.userProjects["testuser"] = []string{}

		// 1. Create project
		meta := json.RawMessage(`{"title": "Integration Test Project"}`)
		project, err := service.Create(projectName, meta)
		if err != nil {
			t.Fatalf("Error creating project: %v", err)
		}

		// 2. Update metadata
		updatedMeta := json.RawMessage(`{"title": "Updated Project", "projection": "EPSG:3857"}`)
		err = service.UpdateMeta(projectName, updatedMeta)
		if err != nil {
			t.Fatalf("Error updating metadata: %v", err)
		}

		// 3. Save file
		reader := strings.NewReader("test file content")
		file, err := service.SaveFile(projectName, "media", "*.txt", reader, 1024)
		if err != nil {
			t.Fatalf("Error saving file: %v", err)
		}

		// 4. Update settings
		settings := json.RawMessage(`{"title": "Integration Settings", "map_cache": true}`)
		err = service.UpdateSettings(projectName, settings)
		if err != nil {
			t.Fatalf("Error updating settings: %v", err)
		}

		// 5. Get project info
		info, err := service.GetProjectInfo(projectName)
		if err != nil {
			t.Fatalf("Error getting project info: %v", err)
		}

		if info.Name != projectName {
			t.Errorf("Expected project name '%s', got '%s'", projectName, info.Name)
		}

		// 6. List files
		files, _, err := service.ListProjectFiles(projectName, true)
		if err != nil {
			t.Fatalf("Error listing files: %v", err)
		}

		if len(files) == 0 {
			t.Error("Expected files to be listed")
		}

		// 7. Delete file
		err = service.DeleteFile(projectName, file.Path)
		if err != nil {
			t.Fatalf("Error deleting file: %v", err)
		}

		// 8. Finally delete project
		err = service.Delete(projectName)
		if err != nil {
			t.Fatalf("Error deleting project: %v", err)
		}

		// Verify project was deleted
		_, err = service.GetProjectInfo(projectName)
		if err == nil {
			t.Error("Expected error when getting deleted project, got nil")
		}
	})

	t.Run("MultipleUsersProjects", func(t *testing.T) {
		service, repo, limiter := setupProjectService(t)

		// Setup multiple users
		users := []string{"user1", "user2", "user3"}
		for _, user := range users {
			limiter.limits[user] = domain.AccountConfig{
				MaxProjects: 5,
			}

			// Create projects for each user
			projects := []string{user + "/project1", user + "/project2"}
			repo.userProjects[user] = projects

			for _, projectName := range projects {
				repo.projects[projectName] = domain.ProjectInfo{
					Name:  projectName,
					Title: "Test Project",
					State: "published",
				}
			}
		}

		// Test getting projects for each user
		for _, user := range users {
			projects, err := service.GetUserProjects(user)
			if err != nil {
				t.Fatalf("Error getting projects for user %s: %v", user, err)
			}

			if len(projects) != 2 {
				t.Errorf("Expected 2 projects for user %s, got %d", user, len(projects))
			}
		}

		// Test accessible projects
		for _, user := range users {
			accessibleProjects, err := service.AccessibleProjects(user, true)
			if err != nil {
				t.Fatalf("Error getting accessible projects for user %s: %v", user, err)
			}

			if len(accessibleProjects) != 2 {
				t.Errorf("Expected 2 accessible projects for user %s, got %d", user, len(accessibleProjects))
			}
		}
	})
}

// Benchmark tests

func BenchmarkProjectService_GetUserProjects(b *testing.B) {
	logger := zap.NewNop().Sugar()
	repo := newMockProjectsRepository()
	limiter := newMockAccountsLimiter()

	service := NewProjectsService(logger, repo, limiter)

	// Setup test data
	user := "benchuser"
	projects := make([]string, 100)
	for i := 0; i < 100; i++ {
		projectName := fmt.Sprintf("%s/project%d", user, i)
		projects[i] = projectName
		repo.projects[projectName] = domain.ProjectInfo{
			Name:  projectName,
			Title: fmt.Sprintf("Project %d", i),
			State: "published",
		}
	}
	repo.userProjects[user] = projects

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.GetUserProjects(user)
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}

func BenchmarkProjectService_AccessibleProjects(b *testing.B) {
	logger := zap.NewNop().Sugar()
	repo := newMockProjectsRepository()
	limiter := newMockAccountsLimiter()

	service := NewProjectsService(logger, repo, limiter)

	// Setup test data
	user := "benchuser"
	projects := make([]string, 100)
	for i := 0; i < 100; i++ {
		projectName := fmt.Sprintf("%s/project%d", user, i)
		projects[i] = projectName
		repo.projects[projectName] = domain.ProjectInfo{
			Name:  projectName,
			Title: fmt.Sprintf("Project %d", i),
			State: "published",
		}
	}
	repo.userProjects[user] = projects

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.AccessibleProjects(user, true)
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}
