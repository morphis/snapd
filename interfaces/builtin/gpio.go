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

package builtin

import (
	"fmt"
	"os"
	"strconv"

	"github.com/snapcore/snapd/interfaces"
)

var gpioSysfsGpioBase = "/sys/class/gpio/gpio"
var gpioSysfsExport = "/sys/class/gpio/export"

type GpioInterface struct{}

func (iface *GpioInterface) Name() string {
	return "gpio"
}

func (iface *GpioInterface) PermanentPlugSnippet(plug *interfaces.Plug, securitySystem interfaces.SecuritySystem) ([]byte, error) {
	switch securitySystem {
	case interfaces.SecurityDBus, interfaces.SecurityAppArmor, interfaces.SecuritySecComp, interfaces.SecurityUDev, interfaces.SecurityMount:
		return nil, nil
	default:
		return nil, interfaces.ErrUnknownSecurity
	}
}

func (iface *GpioInterface) ConnectedPlugSnippet(plug *interfaces.Plug, slot *interfaces.Slot, securitySystem interfaces.SecuritySystem) ([]byte, error) {
	switch securitySystem {
	case interfaces.SecurityDBus, interfaces.SecuritySecComp, interfaces.SecurityUDev, interfaces.SecurityMount:
		return nil, nil
	case interfaces.SecurityAppArmor:
		path := fmt.Sprint(gpioSysfsGpioBase, slot.Attrs["number"])
		// Entries in /sys/class/gpio for single GPIO's are just symlinks
		// to their correct device part in the sysfs tree. As AppArmor
		// does not handle symlinks we need to dereference the GPIO
		// path and add the correct absolute path to the AppArmor snippet.
		dereferencedPath, err := evalSymlinks(path)
		if err != nil {
			return nil, err
		}
		return []byte(fmt.Sprintf("%s/** rwk,\n", dereferencedPath)), nil
	default:
		return nil, interfaces.ErrUnknownSecurity
	}
}

func (iface *GpioInterface) PermanentSlotSnippet(slot *interfaces.Slot, securitySystem interfaces.SecuritySystem) ([]byte, error) {
	switch securitySystem {
	case interfaces.SecurityDBus, interfaces.SecurityAppArmor, interfaces.SecuritySecComp, interfaces.SecurityUDev, interfaces.SecurityMount:
		return nil, nil
	default:
		return nil, interfaces.ErrUnknownSecurity
	}
}

func (iface *GpioInterface) ConnectedSlotSnippet(plug *interfaces.Plug, slot *interfaces.Slot, securitySystem interfaces.SecuritySystem) ([]byte, error) {
	// We need to export the GPIO so that it becomes as entry in sysfs
	// availalbe and we can assign it to a connecting plug.
	number := []byte(strconv.Itoa(slot.Attrs["number"].(int)))
	fileExport, err := os.OpenFile(gpioSysfsExport, os.O_WRONLY, 0200)
	defer fileExport.Close()
	if err != nil {
		return nil, err
	}
	fileExport.Write(number)

	switch securitySystem {
	case interfaces.SecurityDBus, interfaces.SecuritySecComp, interfaces.SecurityUDev, interfaces.SecurityMount:
		return nil, nil
	case interfaces.SecurityAppArmor:
		return nil, nil
	default:
		return nil, interfaces.ErrUnknownSecurity
	}
}

func (iface *GpioInterface) SanitizePlug(plug *interfaces.Plug) error {
	return nil
}

func (iface *GpioInterface) SanitizeSlot(slot *interfaces.Slot) error {
	return nil
}

func (iface *GpioInterface) AutoConnect() bool {
	return false
}
