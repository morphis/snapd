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

package snap

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/snapcore/snapd/osutil"
	"github.com/snapcore/snapd/release"
)

var implicitSlots = []string{
	"firewall-control",
	"home",
	"hardware-observe",
	"locale-control",
	"log-observe",
	"mount-observe",
	"network",
	"network-bind",
	"network-control",
	"network-observe",
	"ppp",
	"snapd-control",
	"system-observe",
	"timeserver-control",
	"timezone-control",
}

var implicitClassicSlots = []string{
	"cups-control",
	"gsettings",
	"network-manager",
	"opengl",
	"pulseaudio",
	"unity7",
	"x11",
	"modem-manager",
	"optical-drive",
	"camera",
}

var gpioSysfsBasePath = "/sys/class/gpio"
var gpioChipPrefix = "gpiochip"

func readNumber(path string) (number int, err error) {
	f, err := os.Open(path)
	if err != nil {
		fmt.Println("Failed to open path ", path)
		return
	}

	buf := bytes.NewBuffer(nil)
	io.Copy(buf, f)
	f.Close()

	number, err = strconv.Atoi(strings.TrimSpace(string(buf.Bytes())))
	return number, err
}

func createGpioSlots(snapInfo *Info, base int, ngpio int) {
	for n := base; n < base+ngpio; n++ {
		slotName := fmt.Sprintf("gpio-%d", n)
		if _, ok := snapInfo.Slots[slotName]; !ok {
			snapInfo.Slots[slotName] = &SlotInfo{
				Name:      slotName,
				Snap:      snapInfo,
				Interface: "gpio",
				Attrs:     map[string]interface{}{"number": n},
			}
		}
	}
}

func addSlotsForGpioChip(snapInfo *Info, chipPath string) {
	fmt.Printf("Adding slots for GPIO chip %s\n", chipPath)

	base, err := readNumber(path.Join(chipPath, "base"))
	if err != nil {
		fmt.Printf("Failed to read base GPIO number for chip %s\n", chipPath)
		return
	}

	ngpio, err := readNumber(path.Join(chipPath, "ngpio"))
	if err != nil {
		fmt.Printf("Failed to read number of GPIO for chip %s\n", chipPath)
		return
	}

	createGpioSlots(snapInfo, base, ngpio)
}

func gpioCreateImplicitSlots(snapInfo *Info) {
	fmt.Printf("Adding GPIO implicit slots\n")

	d, err := os.Open(gpioSysfsBasePath)
	if err != nil {
		fmt.Printf("System without GPIO support!\n")
		return
	}

	defer d.Close()

	files, err := d.Readdir(-1)
	if err != nil {
		fmt.Printf("Failed to detect available GPIOs\n")
		return
	}

	// Parse all GPIO chip's and add slots for their GPIOs
	for _, file := range files {
		if (file.Mode()&os.ModeSymlink != 0) && strings.HasPrefix(file.Name(), gpioChipPrefix) {
			addSlotsForGpioChip(snapInfo, path.Join(gpioSysfsBasePath, file.Name()))
		}
	}
}

// AddImplicitSlots adds implicitly defined slots to a given snap.
//
// Only the OS snap has implicit slots.
//
// It is assumed that slots have names matching the interface name. Existing
// slots are not changed, only missing slots are added.
func AddImplicitSlots(snapInfo *Info) {
	if snapInfo.Type != TypeOS {
		return
	}
	for _, ifaceName := range implicitSlots {
		if _, ok := snapInfo.Slots[ifaceName]; !ok {
			snapInfo.Slots[ifaceName] = makeImplicitSlot(snapInfo, ifaceName)
		}
	}

	gpioCreateImplicitSlots(snapInfo)

	if !release.OnClassic {
		return
	}
	for _, ifaceName := range implicitClassicSlots {
		if _, ok := snapInfo.Slots[ifaceName]; !ok {
			snapInfo.Slots[ifaceName] = makeImplicitSlot(snapInfo, ifaceName)
		}
	}
}

func makeImplicitSlot(snapInfo *Info, ifaceName string) *SlotInfo {
	return &SlotInfo{
		Name:      ifaceName,
		Snap:      snapInfo,
		Interface: ifaceName,
	}
}

// addImplicitHooks adds hooks from the installed snap's hookdir to the snap info.
//
// Existing hooks (i.e. ones defined in the YAML) are not changed; only missing
// hooks are added.
func addImplicitHooks(snapInfo *Info) error {
	// First of all, check to ensure the hooks directory exists. If it doesn't,
	// it's not an error-- there's just nothing to do.
	hooksDir := snapInfo.HooksDir()
	if !osutil.IsDirectory(hooksDir) {
		return nil
	}

	fileInfos, err := ioutil.ReadDir(hooksDir)
	if err != nil {
		return fmt.Errorf("unable to read hooks directory: %s", err)
	}

	for _, fileInfo := range fileInfos {
		addHookName(snapInfo, fileInfo.Name())
	}

	return nil
}

// addImplicitHooksFromContainer adds hooks from the snap file's hookdir to the snap info.
//
// Existing hooks (i.e. ones defined in the YAML) are not changed; only missing
// hooks are added.
func addImplicitHooksFromContainer(snapInfo *Info, snapf Container) error {
	// Read the hooks directory. If this fails we assume the hooks directory
	// doesn't exist, which means there are no implicit hooks to load (not an
	// error).
	fileNames, err := snapf.ListDir("meta/hooks")
	if err != nil {
		return nil
	}

	for _, fileName := range fileNames {
		addHookName(snapInfo, fileName)
	}

	return nil
}

func addHookName(snapInfo *Info, hookName string) {
	// Don't overwrite a hook that has already been loaded from the YAML
	if _, ok := snapInfo.Hooks[hookName]; !ok {
		snapInfo.Hooks[hookName] = &HookInfo{Snap: snapInfo, Name: hookName}
	}
}
