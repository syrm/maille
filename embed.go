package embed

import "embed"

//go:embed migration/*
var MigrationFS embed.FS
