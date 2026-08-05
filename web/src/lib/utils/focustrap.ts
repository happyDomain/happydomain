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

const FOCUSABLE_SELECTOR = [
    "a[href]",
    "area[href]",
    "button:not([disabled])",
    'input:not([disabled]):not([type="hidden"])',
    "select:not([disabled])",
    "textarea:not([disabled])",
    "iframe",
    "summary",
    '[contenteditable="true"]',
    '[tabindex]:not([tabindex="-1"])',
].join(", ");

// Traps are stacked, so that a dialog opened on top of another one is the only
// one handling Tab until it closes.
const stack: HTMLElement[] = [];

function focusableWithin(container: HTMLElement): HTMLElement[] {
    return Array.from(container.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR)).filter(
        (el) =>
            !el.hasAttribute("inert") &&
            el.getAttribute("aria-hidden") !== "true" &&
            el.getClientRects().length > 0,
    );
}

/**
 * Give the dialog an accessible name taken from its own header, as our dialogs
 * carry their title in a `<ModalHeader>` child instead of sveltestrap's
 * `header` prop (which is what wires up `aria-labelledby` on its own), and
 * sveltestrap's Offcanvas never wires it up at all.
 */
function labelFromTitle(container: HTMLElement, id: string): void {
    if (container.hasAttribute("aria-labelledby") || container.hasAttribute("aria-label")) return;

    const title = container.querySelector<HTMLElement>(".modal-title, .offcanvas-title");
    if (!title) return;

    if (!title.id) title.id = `${id}-title`;
    container.setAttribute("aria-labelledby", title.id);
}

/**
 * Confine keyboard focus to an open dialog: focus moves inside it, Tab and
 * Shift+Tab cycle through its own controls instead of reaching the page
 * underneath, and assistive technologies are told the rest of the page is not
 * available (WCAG 2.2 AA, 2.1.1 Keyboard and 2.4.3 Focus Order).
 *
 * Returns a function releasing the trap. Restoring focus to the element that
 * opened the dialog is left to the caller (sveltestrap's Modal already does it).
 */
export function trapFocus(container: HTMLElement, id: string): () => void {
    // `aria-modal` rather than marking the rest of the page `inert`: toasts are
    // rendered outside dialogs and their live region has to keep announcing
    // while one is open.
    container.setAttribute("aria-modal", "true");
    if (!container.hasAttribute("tabindex")) container.setAttribute("tabindex", "-1");
    labelFromTitle(container, id);

    stack.push(container);

    // Focus the dialog itself rather than its first control, so screen readers
    // announce its name and role before its content.
    container.focus({ preventScroll: true });

    function onKeydown(event: KeyboardEvent) {
        if (event.key !== "Tab" || stack[stack.length - 1] !== container) return;

        const focusables = focusableWithin(container);
        if (focusables.length === 0) {
            event.preventDefault();
            container.focus();
            return;
        }

        const first = focusables[0];
        const last = focusables[focusables.length - 1];
        const active = document.activeElement;

        if (!container.contains(active)) {
            event.preventDefault();
            (event.shiftKey ? last : first).focus();
        } else if (event.shiftKey && (active === first || active === container)) {
            event.preventDefault();
            last.focus();
        } else if (!event.shiftKey && active === last) {
            event.preventDefault();
            first.focus();
        }
    }

    // Captured on the document: focus may have escaped the dialog already (a
    // click on the page behind it, for instance), and we still want it back.
    document.addEventListener("keydown", onKeydown, true);

    return () => {
        document.removeEventListener("keydown", onKeydown, true);

        const index = stack.lastIndexOf(container);
        if (index !== -1) stack.splice(index, 1);

        container.removeAttribute("aria-modal");
    };
}
