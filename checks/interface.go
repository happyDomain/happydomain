// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2026 happyDomain
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

// Package checks provides the registry for domain health checkers.
// It allows individual checker implementations to self-register at startup
// via init() functions and exposes functions to retrieve registered checkers.
package checks // import "git.happydns.org/happyDomain/checks"

import (
	"encoding/json"
	"fmt"
	"log"

	"git.happydns.org/happyDomain/model"
)

// checkersList is the ordered list of all registered checks.
var checkersList map[string]happydns.Checker = map[string]happydns.Checker{}

// RegisterChecker declares the existence of the given check. It is intended to
// be called from init() functions in individual check files so that each check
// self-registers at program startup.
//
// If two checks try to register the same environment name the program will
// terminate: name collisions are a configuration error, not a runtime one.
func RegisterChecker(name string, checker happydns.Checker) {
	log.Println("Registering new checker:")
	checkersList[name] = checker
}

// GetCheckers returns the ordered list of all registered checks.
func GetCheckers() *map[string]happydns.Checker {
	return &checkersList
}

// FindChecker returns the check registered under the given environment name,
// or an error if no check with that name exists.
func FindChecker(name string) (happydns.Checker, error) {
	c, ok := checkersList[name]
	if !ok {
		return nil, fmt.Errorf("unable to find check %q", name)
	}
	return c, nil
}

// GetHTMLReport renders an HTML report for the given checker and raw JSON report data.
// Returns (html, true, nil) if the checker supports HTML reports, or ("", false, nil) if not.
func GetHTMLReport(checker happydns.Checker, raw json.RawMessage) (string, bool, error) {
	hr, ok := checker.(happydns.CheckerHTMLReporter)
	if !ok {
		return "", false, nil
	}
	html, err := hr.GetHTMLReport(raw)
	return html, true, err
}
