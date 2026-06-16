package templates

import "embed"

//go:embed embedded/*
var embeddedTemplates embed.FS
