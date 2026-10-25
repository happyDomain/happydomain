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

/**
 * Stands in for the Web Animations API, which jsdom lacks, so that the
 * transitions and animations of the components end at once instead of
 * throwing.
 */
export function stubWebAnimations(): void {
    Element.prototype.getAnimations = () => [];
    Element.prototype.animate = function () {
        const animation: Partial<Animation> = { cancel() {}, finish() {} };
        let onfinish: Animation["onfinish"] = null;
        Object.defineProperty(animation, "onfinish", {
            get: () => onfinish,
            set: (fn: Animation["onfinish"]) => {
                onfinish = fn;
                queueMicrotask(() =>
                    onfinish?.call(animation as Animation, {} as AnimationPlaybackEvent),
                );
            },
        });
        return animation as Animation;
    };
}
