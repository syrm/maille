package web

import "embed"

//go:embed template/*
var TemplateFS embed.FS
