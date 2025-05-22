// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2024 happyDomain
// Authors: Pierre-Olivier Mercier, et al.
//
// This program is offered under a commercial and under the AGPL license.
// For commercial licensing, contact us at <contact@happydomain.org>.
//
// For AGPL licensing:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

//go:build web

package web

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
)

//go:generate npm run build
//go:embed all:build

var _assets embed.FS

var Assets http.FileSystem

func GetEmbedFS() embed.FS {
	return _assets
}

func init() {
	sub, err := fs.Sub(_assets, "build")
	if err != nil {
		log.Fatal("Unable to cd to build/ directory:", err)
	}
	Assets = http.FS(sub)
}
