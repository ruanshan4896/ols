package templates

import "embed"

//go:embed core/* site/*
var FS embed.FS
