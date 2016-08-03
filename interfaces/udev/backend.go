// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2016 Canonical Ltd
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

// Package udev implements integration between snappy, udev and
// ubuntu-core-laucher around tagging character and block devices so that they
// can be accessed by applications.
//
// TODO: Document this better
package udev

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"os"

	"github.com/snapcore/snapd/dirs"
	"github.com/snapcore/snapd/interfaces"
	"github.com/snapcore/snapd/osutil"
	"github.com/snapcore/snapd/snap"
)

func snapSecurityTagGlob(snapName string) string {
	return fmt.Sprintf("snap.%s", snapName)
}

// Backend is responsible for maintaining udev rules.
type Backend struct{}

// Name returns the name of the backend.
func (b *Backend) Name() string {
	return "udev"
}

// Setup creates udev rules specific to a given snap.
// If any of the rules are changed or removed then udev database is reloaded.
//
// Since udev has no concept of a complain mode, devMode is ignored.
//
// If the method fails it should be re-tried (with a sensible strategy) by the caller.
func (b *Backend) Setup(snapInfo *snap.Info, devMode bool, repo *interfaces.Repository) error {
	snapName := snapInfo.Name()
	snippets, err := repo.SecuritySnippetsForSnap(snapInfo.Name(), interfaces.SecurityUDev)
	if err != nil {
		return fmt.Errorf("cannot obtain udev security snippets for snap %q: %s", snapName, err)
	}
	content, err := b.combineSnippets(snapInfo, snippets)
	if err != nil {
		return fmt.Errorf("cannot obtain expected udev rules for snap %q: %s", snapName, err)
	}
	glob := fmt.Sprintf("70-%s.rules", snapSecurityTagGlob(snapName))
	dir := dirs.SnapUdevRulesDir
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("cannot create directory for udev rules %q: %s", dir, err)
	}
	return ensureDirState(dir, glob, content, snapName)
}

// Remove removes udev rules specific to a given snap.
// If any of the rules are removed then udev database is reloaded.
//
// This method should be called after removing a snap.
//
// If the method fails it should be re-tried (with a sensible strategy) by the caller.
func (b *Backend) Remove(snapName string) error {
	glob := fmt.Sprintf("70-%s.rules", snapSecurityTagGlob(snapName))
	return ensureDirState(dirs.SnapUdevRulesDir, glob, nil, snapName)
}

func ensureDirState(dir, glob string, content map[string]*osutil.FileState, snapName string) error {
	var errReload error
	changed, removed, errEnsure := osutil.EnsureDirState(dir, glob, content)
	if len(changed) > 0 || len(removed) > 0 {
		// Try reload the rules regardless of errEnsure.
		errReload = ReloadRules()
	}
	if errEnsure != nil {
		return fmt.Errorf("cannot synchronize udev rules for snap %q: %s", snapName, errEnsure)
	}
	return errReload
}

func generateHash(bytes []byte) uint32 {
	h := fnv.New32a()
	h.Write(bytes)
	return h.Sum32()
}

// combineSnippets combines security snippets collected from all the interfaces
// affecting a given snap into a content map applicable to EnsureDirState.
func (b *Backend) combineSnippets(snapInfo *snap.Info, snippets map[string][][]byte) (content map[string]*osutil.FileState, err error) {
	var snapSnippets = make(map[uint32][]byte)
	var finalSnapSnippets [][]byte

	for _, appInfo := range snapInfo.Apps {
		securityTag := appInfo.SecurityTag()
		appSnippets := snippets[securityTag]
		if len(appSnippets) == 0 {
			continue
		}

		// Add all app snippets to the snap snippet list and
		// make sure we don't have doubles in there as they
		// get all added to the same udev rule file in the
		// end.
		for _, snippet := range appSnippets {
			snippetHash := generateHash(snippet)
			if _, ok := snapSnippets[snippetHash]; ok {
				continue
			}
			snapSnippets[snippetHash] = snippet
			finalSnapSnippets = append(finalSnapSnippets, snippet)
		}
	}

	if content == nil {
		content = make(map[string]*osutil.FileState)
	}

	snapSecurityTag := snapSecurityTagGlob(snapInfo.Name())
	addContent(snapSecurityTag, finalSnapSnippets, content)

	for _, hookInfo := range snapInfo.Hooks {
		securityTag := hookInfo.SecurityTag()
		hookSnippets := snippets[securityTag]
		if len(hookSnippets) == 0 {
			continue
		}
		if content == nil {
			content = make(map[string]*osutil.FileState)
		}

		addContent(securityTag, hookSnippets, content)
	}

	return content, nil
}

func addContent(securityTag string, executableSnippets [][]byte, content map[string]*osutil.FileState) {
	var buffer bytes.Buffer
	buffer.WriteString("# This file is automatically generated.\n")
	for _, snippet := range executableSnippets {
		buffer.Write(snippet)
		buffer.WriteRune('\n')
	}

	content[fmt.Sprintf("70-%s.rules", securityTag)] = &osutil.FileState{
		Content: buffer.Bytes(),
		Mode:    0644,
	}
}
