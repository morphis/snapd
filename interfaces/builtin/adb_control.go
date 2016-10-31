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
	"github.com/snapcore/snapd/interfaces"
)

const adbControlConnectedPlugAppArmor = `
# Description: Allow managing the kernel side adb stack. Reserved
#  because this gives privileged access to the system.
# Usage: reserved

/dev/adb rw,
`

func NewAdbControlInterface() interfaces.Interface {
	return &commonInterface{
		name: "adb-control",
		connectedPlugAppArmor: adbControlConnectedPlugAppArmor,
		reservedForOS:         true,
	}
}
