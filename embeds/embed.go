package embeds

import "embed"

//go:embed stubs/hover_runtime/*
var HoverRuntimeStubs embed.FS

//go:embed stubs/misc/manifest.yml
var HoverManifest string

//go:embed stubs/misc/.Dockerfile
var HoverDockerfile string
