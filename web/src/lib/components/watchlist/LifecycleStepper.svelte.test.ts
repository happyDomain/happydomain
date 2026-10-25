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

import { render } from "@testing-library/svelte";
import { beforeAll, describe, expect, it } from "vitest";

import type { DomainWatchPhase } from "$lib/model/watchlist";
import { loadTranslations } from "$lib/translations";
import LifecycleStepper from "./LifecycleStepper.svelte";

beforeAll(async () => {
    await loadTranslations("en", "/");
});

// steps tells, for each step, whether it is the current one ("current"), one
// walked through ("walked") or one to come ("").
function steps(phase: DomainWatchPhase | undefined): string[] {
    const { container } = render(LifecycleStepper, { phase });
    return [...container.querySelectorAll(".step")].map((li) =>
        li.getAttribute("aria-current") === "step"
            ? "current"
            : li.querySelector(".progress-bar")
              ? "walked"
              : "",
    );
}

describe("LifecycleStepper", () => {
    it("shows the steps a domain went through", () => {
        expect(steps("redemption")).toEqual(["walked", "walked", "current", "", ""]);
    });

    it("places an expiring domain on the registered step", () => {
        expect(steps("expiring")).toEqual(["current", "", "", "", ""]);
    });

    it("does not pretend an available domain went through the previous steps", () => {
        expect(steps("available")).toEqual(["", "", "", "", "current"]);
    });

    it("has no current step for a domain not checked yet", () => {
        expect(steps("unknown")).toEqual(["", "", "", "", ""]);
        expect(steps(undefined)).toEqual(["", "", "", "", ""]);
    });
});
