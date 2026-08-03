// This file is part of the happyDomain (R) project.
// Copyright (c) 2022-2024 happyDomain
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

// REDACTED_SECRET is what the API sends in place of a stored value whose field
// is tagged `secret`. Submitting it back unchanged means "keep the value
// already stored"; the server resolves it (happydns.RedactedSecret in
// model/form.go, forms.MergeSecrets), so it must be sent back verbatim and
// never edited in part.
export const REDACTED_SECRET = "••••••••";

// REDACTED_SECRET_B64 is the same sentinel as it appears on the wire for a
// `[]byte`-typed secret field: encoding/json base64-encodes byte slices, and
// the server sets those to the raw UTF-8 bytes of REDACTED_SECRET before
// encoding, so the literal string never shows up for those fields.
export const REDACTED_SECRET_B64 = btoa(
    String.fromCharCode(...new TextEncoder().encode(REDACTED_SECRET)),
);

export class Field {
    id = $state<string>('');
    type = $state<string>('');
    label? = $state<string>();
    description? = $state<string>();
    placeholder? = $state<string>();
    default? = $state<string>();
    choices? = $state<Array<string>>();
    hide? = $state<boolean>();
    required? = $state<boolean>();
    secret? = $state<boolean>();
    textarea? = $state<boolean>();
    autoFill? = $state<string>();
}

export class CustomForm {
    beforeText? = $state<string>();
    sideText? = $state<string>();
    afterText? = $state<string>();
    fields = $state<Array<Field>>([]);
    nextButtonText? = $state<string>();
    nextEditButtonText? = $state<string>();
    previousButtonText? = $state<string>();
    previousEditButtonText? = $state<string>();
    nextButtonLink? = $state<string>();
    nextButtonState? = $state<number>();
    previousButtonLink? = $state<string>();
    previousButtonState? = $state<number>();
}

export class FormState {
    _id? = $state<string>();
    _comment? = $state<string>();
    state = $state<number>(0);
    recall? = $state<string>();
    redirect? = $state<string>();
}

export class FormResponse<T> {
    form? = $state<CustomForm>();
    values? = $state<T>();
    redirect? = $state<string>();
}
