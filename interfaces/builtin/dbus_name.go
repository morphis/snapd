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
	"github.com/ubuntu-core/snappy/snap"
)

var dbusNamePermanentPlugAppArmor = []byte(`
#include <abstractions/dbus-strict>

# Allow request/release a dbus name
dbus (send)
  bus=system
  path=/org/freedesktop/DBus
  interface=org.freedesktop.DBus
  member={Request,Release}Name
  peer=(name=org.freedesktop.DBus),
`)

var dbusNameConnectedPlugAppArmor = []byte(`
# Allow binding the service to the requested connection name
dbus (bind)
  bus=system
  name="###PLUG_BUS_NAME###",
`)

var dbusNameConnectedPlugSecComp = []byte(`
# Description: Allow dbus service.

# Can communicate with DBus system service
connect
getsockname
recv
recvmsg
send
sendto
sendmsg
socket
`)

var dbusNameConnectedPlugDBus = []byte(`
<policy user="root">
	<allow own="###PLUG_BUS_NAME###"/>
	<allow send_destination="###PLUG_BUS_NAME###"/>
</policy>
<policy context="default">
	# By default deny all access for not-root
	<deny send_destination="###PLUG_BUS_NAME###"/>
</policy>
`)

type DBusNameInterface struct{}

func (iface *DBusNameInterface) Name() string {
	return "dbus-name"
}

func (iface *DBusNameInterface) PermanentPlugSnippet(plug *interfaces.Plug, securitySystem interfaces.SecuritySystem) ([]byte, error) {
	switch securitySystem {
	case interfaces.SecurityAppArmor:
		return dbusNamePermanentPlugAppArmor, nil
	case interfaces.SecurityDBus, interfaces.SecuritySecComp, interfaces.SecurityUDev:
		return nil, nil
	default:
		return nil, interfaces.ErrUnknownSecurity
	}
}

var plugBusNamePlaceholder = []byte("###PLUG_BUS_NAME###")

func (iface *DBusNameInterface) ConnectedPlugSnippet(plug *interfaces.Plug, slot *interfaces.Slot, securitySystem interfaces.SecuritySystem) ([]byte, error) {
	bus_name, _ := plug.Attrs["name"].(string)

	switch securitySystem {
	case interfaces.SecurityDBus:
		snippet := bytes.Replace(dbusNameConnectedPlugDBus, plugBusNamePlaceholder, []byte(bus_name), -1)
		return snippet, nil
	case interfaces.SecurityAppArmor:
		snippet := bytes.Replace(dbusNameConnectedPlugAppArmor, plugBusNamePlaceholder, []byte(bus_name), -1)
		return snippet, nil
	case interfaces.SecuritySecComp:
		return dbusNameConnectedPlugSecComp, nil
	case interfaces.SecurityUDev:
		return nil, nil
	default:
		return nil, interfaces.ErrUnknownSecurity
	}
}

func (iface *DBusNameInterface) PermanentSlotSnippet(slot *interfaces.Slot, securitySystem interfaces.SecuritySystem) ([]byte, error) {
	switch securitySystem {
	case interfaces.SecurityDBus, interfaces.SecurityAppArmor, interfaces.SecuritySecComp, interfaces.SecurityUDev:
		return nil, nil
	default:
		return nil, interfaces.ErrUnknownSecurity
	}
}

func (iface *DBusNameInterface) ConnectedSlotSnippet(plug *interfaces.Plug, slot *interfaces.Slot, securitySystem interfaces.SecuritySystem) ([]byte, error) {
	switch securitySystem {
	case interfaces.SecurityDBus, interfaces.SecurityAppArmor, interfaces.SecuritySecComp, interfaces.SecurityUDev:
		return nil, nil
	default:
		return nil, interfaces.ErrUnknownSecurity
	}
}

func (iface *DBusNameInterface) SanitizePlug(plug *interfaces.Plug) error {
	if iface.Name() != plug.Interface {
		panic(fmt.Sprintf("plug is not of interface %q", iface))
	}
	name, ok := plug.Attrs["name"].(string)
	if !ok || name == "" {
		return fmt.Errorf("dbus-name must contain name attribute")
	}
	return nil
}

func (iface *DBusNameInterface) SanitizeSlot(slot *interfaces.Slot) error {
	if slot.Snap.Type != snap.TypeOS {
		return fmt.Errorf("%s slots are reserved for the operating system snap", iface.Name())
	}
	return nil
}

func (iface *DBusNameInterface) AutoConnect() bool {
	return true
}
