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

package checker

import (
	checkerPkg "git.happydns.org/happyDomain/internal/dnschecker"
	"git.happydns.org/happyDomain/model"
)

// fieldMetaOf returns the field metadata of the checker, or nil for a checker
// this build does not load.
func fieldMetaOf(checkerName string) *checkerFieldMeta {
	def := checkerPkg.FindChecker(checkerName)
	if def == nil {
		return nil
	}
	meta := computeFieldMeta(def)
	return &meta
}

// withheld reports whether the value of key leaves happyDomain as
// happydns.RedactedSecret: a secret, and any option the documentation does
// not describe (meta nil for an unloaded checker), as it may be a secret
// stored in clear.
func withheld(meta *checkerFieldMeta, key string, value any) bool {
	if isEmptyValue(value) {
		return false
	}
	if meta == nil {
		return true
	}
	if _, documented := meta.fields[key]; !documented {
		return true
	}
	return meta.secretIds[key]
}

// RedactOptions returns a copy of opts where every option withheld is
// replaced by happydns.RedactedSecret, for whatever leaves happyDomain: the
// API, the export of an account.
func RedactOptions(checkerName string, opts happydns.CheckerOptions) happydns.CheckerOptions {
	if opts == nil {
		return nil
	}

	meta := fieldMetaOf(checkerName)
	out := make(happydns.CheckerOptions, len(opts))
	for k, v := range opts {
		if withheld(meta, k, v) {
			out[k] = happydns.RedactedSecret
		} else {
			out[k] = v
		}
	}
	return out
}

// RedactCheckerOptions is RedactOptions, for the API.
func (u *CheckerOptionsUsecase) RedactCheckerOptions(checkerName string, opts happydns.CheckerOptions) happydns.CheckerOptions {
	return RedactOptions(checkerName, opts)
}

// RedactCheckerOptionValue returns value, or happydns.RedactedSecret when
// RedactOptions would withhold it.
func (u *CheckerOptionsUsecase) RedactCheckerOptionValue(checkerName, optName string, value any) any {
	if withheld(fieldMetaOf(checkerName), optName, value) {
		return happydns.RedactedSecret
	}
	return value
}
