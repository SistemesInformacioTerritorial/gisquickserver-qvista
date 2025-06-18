package application

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/gisquick/gisquick-server/internal/domain"
	"go.uber.org/zap"
)

var (
	ErrAccountProjectsLimit = errors.New("account projects count limit reached")
	ErrAccountStorageLimit  = errors.New("account storage limit reached")
	ErrProjectSizeLimit     = errors.New("project size limit reached")
)

type ProjectService interface {
	Create(projectName string, meta json.RawMessage) (*domain.ProjectInfo, error)
	Delete(projectName string) error
	GetProjectInfo(projectName string) (domain.ProjectInfo, error)
	GetUserProjects(username string) ([]domain.ProjectInfo, error)
	AccessibleProjects(username string, skipErrors bool) ([]domain.ProjectInfo, error)
	// SaveFile(projectName, filename string, r io.Reader) (string, error)
	SaveFile(projectName, dir, pattern string, r io.Reader, size int64) (domain.ProjectFile, error)
	DeleteFile(projectName, path string) error
	ListProjectFiles(projectName string, checksum bool) ([]domain.ProjectFile, []domain.ProjectFile, error)

	GetQgisMetadata(projectName string, data interface{}) error
	UpdateMeta(projectName string, meta json.RawMessage) error

	GetSettings(projectName string) (domain.ProjectSettings, error)
	UpdateSettings(projectName string, data json.RawMessage) error

	GetThumbnailPath(projectName string) string
	SaveThumbnail(projectName string, r io.Reader) error

	UpdateFiles(projectName string, info domain.FilesChanges, next func() (string, io.ReadCloser, error)) ([]domain.ProjectFile, error)

	GetLayersData(projectName string) (LayersData, error)
	GetMapConfig(projectName string, user domain.User) (map[string]interface{}, error)

	GetScripts(projectName string) (domain.Scripts, error)
	UpdateScripts(projectName string, scripts domain.Scripts) error
	RemoveScripts(projectName string, modules ...string) (domain.Scripts, error)

	GetProjectCustomizations(projectName string) (json.RawMessage, error)
	Close()
}

type AccountsLimiter interface {
	GetAccountLimits(username string) (domain.AccountConfig, error)
}

type projectService struct {
	log     *zap.SugaredLogger
	repo    domain.ProjectsRepository
	limiter AccountsLimiter
	// cache *ttlcache.Cache
}

func NewProjectsService(log *zap.SugaredLogger, repo domain.ProjectsRepository, limiter AccountsLimiter) *projectService {
	return &projectService{
		log:     log,
		repo:    repo,
		limiter: limiter,
	}
}

func (s *projectService) Create(name string, meta json.RawMessage) (*domain.ProjectInfo, error) {
	username := strings.Split(name, "/")[0]
	projects, err := s.repo.UserProjects(username)
	if err != nil {
		return nil, fmt.Errorf("getting user's projects: %w", err)
	}
	accountConfig, err := s.limiter.GetAccountLimits(username)
	if err != nil {
		return nil, fmt.Errorf("getting user account limits config: %w", err)
	}
	canCreate := accountConfig.CheckProjectsLimit(len(projects) + 1)
	if !canCreate {
		return nil, ErrAccountProjectsLimit
	}
	return s.repo.Create(name, meta)
}

func (s *projectService) GetProjectInfo(name string) (domain.ProjectInfo, error) {
	return s.repo.GetProjectInfo(name)
}

func (s *projectService) Delete(name string) error {
	return s.repo.Delete(name)
}

func (s *projectService) ListProjectFiles(project string, checksum bool) ([]domain.ProjectFile, []domain.ProjectFile, error) {
	return s.repo.ListProjectFiles(project, checksum)
}

func (s *projectService) GetUserProjects(username string) ([]domain.ProjectInfo, error) {
	fmt.Printf("Passo per GetUserProjects")
	/*
		projects, err := s.repo.UserProjects(username)
		if err != nil {
			return nil, err
		}
		data := make([]domain.ProjectInfo, len(projects))
		for i, name := range projects {
			info, err := s.repo.GetProjectInfo(name)
			if err != nil {
				// TODO: skip or fail?
				return nil, err
			}
			data[i] = info
		}
		return data, nil
	*/

	//jfs
	fmt.Printf("Entrando a GetUserProjects\n")
	projects, err := s.repo.UserProjects(username)
	if err != nil {
		fmt.Printf("Error obteniendo proyectos de usuario: %v\n", err)
		return nil, err
	}
	fmt.Printf("Proyectos obtenidos: %v\n", projects)
	data := make([]domain.ProjectInfo, len(projects))
	for i, name := range projects {
		fmt.Printf("Obteniendo información del proyecto: %s\n", name)
		info, err := s.repo.GetProjectInfo(name)
		if err != nil {
			fmt.Printf("Error obteniendo información del proyecto %s: %v\n", name, err)
			// TODO: skip or fail?
			return nil, err
		}
		data[i] = info
		fmt.Printf("Información del proyecto obtenida: %v\n", info)
	}
	fmt.Printf("Saliendo de GetUserProjects con datos: %v\n", data)
	return data, nil

}

