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
	"bytes"

	"github.com/ubuntu-core/snappy/interfaces"
)

var dbusAccessPermanentSlotAppArmor = []byte(`
#include <abstractions/dbus-strict>

# Allow binding the service to the requested connection name
dbus (receive, send)
  bus=system
  path=###SLOT_PATH_NAME###
  peer=(label=###SLOT_SECURITY_TAGS##),
`)

type DBusAccessInterface struct{}

func (iface *DBusAccessInterface) Name() string {
	return "dbus-access"
}

func (iface *DBusAccessInterface) PermanentPlugSnippet(plug *interfaces.Plug, securitySystem interfaces.SecuritySystem) ([]byte, error) {
	switch securitySystem {
	case interfaces.SecurityAppArmor:
		return nil, nil
	case interfaces.SecurityDBus, interfaces.SecuritySecComp, interfaces.SecurityUDev:
		return nil, nil
	default:
		return nil, interfaces.ErrUnknownSecurity
	}
}


func (iface *DBusAccessInterface) ConnectedPlugSnippet(plug *interfaces.Plug, slot *interfaces.Slot, securitySystem interfaces.SecuritySystem) ([]byte, error) {
	if plug.Snap.Name() != slot.Snap.Name() {
		return nil, interfaces.ErrNotAllowed
	}
	switch securitySystem {
	case interfaces.SecurityDBus:
		return nil, nil
	case interfaces.SecurityAppArmor:
		return nil, nil
	case interfaces.SecuritySecComp:
		return dbusNameConnectedPlugSecComp, nil
	case interfaces.SecurityUDev:
		return nil, nil
	default:
		return nil, interfaces.ErrUnknownSecurity
	}
}

func (iface *DBusAccessInterface) PermanentSlotSnippet(slot *interfaces.Slot, securitySystem interfaces.SecuritySystem) ([]byte, error) {
	path, _ := slot.Attrs["path"].(string)
	
	switch securitySystem {
	case interfaces.SecurityAppArmor:
		snippet := bytes.Replace(dbusAccessPermanentSlotAppArmor, []byte("###SLOT_PATH_NAME###"), []byte(path), -1)
		snippet = bytes.Replace(snippet, []byte("###SLOT_SECURITY_TAGS###"), slotAppLabelExpr(slot), -1)
		return nil, nil
	case interfaces.SecurityDBus, interfaces.SecuritySecComp, interfaces.SecurityUDev:
		return nil, nil
	default:
		return nil, interfaces.ErrUnknownSecurity
	}
}

func (iface *DBusAccessInterface) ConnectedSlotSnippet(plug *interfaces.Plug, slot *interfaces.Slot, securitySystem interfaces.SecuritySystem) ([]byte, error) {
	switch securitySystem {
	case interfaces.SecurityDBus, interfaces.SecurityAppArmor, interfaces.SecuritySecComp, interfaces.SecurityUDev:
		return nil, nil
	default:
		return nil, interfaces.ErrUnknownSecurity
	}
}

func (iface *DBusAccessInterface) SanitizePlug(plug *interfaces.Plug) error {
	if iface.Name() != plug.Interface {
		panic(fmt.Sprintf("plug is not of interface %q", iface))
	}
	return nil
}

func (iface *DBusAccessInterface) SanitizeSlot(slot *interfaces.Slot) error {
	return nil
}

func (iface *DBusAccessInterface) AutoConnect() bool {
	return false
}
