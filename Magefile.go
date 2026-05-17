//go:build mage

package main

import (
	"github.com/grafana/grafana-plugin-sdk-go/build"
	"github.com/magefile/mage/mg"
)

// Default target to run when none is specified.
var Default = Build.Linux

// Build contains build targets.
type Build mg.Namespace

// Linux builds the plugin for Linux.
func (Build) Linux() error {
	return build.Linux()
}

// Darwin builds the plugin for macOS.
func (Build) Darwin() error {
	return build.Darwin()
}

// Windows builds the plugin for Windows.
func (Build) Windows() error {
	return build.Windows()
}

// All builds the plugin for all platforms.
func (Build) All() error {
	b := build.Build{}
	return b.BuildAll()
}

// Test runs the Go test suite.
func Test() error {
	return build.Test()
}

// Coverage runs tests with coverage reporting.
func Coverage() error {
	return build.Coverage()
}

// Lint runs golangci-lint.
func Lint() error {
	return build.Lint()
}

// Clean removes build artifacts.
func Clean() error {
	return build.Clean()
}
