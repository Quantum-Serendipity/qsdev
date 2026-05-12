package devenv

import "embed"

//go:embed templates/*
var templateFS embed.FS
