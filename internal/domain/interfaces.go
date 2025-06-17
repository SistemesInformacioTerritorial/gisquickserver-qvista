// A internal/domain/interfaces.go (nou fitxer)
package domain

type ProjectConfigGenerator interface {
	GetMapConfig(projectName string, user User) (map[string]interface{}, error)
}
