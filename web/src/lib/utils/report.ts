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

export interface Forge {
    name: string;
    // Base URL of the issue tracker, without the trailing `/new`.
    issues: string;
    // Forges don't agree on how a new issue is pre-filled.
    flavor: "github" | "gitlab" | "forgejo";
}

// We mirror the project on several forges, and we read the issues of all of
// them: the reporter opens the one where they already have an account.
export const FORGES: Forge[] = [
    {
        name: "GitHub",
        issues: "https://github.com/happyDomain/happydomain/issues",
        flavor: "github",
    },
    {
        name: "GitLab",
        issues: "https://gitlab.com/happyDomain/happydomain/-/issues",
        flavor: "gitlab",
    },
    {
        name: "Framagit",
        issues: "https://framagit.org/happyDomain/happydomain/-/issues",
        flavor: "gitlab",
    },
    {
        name: "Codeberg",
        issues: "https://codeberg.org/happyDomain/happyDomain/issues",
        flavor: "forgejo",
    },
];

// Not everyone has, or wants, an account on a forge: a mail reaches us just
// as well.
export const REPORT_EMAIL = "contact@happydomain.org";

function diagnosticsBlock(diagnostics: string): string {
    if (!diagnostics.trim()) return "";
    return `<details><summary>Diagnostics</summary>\n\n\`\`\`\n${diagnostics.trim()}\n\`\`\`\n\n</details>`;
}

/**
 * The report as a single Markdown document, for the forges that only accept a
 * whole issue body, and for the clipboard.
 */
export function reportMarkdown(description: string, diagnostics: string): string {
    return [description.trim(), diagnosticsBlock(diagnostics)].filter(Boolean).join("\n\n") + "\n";
}

/**
 * Build the URL opening a new issue, with everything we already know filled
 * in: the reporter only has to press the submit button.
 */
export function issueURL(forge: Forge, description: string, diagnostics: string): string {
    const params = new URLSearchParams();

    switch (forge.flavor) {
        case "github":
            // We ship no issue template on GitHub, so fall back to the
            // plain title/body params every repository supports.
            params.set("labels", "bug");
            params.set("title", description.trim().split("\n")[0] || "Problem report");
            params.set("body", reportMarkdown(description, diagnostics));
            break;
        case "gitlab":
            params.set("issuable_template", "Bug");
            params.set("issue[description]", reportMarkdown(description, diagnostics));
            break;
        case "forgejo":
            // Forgejo pre-fills the issue body, not the fields of a form.
            params.set("body", reportMarkdown(description, diagnostics));
            break;
    }

    return `${forge.issues}/new?${params.toString()}`;
}

/**
 * Build the mail opening the user's mail client, with the same report as the
 * one the forges would receive.
 */
export function mailtoURL(description: string, diagnostics: string): string {
    const firstLine = description.trim().split("\n")[0];
    const subject = firstLine ? `happyDomain: ${firstLine}` : "happyDomain: problem report";

    const params = new URLSearchParams({
        subject,
        body: reportMarkdown(description, diagnostics),
    });

    // Mail clients don't decode `+` as a space in a mailto body, so spell out
    // the spaces the form encoding would have swallowed.
    return `mailto:${REPORT_EMAIL}?${params.toString().replace(/\+/g, "%20")}`;
}
