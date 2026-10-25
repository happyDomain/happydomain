// This file is part of the happyDomain (R) project.
// Copyright (c) 2022-2026 happyDomain
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

import { render, screen } from "@testing-library/svelte";
import { beforeAll, describe, expect, it } from "vitest";

import { loadTranslations } from "$lib/translations";
import DomainStatusBadges from "./DomainStatusBadges.svelte";

beforeAll(async () => {
    await loadTranslations("en", "/");
});

describe("DomainStatusBadges", () => {
    it("describes a single known status", () => {
        render(DomainStatusBadges, { status: ["clientTransferProhibited"] });

        const badge = screen.getByText("clientTransferProhibited");
        expect(badge).toHaveAttribute("title", "Transfer to another registrar is locked.");
        expect(screen.getByText("Transfer to another registrar is locked.")).toBeVisible();
    });

    it("tells the code of a status it cannot describe", () => {
        const { container } = render(DomainStatusBadges, { status: ["someRegistryStatus"] });

        expect(screen.getByText("someRegistryStatus")).toHaveAttribute(
            "title",
            "someRegistryStatus",
        );
        expect(container.querySelector("p")).toBeNull();
    });

    it("describes several statuses only on hover", () => {
        const { container } = render(DomainStatusBadges, {
            status: ["clientTransferProhibited", "clientHold"],
        });

        expect(screen.getByText("clientHold")).toHaveAttribute(
            "title",
            "The domain is on hold (not resolving).",
        );
        expect(container.querySelector("p")).toBeNull();
    });

    it("shows a status listed twice", () => {
        render(DomainStatusBadges, { status: ["ok", "ok"] });

        expect(screen.getAllByText("ok")).toHaveLength(2);
    });

    it("leaves the single status undescribed when asked to", () => {
        const { container } = render(DomainStatusBadges, {
            status: ["clientTransferProhibited"],
            describeSingle: false,
        });

        expect(screen.getByText("clientTransferProhibited")).toHaveAttribute(
            "title",
            "Transfer to another registrar is locked.",
        );
        expect(container.querySelector("p")).toBeNull();
    });
});
