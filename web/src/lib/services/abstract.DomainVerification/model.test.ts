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

import { describe, it, expect } from "vitest";

import { getRrtype, newRR, type dnsTypeTXT } from "$lib/dns_rr";
import {
    findVerificationProvider,
    selectVerificationProvider,
    setVerificationValue,
} from "./model";

function txt(name: string, value: string): dnsTypeTXT {
    const rr = newRR(name, getRrtype("TXT")) as dnsTypeTXT;
    rr.Txt = value;
    return rr;
}

describe("findVerificationProvider", () => {
    it("tells the provider from the value", () => {
        expect(findVerificationProvider(txt("", "google-site-verification=abc"))?.name).toBe("Google");
        expect(
            findVerificationProvider(txt("", "anthropic-domain-verification-4az7qn=if8Y"))?.name,
        ).toBe("Anthropic");
    });

    it("tells the provider from the label", () => {
        expect(findVerificationProvider(txt("_keybase", "keybase-site-verification=abc"))?.name).toBe(
            "Keybase",
        );
        expect(findVerificationProvider(txt("_scaleway-challenge", "token"))?.name).toBe("Scaleway");
    });

    it("finds nothing for an unknown record or the wrong label", () => {
        expect(findVerificationProvider(txt("", "hello"))).toBeUndefined();
        expect(findVerificationProvider(undefined)).toBeUndefined();
    });
});

describe("setVerificationValue", () => {
    it("keeps an apex value on the apex", () => {
        const rr = txt("", "");
        setVerificationValue(rr, " stripe-verification=f88e ");
        expect(rr.Txt).toBe("stripe-verification=f88e");
        expect(rr.Hdr.Name).toBe("");
    });

    it("moves the record under the label of its provider", () => {
        const rr = txt("", "");
        setVerificationValue(rr, "did=did:plc:z72i7hdynmk6r22z27h6tvur");
        expect(rr.Hdr.Name).toBe("_atproto");
    });

    it("keeps a value its provider also accepts on the apex there", () => {
        const rr = txt("", "");
        setVerificationValue(rr, "keybase-site-verification=abc");
        expect(rr.Hdr.Name).toBe("");
        expect(findVerificationProvider(rr)?.name).toBe("Keybase");
    });

    it("leaves the label of a provider taking any value", () => {
        const rr = txt("_scaleway-challenge", "");
        setVerificationValue(rr, "token");
        expect(rr.Hdr.Name).toBe("_scaleway-challenge");
    });
});

describe("selectVerificationProvider", () => {
    it("swaps the prefix and keeps the code", () => {
        const rr = txt("", "google-site-verification=abc");
        selectVerificationProvider(rr, "Keybase");
        expect(rr.Hdr.Name).toBe("_keybase");
        expect(rr.Txt).toBe("keybase-site-verification=abc");
    });

    it("prefixes a bare code", () => {
        const rr = txt("", "");
        selectVerificationProvider(rr, "Stripe");
        expect(rr.Txt).toBe("stripe-verification=");
    });

    it("keeps a value already matching the provider", () => {
        const rr = txt("", "anthropic-domain-verification-4az7qn=if8Y");
        selectVerificationProvider(rr, "Anthropic");
        expect(rr.Txt).toBe("anthropic-domain-verification-4az7qn=if8Y");
    });

    it("applies the first form of a provider having several", () => {
        const rr = txt("", "");
        selectVerificationProvider(rr, "Twilio");
        expect(rr.Hdr.Name).toBe("_twilio");
        expect(rr.Txt).toBe("twilio-domain-verification=");
    });

    it("moves a value taking any content under its label", () => {
        const rr = txt("", "KaBKBH9D");
        selectVerificationProvider(rr, "Amazon SES");
        expect(rr.Hdr.Name).toBe("_amazonses");
        expect(rr.Txt).toBe("KaBKBH9D");
    });
});

describe("several forms of a provider", () => {
    it("keeps the apex form of Twilio on the apex", () => {
        const rr = txt("", "");
        setVerificationValue(rr, "twilio-domain-verification=7d1f");
        expect(rr.Hdr.Name).toBe("");
        expect(findVerificationProvider(rr)?.name).toBe("Twilio");
    });

    it("tells Amazon SES under its label", () => {
        expect(findVerificationProvider(txt("_amazonses", "KaBKBH9D"))?.name).toBe("Amazon SES");
        expect(findVerificationProvider(txt("", "amazonses:KaBKBH9D"))?.name).toBe("Amazon SES");
    });
});
