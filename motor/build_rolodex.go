// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package motor

import (
	"github.com/pb33f/libopenapi/index"
	"os"
	"path/filepath"
)

func BuildRolodexFromIndexConfig(indexConfig *index.SpecIndexConfig) (*index.Rolodex, error) {

	// create a rolodex
	rolodex := index.NewRolodex(indexConfig)

	// we need to create a local filesystem for the rolodex.
	if indexConfig.BasePath != "" {
		cwd, absErr := filepath.Abs(indexConfig.BasePath)
		if absErr != nil {
			return nil, absErr
		}

		// create a local filesystem
		fileFS, err := index.NewLocalFS(cwd, os.DirFS(cwd))
		if err != nil {
			return nil, err
		}

		// add the filesystem to the rolodex
		rolodex.AddLocalFS(cwd, fileFS)
	}

	// TODO: Remote filesystem

	return rolodex, nil

}
