// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The semrel Authors

package main

import (
	"log"

	plugin "github.com/SemRels/hook-gitplugin/internal/plugin"
)

func main() {
	gitPlugin := plugin.NewPlugin(plugin.Config{})
	log.Printf("hook-gitplugin plugin ready: creates git commits, tags, and pushes (%T)", gitPlugin)
}