func (s *projectService) AccessibleProjects(username string, skipErrors bool) ([]domain.ProjectInfo, error) {
	projects, err := s.repo.UserProjects(username)
	if err != nil {
		return nil, fmt.Errorf("getting user's projects: %w", err)
	}

	data := make([]domain.ProjectInfo, 0, len(projects))
	for _, name := range projects {
		info, err := s.repo.GetProjectInfo(name)
		if err != nil {
			if skipErrors {
				s.log.Warnw("Skipping project with error", "project", name, "error", err)
				continue
			}
			return nil, fmt.Errorf("getting project info for %s: %w", name, err)
		}
		data = append(data, info)
	}
	return data, nil
}

func (s *projectService) SaveFile(projectName, directory, pattern string, r io.Reader, size int64) (domain.ProjectFile, error) {
	username := strings.Split(projectName, "/")[0]
	accountConfig, err := s.limiter.GetAccountLimits(username)
	var finfo domain.ProjectFile
	if err != nil {
		return finfo, fmt.Errorf("getting user account limits config: %w", err)
	}
	checkProjectSizeLimit := accountConfig.HasProjectSizeLimit()
	checkStorageLimit := accountConfig.HasStorageLimit()

	var projectsSizes map[string]int64
	if checkStorageLimit {
		var err error
		projectsSizes, err = s.getProjectsSize(username)
		if err != nil {
			return finfo, fmt.Errorf("checking user storage limit: %w", err)
		}
		var totalSize int64 = 0
		for _, pSize := range projectsSizes {
			totalSize += pSize
		}
		canSave := accountConfig.CheckStorageLimit(totalSize + size)
		if !canSave {
			return finfo, ErrAccountStorageLimit
		}
	}
	if checkProjectSizeLimit {
		projectSize, ok := projectsSizes[projectName]
		if !ok {
			pi, err := s.GetProjectInfo(projectName)
			if err != nil {
				return finfo, fmt.Errorf("getting project size: %w", err)
			}
			projectSize = pi.Size
		}
		canSave := accountConfig.CheckProjectSizeLimit(projectSize + size)
		if !canSave {
			return finfo, ErrProjectSizeLimit
		}
	}

	finfo, err = s.repo.CreateFile(projectName, directory, pattern, r)
	if err != nil {
		return finfo, fmt.Errorf("saving project file: %w", err)
	}
	return finfo, nil
}

func (s *projectService) DeleteFile(projectName, path string) error {
	changes := domain.FilesChanges{Removes: []string{path}}
	_, err := s.UpdateFiles(projectName, changes, nil)
	if err != nil {
		return fmt.Errorf("deleting media file: %w", err)
	}
	return nil
}

func (s *projectService) GetQgisMetadata(projectName string, data interface{}) error {
	return s.repo.ParseQgisMetadata(projectName, data)
}

func (s *projectService) UpdateMeta(projectName string, meta json.RawMessage) error {
	return s.repo.UpdateMeta(projectName, meta)
}

func (s *projectService) GetSettings(projectName string) (domain.ProjectSettings, error) {
	return s.repo.GetSettings(projectName)
}

func (s *projectService) UpdateSettings(projectName string, data json.RawMessage) error {
	return s.repo.UpdateSettings(projectName, data)
}

func (s *projectService) SaveThumbnail(projectName string, r io.Reader) error {
	return s.repo.SaveThumbnail(projectName, r)
}

func (s *projectService) GetThumbnailPath(projectName string) string {
	return s.repo.GetThumbnailPath(projectName)
}

func (s *projectService) getProjectsSize(username string) (map[string]int64, error) {
	projNames, err := s.repo.UserProjects(username)
	if err != nil {
		return nil, fmt.Errorf("listing user projects: %w", err)
	}
	data := make(map[string]int64, len(projNames))
	for _, projName := range projNames {
		pInfo, err := s.repo.GetProjectInfo(projName)
		if err != nil {
			return nil, fmt.Errorf("getting project info: %w", err)
		}
		data[projName] = pInfo.Size
	}
	return data, nil
}

