package pkg

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/andrewkroh/go-fleetpkg"
	"gopkg.in/yaml.v3"
)

func ReadManifest(dir string) (*fleetpkg.Manifest, error) {
	var manifest fleetpkg.Manifest

	if err := readYAML(filepath.Join(dir, "manifest.yml"), &manifest, true); err != nil {
		return nil, fmt.Errorf("unable to read manifest: %w", err)
	}

	return &manifest, nil
}

func readYAML(path string, v any, strict bool) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	dec := yaml.NewDecoder(f)
	dec.KnownFields(strict)

	if err = dec.Decode(v); err != nil {
		return fmt.Errorf("failed decoding %s: %w", path, err)
	}

	return nil
}
