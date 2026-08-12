// This file is part of the happyDomain (R) project.
// Copyright (c) 2022-2026 happyDomain
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

import { describe, expect, it } from "vitest";

import { FORGES, issueURL, mailtoURL, REPORT_EMAIL, reportMarkdown } from "./report";

const forge = (name: string) => {
    const f = FORGES.find((f) => f.name === name);
    if (!f) throw new Error(`unknown forge ${name}`);
    return f;
};

describe("reportMarkdown", () => {
    it("keeps the diagnostics in a foldable block", () => {
        const md = reportMarkdown("the page stayed empty", "happyDomain: 1.0.0");
        expect(md).toContain("the page stayed empty");
        expect(md).toContain("<details><summary>Diagnostics</summary>");
        expect(md).toContain("happyDomain: 1.0.0");
    });

    it("omits the block when the user cleared the diagnostics", () => {
        expect(reportMarkdown("nothing works", "  ")).toBe("nothing works\n");
    });
});

describe("issueURL", () => {
    it("fills the fields of the GitHub issue form", () => {
        const url = new URL(issueURL(forge("GitHub"), "it broke", "happyDomain: 1.0.0"));
        expect(url.pathname).toBe("/happyDomain/happydomain/issues/new");
        expect(url.searchParams.get("labels")).toBe("bug");
        expect(url.searchParams.get("title")).toBe("it broke");
        expect(url.searchParams.get("body")).toContain("it broke");
        expect(url.searchParams.get("body")).toContain("happyDomain: 1.0.0");
    });

    it("fills the description of the GitLab issue", () => {
        const url = new URL(issueURL(forge("Framagit"), "it broke", "happyDomain: 1.0.0"));
        expect(url.searchParams.get("issuable_template")).toBe("Bug");
        expect(url.searchParams.get("issue[description]")).toContain("it broke");
        expect(url.searchParams.get("issue[description]")).toContain("happyDomain: 1.0.0");
    });

    it("fills the body of the Forgejo issue", () => {
        const url = new URL(issueURL(forge("Codeberg"), "it broke", "happyDomain: 1.0.0"));
        expect(url.searchParams.get("body")).toContain("it broke");
        expect(url.searchParams.get("body")).toContain("happyDomain: 1.0.0");
    });
});

describe("mailtoURL", () => {
    it("addresses us, with the report as subject and body", () => {
        const url = new URL(mailtoURL("the page stayed empty", "happyDomain: 1.0.0"));
        expect(url.pathname).toBe(REPORT_EMAIL);
        expect(url.searchParams.get("subject")).toBe("happyDomain: the page stayed empty");
        expect(url.searchParams.get("body")).toContain("the page stayed empty");
        expect(url.searchParams.get("body")).toContain("happyDomain: 1.0.0");
    });

    it("keeps a subject when the user described nothing", () => {
        const url = new URL(mailtoURL("", "happyDomain: 1.0.0"));
        expect(url.searchParams.get("subject")).toBe("happyDomain: problem report");
    });

    it("encodes spaces so mail clients don't show plus signs", () => {
        const url = mailtoURL("the page stayed empty", "");
        expect(url).toContain("%20");
        expect(url).not.toContain("+");
    });
});