func (s *projectService) UpdateFiles(projectName string, info domain.FilesChanges, next func() (string, io.ReadCloser, error)) ([]domain.ProjectFile, error) {
	username := strings.Split(projectName, "/")[0]
	accountConfig, err := s.limiter.GetAccountLimits(username)
	if err != nil {
		return nil, fmt.Errorf("getting user account limits config: %w", err)
	}
	checkProjectSizeLimit := accountConfig.HasProjectSizeLimit()
	checkStorageLimit := accountConfig.HasStorageLimit()
	if len(info.Updates) > 0 && (checkProjectSizeLimit || checkStorageLimit) {
		p, err := s.GetProjectInfo(projectName)
		if err != nil {
			return nil, err
		}
		// v1
		/*
			size := p.Size
			for _, p := range info.Removes {
				fi, err := s.repo.GetFileInfo(projectName, p, false)
				if err != nil {
					return nil, fmt.Errorf("calculating project size [%s]: %w", p, err)
				}
				size -= fi.Size
			}
			for _, i := range info.Updates {
				fi, err := s.repo.GetFileInfo(projectName, i.Path, false)
				if err != nil && err != domain.ErrFileNotExists {
					return nil, fmt.Errorf("calculating project size [%s]: %w", i.Path, err)
				}
				if err == nil {
					size -= fi.Size
				}
				size += i.Size
			}
		*/
		// v2
		size := p.Size
		files := make([]string, 0, len(info.Updates)+len(info.Removes))
		files = append(files, info.Removes...)
		for _, f := range info.Updates {
			files = append(files, f.Path)
		}
		filesInfo, err := s.repo.GetFilesInfo(projectName, files...)
		for _, p := range info.Removes {
			fi, ok := filesInfo[p]
			if ok {
				size -= fi.Size
			}
		}
		for _, f := range info.Updates {
			fi, ok := filesInfo[f.Path]
			if ok {
				size -= fi.Size
			}
			size += f.Size
		}

		// s.log.Infow("UpdateFiles", "currentSize", p.Size, "expected size", size)
		if !accountConfig.CheckProjectSizeLimit(size) {
			return nil, ErrProjectSizeLimit
		}
		if checkStorageLimit {
			sizes, err := s.getProjectsSize(username)
			if err != nil {
				return nil, fmt.Errorf("checking user storage limit: %w", err)
			}
			var totalSize int64 = 0
			for _, pSize := range sizes {
				totalSize += pSize
			}
			totalSize += (-p.Size + size)
			if !accountConfig.CheckStorageLimit(totalSize) {
				return nil, ErrAccountStorageLimit
			}
		}
	}
	return s.repo.UpdateFiles(projectName, info, next)
}

func (s *projectService) GetScripts(projectName string) (domain.Scripts, error) {
	return s.repo.GetScripts(projectName)
}

func (s *projectService) UpdateScripts(projectName string, scripts domain.Scripts) error {
	return s.repo.UpdateScripts(projectName, scripts)
}

func (s *projectService) RemoveScripts(projectName string, modules ...string) (domain.Scripts, error) {
	scripts, err := s.GetScripts(projectName)
	if err != nil {
		return nil, err
	}
	files := make([]string, len(modules))
	for i, m := range modules {
		meta := scripts[m]
		files[i] = filepath.Join("web", meta.Path)
		delete(scripts, m)
	}
	changes := domain.FilesChanges{Removes: files}
	_, err = s.UpdateFiles(projectName, changes, nil)
	if err != nil {
		return nil, err
	}
	return scripts, s.UpdateScripts(projectName, scripts)
}

func contains(items []string, value string) bool {
	for _, i := range items {
		if i == value {
			return true
		}
	}
	return false
}

type LayersData struct {
	LayerNameToID map[string]string
}

func (s *projectService) GetLayersData(projectName string) (LayersData, error) {
	type LayersMetadata struct {
		Layers map[string]domain.LayerMeta `json:"layers"`
	}
	var meta LayersMetadata
	if err := s.GetQgisMetadata(projectName, &meta); err != nil {
		return LayersData{}, err
	}
	nameToID := make(map[string]string, len(meta.Layers))
	for id, layer := range meta.Layers {
		nameToID[layer.Name] = id
	}
	data := LayersData{
		LayerNameToID: nameToID,
	}
	return data, nil
}

type Layer struct {
	Name             string                     `json:"name"`
	Title            string                     `json:"title"`
	Type             string                     `json:"type"`
	Projection       string                     `json:"projection"`
	LegendURL        string                     `json:"legend_url,omitempty"`
	LegendDisabled   bool                       `json:"legend_disabled,omitempty"`
	Metadata         map[string]string          `json:"metadata"`
	Attribution      map[string]string          `json:"attribution,omitempty"`
	Extent           []float64                  `json:"extent"`
	Provider         string                     `json:"provider_type"`
	SourceParams     map[string]json.RawMessage `json:"source"`
	Visible          bool                       `json:"visible"`
	CustomProperties json.RawMessage            `json:"custom,omitempty"`
}

type BaseLayer struct {
	Layer
	// WMS params, old API
	URL       string   `json:"url"`
	Format    string   `json:"format"`
	WmsLayers []string `json:"wms_layers"`
}

