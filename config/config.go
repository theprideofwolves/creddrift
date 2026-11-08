// Package config defines the explicit in-memory directory scan scope.
package config

// Config is set by the launcher and the user's scan request, not a startup file.
type Config struct {
	ScanTargets []string
	IgnoreDirs  []string
}
