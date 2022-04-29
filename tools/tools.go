//go:build tools
// +build tools

package tools

// This package contains import references to packages required only for the
// build process.
// https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module

import (
	//_ "github.com/kevinburke/go-bindata/go-bindata"
	//"github.com/securego/gosec/cmd/gosec"
	//"k8s.io/code-generator"
	//"sigs.k8s.io/controller-tools/cmd/controller-gen"

	_ "github.com/openshift/library-go/alpha-build-machinery"
	_ "k8s.io/code-generator"
)