type OverlayLayer struct {
	Layer
	Attributes           []domain.LayerAttribute `json:"attributes,omitempty"`
	Bands                []string                `json:"bands,omitempty"`
	DrawingOrder         *int                    `json:"drawing_order,omitempty"`
	Hidden               bool                    `json:"hidden"`
	Queryable            bool                    `json:"queryable"`
	GeomType             string                  `json:"wkb_type,omitempty"`
	InfoPanel            string                  `json:"infopanel_component,omitempty"`
	Permissions          domain.LayerPermission  `json:"permissions"`
	AttributeTableFields []string                `json:"attr_table_fields,omitempty"`
	InfoPanelFields      []string                `json:"info_panel_fields,omitempty"`
	ExportFields         []string                `json:"export_fields,omitempty"`
	// Relations            json.RawMessage         `json:"relations,omitempty"`
	Relations []map[string]any `json:"relations,omitempty"`
	// Campos existentes...
	QgisId string `json:"qgis_id"`
	Name   string `json:"name"`
	// Añadir este campo
	QVSearch string `json:"qV_search,omitempty"`
}

type SearchConfig struct {
	GeocodingAPI     string `json:"geocoding_api,omitempty"`
	SearchByLocation bool   `json:"search_by_coords"`
}

func filterList(list []string, test func(item string) bool) []string {
	res := make([]string, 0)
	for _, v := range list {
		if test(v) {
			res = append(res, v)
		}
	}
	return res
}

func MergeAttributeConfig(meta domain.LayerAttribute, settings domain.AttributeSettings) domain.LayerAttribute {
	attr := domain.LayerAttribute{
		Alias:       meta.Alias,
		Name:        meta.Name,
		Type:        meta.Type,
		Widget:      meta.Widget,
		Config:      meta.Config,
		Constraints: meta.Constraints,
	}
	if settings.Widget != "" {
		attr.Widget = settings.Widget
	}
	if len(settings.Config) > 0 {
		attr.Config = settings.Config
	}
	if settings.Formatter != "" {
		attr.Format = settings.Formatter
	}
	return attr
}

func indexOf(items []string, value string) int {
	for i, v := range items {
		if v == value {
			return i
		}
	}
	return -1
}

