package gradle

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	glob "github.com/ryanuber/go-glob"
)

// Project ...
type Project struct {
	location string
	monoRepo bool
}

// NewProject ...
func NewProject(location string) (Project, error) {
	var err error
	location, err = filepath.Abs(location)
	if err != nil {
		return Project{}, err
	}

	buildGradleFound, err := pathutil.IsPathExists(filepath.Join(location, "build.gradle"))
	if err != nil {
		return Project{}, err
	}

	if !buildGradleFound {
		return Project{}, fmt.Errorf("no build.gradle file found in (%s)", location)
	}

	if location == "/" {
		return Project{location: location, monoRepo: false}, nil
	}

	root := filepath.Join(location, "..")

	files, err := ioutil.ReadDir(root)
	if err != nil {
		return Project{}, err
	}

	projectsCount := 0
	for _, file := range files {
		if file.IsDir() {
			if exists, err := pathutil.IsPathExists(filepath.Join(root, file.Name(), "build.gradle")); err != nil {
				return Project{}, err
			} else if exists {
				projectsCount++
			}
		}
	}

	return Project{location: location, monoRepo: (projectsCount > 1)}, nil
}

func getGradleModule(configModule string) string {
	if configModule != "" {
		return fmt.Sprintf(":%s:", configModule)
	}
	return ""
}

// GetModule ...
func (proj Project) GetModule(module string) Module {
	return Module{
		project: proj,
		name:    getGradleModule(module),
	}
}

// FindArtifacts ...
func (proj Project) FindArtifacts(generatedAfter time.Time, pattern string) ([]Artifact, error) {
	var a []Artifact
	return a, filepath.Walk(proj.location, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Warnf("failed to walk path: %s", err)
			return nil
		}

		if info.ModTime().Before(generatedAfter) || info.IsDir() || !glob.Glob(pattern, path) {
			return nil
		}

		name, err := proj.extractArtifactName(path)
		if err != nil {
			return err
		}

		a = append(a, Artifact{Name: name, Path: path})
		return nil
	})
}

func (proj Project) extractArtifactName(path string) (string, error) {
	relPath, err := filepath.Rel(proj.location, path)
	if err != nil {
		return "", err
	}

	module := strings.Split(relPath, "/")[0]
	fileName := filepath.Base(relPath)

	if proj.monoRepo {
		splitPath := strings.Split(proj.location, "/")
		prefix := splitPath[len(splitPath)-1]
		if prefix != "" {
			module = prefix + "-" + module
		}
	}

	return module + "-" + fileName, nil
}