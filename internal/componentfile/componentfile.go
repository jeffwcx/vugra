package componentfile

import (
	"path/filepath"
	"strings"
)

const DefaultExt = ".vue"
const VugraExt = ".vugra"
const LegacyExt = ".vuego"

func IsComponentPath(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == DefaultExt || ext == VugraExt || ext == LegacyExt
}

func TrimComponentExt(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == DefaultExt || ext == VugraExt || ext == LegacyExt {
		return strings.TrimSuffix(path, filepath.Ext(path))
	}
	return path
}

func AlternatePath(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case DefaultExt:
		return strings.TrimSuffix(path, filepath.Ext(path)) + VugraExt
	case VugraExt:
		return strings.TrimSuffix(path, filepath.Ext(path)) + DefaultExt
	case LegacyExt:
		return strings.TrimSuffix(path, filepath.Ext(path)) + DefaultExt
	default:
		return path
	}
}