func TransformLayersTree(tree []domain.TreeNode, accept func(id string) bool, transform func(id string) any, groupsSettings map[string]domain.GroupSettings) ([]any, error) {
	list := make([]any, 0)
	// without reordering
	// for _, n := range tree {
	// 	if n.IsGroup() {
	// 		layers, err := TransformLayersTree(n.Children(), accept, transform)
	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 		if len(layers) > 0 {
	// 			g := n.(domain.GroupTreeNode)
	// 			ng := map[string]interface{}{
	// 				"name":               n.GroupName(),
	// 				"layers":             layers,
	// 				"mutually_exclusive": g.MutuallyExclusive,
	// 			}
	// 			list = append(list, ng)
	// 		}
	// 	} else if accept(n.LayerID()) {
	// 		list = append(list, transform(n.LayerID()))
	// 	}
	// }

	// with reordering - groups after layers
	for _, n := range tree {
		if !n.IsGroup() && accept(n.LayerID()) {
			list = append(list, transform(n.LayerID()))
		}
	}
	for _, n := range tree {
		if n.IsGroup() {
			layers, err := TransformLayersTree(n.Children(), accept, transform, groupsSettings)
			if err != nil {
				return nil, err
			}
			if len(layers) > 0 {
				g := n.(domain.GroupTreeNode)
				ng := map[string]interface{}{
					"name":               n.GroupName(),
					"layers":             layers,
					"mutually_exclusive": g.MutuallyExclusive,
				}
				if g.WmsName != "" {
					// ng["wms_name"] = g.WmsName
					if s, ok := groupsSettings[g.WmsName]; ok {
						ng["collapsed"] = s.Collapsed
						ng["virtual_layer"] = s.VirtualLayer
					}
				}
				list = append(list, ng)
			}
		}
	}
	return list, nil
}

// Returns not excluded ordered fields for InfoPanel
func GetInfoPanelFields(lm domain.LayerMeta, ls domain.LayerSettings) domain.StringArray {
	var fields domain.StringArray
	if ls.FieldsOrder != nil {
		fields = ls.FieldsOrder.Global
		if len(fields) == 0 && len(ls.FieldsOrder.Infopanel) > 0 {
			fields = ls.FieldsOrder.Infopanel
		}
	} else {
		fields = make([]string, len(lm.Attributes))
		for i, a := range lm.Attributes {
			fields[i] = a.Name
		}
	}
	if ls.ExcludedFields != nil {
		return fields.Filter(func(item string) bool {
			return !ls.ExcludedFields.Global.Has(item) && !ls.ExcludedFields.Infopanel.Has(item)
		})
	}
	return fields
}

func GetTableFields(lm domain.LayerMeta, ls domain.LayerSettings) domain.StringArray {
	var fields domain.StringArray
	if ls.FieldsOrder != nil {
		fields = ls.FieldsOrder.Global
		if len(fields) == 0 && len(ls.FieldsOrder.Table) > 0 {
			fields = ls.FieldsOrder.Table
		}
	} else {
		fields = make([]string, len(lm.Attributes))
		for i, a := range lm.Attributes {
			fields[i] = a.Name
		}
	}
	if ls.ExcludedFields != nil {
		return fields.Filter(func(item string) bool {
			return !ls.ExcludedFields.Global.Has(item) && !ls.ExcludedFields.Table.Has(item)
		})
	}
	return fields
}

func (s *projectService) GetProjectCustomizations(projectName string) (json.RawMessage, error) {
	return s.repo.GetProjectCustomizations(projectName)
}

func (s *projectService) GetMapConfig(projectName string, user domain.User) (map[string]interface{}, error) {
	var meta domain.QgisMeta
	if err := s.repo.ParseQgisMetadata(projectName, &meta); err != nil {
		return nil, fmt.Errorf("parsing qgis meta: %w", err)
	}
	settings, err := s.repo.GetSettings(projectName)
	if err != nil {
		return nil, err
	}
	layersTree, err := domain.CreateTree2(meta.LayersTree)
	if err != nil {
		return nil, err
	}

	// override proj4 definitions if set in the settings
	for c, proj4 := range settings.Proj4 {
		proj, ok := meta.Projections[c]
		if ok {
			proj.Proj4 = proj4
		}
	}

	// split layers into base layers and overlay layers
	baseLayers := make([]domain.TreeNode, 0)
	overlays := make([]domain.TreeNode, 0)
	for _, item := range layersTree {
		var id string
		if item.IsGroup() {
			id = item.GroupName()
		} else {
			id = item.LayerID()
		}
		if contains(settings.BaseLayers, id) {
			baseLayers = append(baseLayers, item)
		} else {
			overlays = append(overlays, item)
		}
	}

	rolesPerms := domain.NewUserRolesPermissions(user, settings.Auth)

	baseLayersData, err := TransformLayersTree(
		baseLayers,
		func(id string) bool {
			return !settings.Layers[id].Flags.Has("excluded") && (rolesPerms == nil || rolesPerms.LayerFlags(id).Has("view"))
		},
		func(id string) interface{} {
			lmeta := meta.Layers[id]
			lset := settings.Layers[id]
			ldata := BaseLayer{
				Layer: Layer{
					Name:             lmeta.Name,
					Title:            lmeta.Title,
					Type:             lmeta.Type,
					Projection:       lmeta.Projection,
					Metadata:         lmeta.Metadata,
					LegendURL:        lmeta.LegendURL,
					Attribution:      lmeta.Attribution,
					Extent:           lmeta.Extent,
					Provider:         lmeta.Provider,
					SourceParams:     lmeta.SourceParams,
					Visible:          lmeta.Visible,
					CustomProperties: lset.CustomProperties,
					LegendDisabled:   lset.LegendDisabled,
				},
			}
			if lmeta.Type == "RasterLayer" && lmeta.Provider == "wms" {
				ldata.Format = lmeta.SourceParams.String("format")
				ldata.URL = lmeta.SourceParams.String("url")
				ldata.WmsLayers = lmeta.SourceParams.StringArray("layers")
			}
			return ldata
		},
		settings.Groups,
	)
	if err != nil {
		return nil, err
	}

	layers, err := TransformLayersTree(
		overlays,
		func(id string) bool {
			return !settings.Layers[id].Flags.Has("excluded") && (rolesPerms == nil || rolesPerms.LayerFlags(id).Has("view"))
		},
		func(id string) interface{} {
			lmeta := meta.Layers[id]
			lset := settings.Layers[id]
			lflags := lset.Flags
			if rolesPerms != nil {
				lflags = lflags.Intersection(rolesPerms.LayerFlags(id))
			}

			queryable := lmeta.Flags.Has("query") && !lset.Flags.Has("hidden") && lflags.Has("query")

			relations := lmeta.Relations
			if lset.QgisRelations != nil || lset.Relations != nil {
				relations = make([]map[string]any, len(lmeta.Relations)+len(lset.Relations))
				for i, r := range lmeta.Relations {
					rid := r["id"].(string)
					rs, hasSettings := lset.QgisRelations[rid]
					if hasSettings {
						relations[i] = make(map[string]any)
						for k, v := range r {
							relations[i][k] = v
						}
						for k, v := range rs {
							relations[i][k] = v
						}
					} else {
						relations[i] = lmeta.Relations[i]
					}
				}
				offset := len(lmeta.Relations)
				for i, r := range lset.Relations {
					relations[offset+i] = r
				}
			}
			ldata := OverlayLayer{
				Layer: Layer{
					Name:             lmeta.Name,
					Title:            lmeta.Title,
					Projection:       lmeta.Projection,
					Type:             lmeta.Type,
					Metadata:         lmeta.Metadata,
					LegendURL:        lmeta.LegendURL,
					Attribution:      lmeta.Attribution,
					Visible:          lmeta.Visible,
					CustomProperties: lset.CustomProperties,
					LegendDisabled:   lset.LegendDisabled,
					Extent:           lmeta.Extent,
					Provider:         lmeta.Provider,
					SourceParams:     lmeta.SourceParams,
				},
				Bands: lmeta.Bands,

				Relations: relations,
				Hidden:    lset.Flags.Has("hidden"),
				Queryable: queryable,
				InfoPanel: lset.InfoPanelComponent,
			}

			drawingOrder := -1
			if !lset.Flags.Has("render_off") {
				drawingOrder = indexOf(meta.LayersOrder, id)
			}
			if drawingOrder != -1 {
				ldata.DrawingOrder = &drawingOrder
			} else {
				ldata.Visible = false
			}

			if lmeta.Type == "VectorLayer" {
				json.Unmarshal(lmeta.Options["wkb_type"], &ldata.GeomType)
				var wfsFlags domain.Flags
				json.Unmarshal(lmeta.Options["wfs"], &wfsFlags)

				editable := queryable && lmeta.Flags.Has("edit") && lset.Flags.Has("edit")
				ldata.Permissions = domain.LayerPermission{
					View:         queryable,
					Insert:       editable && wfsFlags.Has("insert"),
					Delete:       editable && wfsFlags.Has("delete"),
					Update:       editable && wfsFlags.Has("update"),
					EditGeometry: editable,
				}
				if rolesPerms != nil {
					lperms := rolesPerms.LayerFlags(id)
					ldata.Permissions.Insert = ldata.Permissions.Insert && lperms.Has("insert")
					ldata.Permissions.Delete = ldata.Permissions.Delete && lperms.Has("delete")
					ldata.Permissions.Update = ldata.Permissions.Update && lperms.Has("update")
				}

				if queryable && len(lmeta.Attributes) > 0 {
					if lset.Flags.Has("export") {
						ldata.ExportFields = lset.ExportFields
					}

					if rolesPerms != nil {
						attrsPerms := rolesPerms.AttributesFlags(id)
						geomPerms, hasGeomPerms := attrsPerms["geometry"]
						ldata.Permissions.EditGeometry = ldata.Permissions.EditGeometry && (!hasGeomPerms || geomPerms.Has("edit"))
						isAttributeVisible := func(item string) bool { return attrsPerms[item].Has("view") }

						ldata.AttributeTableFields = GetTableFields(lmeta, lset).Filter(isAttributeVisible)
						ldata.InfoPanelFields = GetInfoPanelFields(lmeta, lset).Filter(isAttributeVisible)

						if len(ldata.ExportFields) > 0 {
							ldata.ExportFields = filterList(
								ldata.ExportFields,
								func(item string) bool { return attrsPerms[item].Has("export") },
							)
						}
						ldata.Attributes = make([]domain.LayerAttribute, 0, len(lmeta.Attributes))
						for _, a := range lmeta.Attributes {
							if isAttributeVisible(a.Name) {
								attr := MergeAttributeConfig(a, lset.Attributes[a.Name])
								if !attrsPerms[a.Name].Has("edit") && !attr.Constraints.Has("readonly") {
									attr.Constraints = attr.Constraints.Union(domain.Flags{"readonly"})
								}
								ldata.Attributes = append(ldata.Attributes, attr)
							}
						}
					} else {
						ldata.AttributeTableFields = GetTableFields(lmeta, lset)
						ldata.InfoPanelFields = GetInfoPanelFields(lmeta, lset)

						ldata.Attributes = make([]domain.LayerAttribute, len(lmeta.Attributes))
						for i, a := range lmeta.Attributes {
							ldata.Attributes[i] = MergeAttributeConfig(a, lset.Attributes[a.Name])
						}
					}
				}
			}
			return ldata
		},
		settings.Groups,
	)
	if err != nil {
		return nil, err
	}

	data := make(map[string]interface{})
	data["use_mapcache"] = settings.MapCache
	data["zoom_extent"] = settings.InitialExtent
	data["project_extent"] = settings.Extent
	data["scales"] = settings.Scales
	data["tile_resolutions"] = settings.TileResolutions
	data["map_tiling"] = settings.MapTiling
	data["layers"] = layers
	data["base_layers"] = baseLayersData
	data["projection"] = meta.Projection
	data["projections"] = meta.Projections
	data["units"] = meta.Units
	data["print_composers"] = meta.ComposerTemplates

	if len(settings.Formatters) > 0 {
		data["formatters"] = settings.Formatters
	}

	scripts, err := s.GetScripts(projectName)
	if err != nil {
		s.log.Errorw("[GetMapConfig] loading scripts", "project", projectName)
	} else {
		data["scripts"] = scripts
	}

	if settings.Title != "" {
		data["title"] = settings.Title
	} else {
		data["title"] = meta.Title
	}
	data["root_title"] = data["title"]
	data["name"] = projectName
	data["ows_url"] = fmt.Sprintf("/api/map/ows/%s", projectName)
	data["ows_project"] = projectName

	// EXTRAER VARIABLES DE CAPA DEL ARCHIVO QGS/QGZ
	layerVariables := s.extractLayerVariables(projectName, meta.Layers)
	if len(layerVariables) > 0 {
		s.integrateVariablesIntoLayers(layers, layerVariables, meta.Layers)
		s.integrateVariablesIntoLayers(baseLayersData, layerVariables, meta.Layers)
	}

	topics := make([]domain.Topic, 0)
	var visibleTopics []string
	if rolesPerms != nil {
		visibleTopics = rolesPerms.UserTopics()
	}
	for _, topic := range settings.Topics {
		if visibleTopics != nil && !contains(visibleTopics, topic.ID) {
			continue
		}
		topicLayers := make([]string, 0)
		for _, lid := range topic.Layers {
			lset := settings.Layers[lid]
			visible := !lset.Flags.Has("excluded") && !lset.Flags.Has("hidden")
			if visible && rolesPerms != nil {
				visible = rolesPerms.LayerFlags(lid).Has("view")
			}
			if visible {
				topicLayers = append(topicLayers, meta.Layers[lid].Name)
			}
		}
		if len(topicLayers) > 0 {
			topics = append(topics, domain.Topic{Title: topic.Title, Abstract: topic.Abstract, Layers: topicLayers})
		}
	}
	data["topics"] = topics

	if settings.Geocoding != nil || settings.SearchByLocation {
		search := SearchConfig{SearchByLocation: settings.SearchByLocation}
		if settings.Geocoding != nil {
			search.GeocodingAPI = settings.Geocoding.Service
		}
		data["search"] = search
	}

	s.log.Infow("GetMapConfig", "projectName", projectName, "ows_url", data["ows_url"])

	return data, nil
}

// NUEVA función simplificada para extraer variables
func (s *projectService) extractLayerVariables(projectName string, metaLayers map[string]domain.LayerMeta) map[string]interface{} {
	s.log.Infow("🔍 [extractLayerVariables] INICIANDO EXTRACCIÓN",
		"project", projectName,
		"metaLayersCount", len(metaLayers))

	// Mostrar todas las capas del meta para diagnóstico
	s.log.Infow("📋 [extractLayerVariables] CAPAS EN METADATA:")
	for metaId, metaLayer := range metaLayers {
		s.log.Infow("   📌 Meta Capa",
			"metaId", metaId,
			"metaName", metaLayer.Name,
			"metaTitle", metaLayer.Title)
	}

	layerVariables := make(map[string]interface{})

	// Usar el repositorio para obtener variables del QGS
	if repo, ok := s.repo.(interface {
		GetLayerVariables(string) (map[string]map[string]string, error)
	}); ok {
		varsFromQgs, err := repo.GetLayerVariables(projectName)
		if err != nil {
			s.log.Errorw("❌ [extractLayerVariables] Error obteniendo variables del QGS",
				"project", projectName,
				"error", err)
		} else {
			s.log.Infow("📊 [extractLayerVariables] Variables extraídas del QGS",
				"project", projectName,
				"qgsLayersWithVars", len(varsFromQgs))

			// Mostrar todas las variables extraídas del QGS
			s.log.Infow("🗂️ [extractLayerVariables] CAPAS DEL QGS CON VARIABLES:")
			for qgsLayerId, variables := range varsFromQgs {
				s.log.Infow("   🎯 QGS Capa",
					"qgsLayerId", qgsLayerId,
					"variables", variables)
			}

			// Intentar mapear IDs entre QGS y metadata
			s.log.Infow("🔄 [extractLayerVariables] INICIANDO MAPEO DE IDs")
			for qgsLayerId, variables := range varsFromQgs {
				s.log.Infow("🔍 [extractLayerVariables] Buscando match para capa QGS",
					"qgsLayerId", qgsLayerId)

				matchFound := false
				for metaId, metaLayer := range metaLayers {
					// Estrategia 1: Match directo por ID
					if qgsLayerId == metaId {
						layerVariables[metaId] = variables
						s.log.Infow("✅ [extractLayerVariables] MATCH DIRECTO por ID",
							"qgsLayerId", qgsLayerId,
							"metaId", metaId,
							"variables", variables)
						matchFound = true
						break
					}

					// Estrategia 2: Match por nombre (normalizado)
					qgsLayerIdNorm := strings.ToLower(strings.TrimSpace(qgsLayerId))
					metaNameNorm := strings.ToLower(strings.TrimSpace(metaLayer.Name))
					metaTitleNorm := strings.ToLower(strings.TrimSpace(metaLayer.Title))

					if strings.Contains(qgsLayerIdNorm, metaNameNorm) ||
						strings.Contains(metaNameNorm, qgsLayerIdNorm) ||
						strings.Contains(qgsLayerIdNorm, metaTitleNorm) ||
						strings.Contains(metaTitleNorm, qgsLayerIdNorm) {
						layerVariables[metaId] = variables
						s.log.Infow("✅ [extractLayerVariables] MATCH por NOMBRE",
							"qgsLayerId", qgsLayerId,
							"metaId", metaId,
							"metaName", metaLayer.Name,
							"metaTitle", metaLayer.Title,
							"variables", variables)
						matchFound = true
						break
					}

					// Estrategia 3: Match por partes del ID
					qgsLayerParts := strings.Split(qgsLayerId, "_")
					for _, part := range qgsLayerParts {
						if len(part) > 3 && (strings.Contains(metaNameNorm, strings.ToLower(part)) ||
							strings.Contains(metaTitleNorm, strings.ToLower(part))) {
							layerVariables[metaId] = variables
							s.log.Infow("✅ [extractLayerVariables] MATCH por PARTE del ID",
								"qgsLayerId", qgsLayerId,
								"matchingPart", part,
								"metaId", metaId,
								"metaName", metaLayer.Name,
								"variables", variables)
							matchFound = true
							break
						}
					}
					if matchFound {
						break
					}
				}

				if !matchFound {
					s.log.Warnw("⚠️ [extractLayerVariables] NO SE ENCONTRÓ MATCH",
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
		s.log.Warnw("⚠️ [extractLayerVariables] Repositorio no soporta GetLayerVariables",
			"project", projectName)
	}

	s.log.Infow("🏁 [extractLayerVariables] EXTRACCIÓN COMPLETADA",
		"project", projectName,
		"totalVariablesIntegradas", len(layerVariables))

	return layerVariables
}

// integrateVariablesIntoLayers integra las variables directamente en cada capa
func (s *projectService) integrateVariablesIntoLayers(
	layers []interface{},
	layerVariables map[string]interface{},
	metaLayers map[string]domain.LayerMeta) {

	s.log.Infow("🔧 [integrateVariablesIntoLayers] INICIANDO INTEGRACIÓN",
		"layersCount", len(layers),
		"variablesCount", len(layerVariables))

	for i, item := range layers {
		s.log.Debugw("🔍 [integrateVariablesIntoLayers] Procesando item",
			"index", i,
			"itemType", fmt.Sprintf("%T", item))

		switch layer := item.(type) {
		case map[string]interface{}:
			s.integrateVariablesInMap(layer, layerVariables, i)
		case *OverlayLayer:
			s.integrateVariablesInOverlay(layer, layerVariables, i)
		default:
			s.log.Debugw("🤷 [integrateVariablesIntoLayers] Tipo no soportado",
				"index", i,
				"type", fmt.Sprintf("%T", item))
		}
	}

	s.log.Infow("🏁 [integrateVariablesIntoLayers] INTEGRACIÓN COMPLETADA")
}

// integrateVariablesInMap maneja capas como map[string]interface{}
func (s *projectService) integrateVariablesInMap(layerMap map[string]interface{}, layerVariables map[string]interface{}, index int) {
	layerName, _ := layerMap["name"].(string)
	layerId, _ := layerMap["id"].(string)
	qgisId, _ := layerMap["qgis_id"].(string)

	s.log.Infow("🗂️ [integrateVariablesInMap] Procesando capa map",
		"index", index,
		"layerName", layerName,
		"layerId", layerId,
		"qgisId", qgisId)

	// Buscar variables para esta capa
	for varLayerId, variables := range layerVariables {
		if varLayerId == layerId || varLayerId == qgisId {
			if varsMap, ok := variables.(map[string]string); ok {
				if qvSearch, exists := varsMap["qV_search"]; exists {
					layerMap["qV_search"] = qvSearch
					s.log.Infow("✅ [integrateVariablesInMap] Variable qV_search INTEGRADA en MAP",
						"index", index,
						"layerName", layerName,
						"layerId", layerId,
						"varLayerId", varLayerId,
						"qV_search", qvSearch)
				}
			}
		}
	}
}

// integrateVariablesInOverlay maneja capas OverlayLayer
func (s *projectService) integrateVariablesInOverlay(overlay *OverlayLayer, layerVariables map[string]interface{}, index int) {
	s.log.Infow("🗂️ [integrateVariablesInOverlay] Procesando OverlayLayer",
		"index", index,
		"overlayName", overlay.Name,
		"overlayQgisId", overlay.QgisId)

	// Buscar variables para esta capa
	for varLayerId, variables := range layerVariables {
		if varLayerId == overlay.QgisId || strings.Contains(varLayerId, overlay.Name) {
			if varsMap, ok := variables.(map[string]string); ok {
				if qvSearch, exists := varsMap["qV_search"]; exists {
					overlay.QVSearch = qvSearch
					s.log.Infow("✅ [integrateVariablesInOverlay] Variable qV_search INTEGRADA en OVERLAY",
						"index", index,
						"overlayName", overlay.Name,
						"overlayQgisId", overlay.QgisId,
						"varLayerId", varLayerId,
						"qV_search", qvSearch)
				}
			}
		}
	}
}

func (s *projectService) Close() {
	if s.repo != nil {
		s.repo.Close()
	}
}

// A projects.go - afegir aquest mètode
func (s *projectService) SetRepo(repo domain.ProjectsRepository) {
	s.repo = repo
}
