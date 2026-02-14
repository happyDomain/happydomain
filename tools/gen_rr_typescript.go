// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2025 happyDomain
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

//go:build ignore
// +build ignore

package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"slices"
	"sort"
	"strings"

	"github.com/miekg/dns"
)

// getSortedTypes returns DNS types sorted by their numeric value for reproducible output
func getSortedTypes() []uint16 {
	types := make([]uint16, 0, len(dns.TypeToString))
	for ty := range dns.TypeToString {
		types = append(types, ty)
	}
	sort.Slice(types, func(i, j int) bool {
		return types[i] < types[j]
	})
	return types
}

func nsrrtype(fd io.Writer) {
	fmt.Fprint(fd, "    switch (input) {\n")
	for _, ty := range getSortedTypes() {
		rr := dns.TypeToString[ty]
		fmt.Fprintf(fd, `        case "%d":
        case %d:
            return %q;
`, ty, ty, rr)
	}
	fmt.Fprint(fd, "        default:\n            return \"#\";\n    }\n")
}

func getRrtype(fd io.Writer) {
	fmt.Fprint(fd, "    switch (input) {\n")
	for _, ty := range getSortedTypes() {
		rr := dns.TypeToString[ty]
		fmt.Fprintf(fd, `        case "%s":
        case "%s":
            return %d;
`, strings.ToUpper(rr), strings.ToLower(rr), ty)
	}
	fmt.Fprint(fd, "        default:\n            throw \"Unknown rrtype \" + input;\n    }\n")
}

func newRR(fd io.Writer) {
	fmt.Fprint(fd, "    const rec = { Hdr: { Name: dn, Rrtype: rrtype, Class: 1, Ttl: 3600 } } as dnsRR;\n\n")
	fmt.Fprint(fd, "    switch (rrtype) {\n")
	for _, ty := range getSortedTypes() {
		rr, ok := dns.TypeToRR[ty]
		if !ok {
			continue
		}

		if ty == dns.TypeNXNAME || ty == dns.TypeOPT || ty == dns.TypeANY {
			continue
		}

		t := reflect.TypeOf(rr()).Elem()

		if t.NumField() == 1 {
			// This is a redirection to another type
			t = t.Field(0).Type
		}

		fmt.Fprintf(fd, "        case %d: // %s\n", ty, dns.TypeToString[ty])
		for i := 0; i < t.NumField(); i++ {
			if t.Field(i).Name == "Hdr" {
				continue
			}
			fmt.Fprintf(fd, "            rec.%s = ", t.Field(i).Name)
			// Check if this type maps to string in TypeScript (same logic as toTSType)
			if t.Field(i).Type.Name() == "[]string" || t.Field(i).Type.Name() == "IP" || t.Field(i).Name == "Txt" {
				fmt.Fprint(fd, `""`)
			} else if t.Field(i).Type.Kind() == reflect.Array || t.Field(i).Type.Kind() == reflect.Slice {
				fmt.Fprint(fd, "[]")
			} else if strings.HasPrefix(t.Field(i).Type.Name(), "uint") || strings.HasPrefix(t.Field(i).Type.Name(), "int") || t.Field(i).Type.Name() == "time.Duration" {
				fmt.Fprint(fd, "0")
			} else {
				fmt.Fprint(fd, `""`)
			}
			fmt.Fprint(fd, ";\n")
		}
		fmt.Fprint(fd, "            return rec;\n")
	}
	fmt.Fprint(fd, "        default: return rec;\n    }\n")
}

// isStringType determines if a type should be quoted in DNS output
func isStringType(t reflect.Type) bool {
	// Check if this is a basic string type
	if t.Name() == "string" {
		return true
	}
	// Don't quote IP addresses - they have a specific format
	if t.Name() == "IP" {
		return false
	}
	// For arrays/slices of strings, we need to check the element type
	if t.Kind() == reflect.Array || t.Kind() == reflect.Slice {
		if t.Elem().Name() == "string" {
			return true
		}
	}
	return false
}

func rdatatostr(fd io.Writer) {
	fmt.Fprint(fd, "    switch (rr.Hdr.Rrtype) {\n")
	for _, ty := range getSortedTypes() {
		rr, ok := dns.TypeToRR[ty]
		if !ok {
			continue
		}

		if ty == dns.TypeNXNAME || ty == dns.TypeOPT || ty == dns.TypeANY {
			continue
		}

		t := reflect.TypeOf(rr()).Elem()

		if t.NumField() == 1 {
			// This is a redirection to another type
			t = t.Field(0).Type
		}

		fmt.Fprintf(fd, `        case %d: { const rec = rr as dnsType%s; return `, ty, strings.Replace(dns.TypeToString[ty], "-", "_", -1))
		if ty == dns.TypeNAPTR {
			fmt.Fprint(fd, `[rec.Order, rec.Preference, JSON.stringify(String(rec.Flags)), JSON.stringify(String(rec.Service)), JSON.stringify(String(rec.Regexp)), quoteStringIfNeeded(rec.Replacement)].join(' ')`)
		} else if ty == dns.TypeAPL {
			fmt.Fprint(fd, `rec.Prefixes.map((a) => {
        let ret = "";

        if (a.Negation)
            ret += "!";

        if (a.Network.IP.indexOf(':'))
            ret += "2";
        else
            ret += "1";

        ret += ":";
        ret += a.Network.IP;
        ret += "/";
        ret += a.Network.Mask;
        return ret.length + ret;
    }).join(' ')`)
		} else if t.NumField() == 2 {
			var field reflect.StructField
			if t.Field(0).Name == "Hdr" {
				field = t.Field(1)
			} else {
				field = t.Field(0)
			}
			if isStringType(field.Type) {
				fmt.Fprintf(fd, `quoteStringIfNeeded(rec.%s.toString())`, field.Name)
			} else {
				fmt.Fprintf(fd, `rec.%s.toString()`, field.Name)
			}
		} else {
			fmt.Fprint(fd, "[")
			one := false
			for i := 0; i < t.NumField(); i++ {
				if t.Field(i).Name == "Hdr" {
					continue
				}
				if one {
					fmt.Fprint(fd, ", ")
				}

				// Check if this is a string type that needs quoting
				if isStringType(t.Field(i).Type) {
					if t.Field(i).Type.Kind() == reflect.Array || t.Field(i).Type.Kind() == reflect.Slice {
						// Array of strings - quote each element
						fmt.Fprintf(fd, "rec.%s.map(s => quoteStringIfNeeded(s)).join(' ')", t.Field(i).Name)
					} else {
						// Single string
						fmt.Fprintf(fd, "quoteStringIfNeeded(rec.%s.toString())", t.Field(i).Name)
					}
				} else {
					// Non-string type (number, IP, etc.)
					fmt.Fprintf(fd, "rec.%s", t.Field(i).Name)
					if t.Field(i).Type.Name() != "IP" && (t.Field(i).Type.Kind() == reflect.Array || t.Field(i).Type.Kind() == reflect.Slice) {
						fmt.Fprint(fd, ".join(' ')")
					} else {
						fmt.Fprint(fd, ".toString()")
					}
				}
				one = true
			}
			fmt.Fprint(fd, "].join(' ')")
		}
		fmt.Fprintf(fd, "; } // %s\n", dns.TypeToString[ty])
	}
	fmt.Fprint(fd, "        default: return 'unknown #' + rr.Hdr.Rrtype\n    }\n")
}

func rdataFields(fd io.Writer) {
	fmt.Fprint(fd, "    switch (input) {\n")
	for _, ty := range getSortedTypes() {
		rr, ok := dns.TypeToRR[ty]
		if !ok {
			continue
		}

		if ty == dns.TypeNXNAME || ty == dns.TypeOPT || ty == dns.TypeANY {
			continue
		}

		t := reflect.TypeOf(rr()).Elem()

		if t.NumField() == 1 {
			// This is a redirection to another type
			t = t.Field(0).Type
		}

		fmt.Fprintf(fd, `        case %d: case %q: return [`, ty, dns.TypeToString[ty])
		one := false
		for i := 0; i < t.NumField(); i++ {
			if t.Field(i).Name == "Hdr" {
				continue
			}
			if one {
				fmt.Fprint(fd, ", ")
			}
			fmt.Fprintf(fd, "%q", t.Field(i).Name)
			one = true
		}
		fmt.Fprintf(fd, "]; // %s\n", dns.TypeToString[ty])
	}
	fmt.Fprint(fd, "        default: return [];\n    }\n")
}

func dnsrr(fd io.Writer) {
	var seen []string
	var alltypes [][]reflect.Type
	for _, ty := range getSortedTypes() {
		rr, ok := dns.TypeToRR[ty]
		if !ok {
			continue
		}

		t := reflect.TypeOf(rr()).Elem()

		if t.NumField() == 1 {
			// This is a redirection to another type
			continue
		}

		for i := 0; i < t.NumField(); i++ {
			idx := slices.Index(seen, t.Field(i).Name)
			if idx >= 0 {
				if t.Field(i).Name != "Hdr" {
					alltypes[idx] = append(alltypes[idx], t.Field(i).Type)
				}
				continue
			}

			seen = append(seen, t.Field(i).Name)
			alltypes = append(alltypes, []reflect.Type{t.Field(i).Type})
		}
	}

	fmt.Fprint(fd, "    [key: string]: any;\n")
	for i, rrs := range alltypes {
		if seen[i] == "Hdr" {
			fmt.Fprintf(fd, "    %s: ", seen[i])
		} else {
			fmt.Fprintf(fd, "    %s?: ", seen[i])
		}
		if seen[i] == "Txt" {
			fmt.Fprint(fd, "string")
		} else {
			var sumrr []string
			for _, rr := range rrs {
				tst := toTSType(rr, 1)
				if !slices.Contains(sumrr, tst) {
					sumrr = append(sumrr, tst)
				}
			}
			fmt.Fprint(fd, strings.Join(sumrr, " | "))
		}
		fmt.Fprint(fd, ";\n")
	}
}

func toTSType(t reflect.Type, indent int) string {
	fd := &bytes.Buffer{}

	if t.Name() == "uint8" || t.Name() == "uint16" || t.Name() == "uint32" || t.Name() == "uint64" {
		fmt.Fprintf(fd, "number")
	} else if t.Name() == "bool" {
		fmt.Fprintf(fd, "boolean")
	} else if t.Name() == "string" || t.Name() == "[]string" || t.Name() == "IP" {
		fmt.Fprintf(fd, "string")
	} else if t.Name() == "EDNS0" || t.Name() == "SVCBKeyValue" {
		fmt.Fprint(fd, t.Name())
	} else if t.Kind() == reflect.Array || t.Kind() == reflect.Slice {
		fmt.Fprint(fd, "Array<")
		fmt.Fprint(fd, toTSType(t.Elem(), indent+1))
		fmt.Fprint(fd, ">")
	} else {
		fmt.Fprintf(fd, "{\n")
		for i := 0; i < t.NumField(); i++ {
			if t.Field(i).Name == "Txt" {
				fmt.Fprintf(fd, "%s%s: string;\n", strings.Repeat("    ", indent+1), t.Field(i).Name)
			} else {
				fmt.Fprintf(fd, "%s%s: ", strings.Repeat("    ", indent+1), t.Field(i).Name)
				fmt.Fprint(fd, toTSType(t.Field(i).Type, indent+1))
				fmt.Fprintf(fd, ";\n")
			}
		}
		fmt.Fprintf(fd, "%s}", strings.Repeat("    ", indent))
	}

	return fd.String()
}

func main() {
	output := os.Args[1]

	fd, err := os.Create(output)
	if err != nil {
		panic(err)
	}
	defer fd.Close()

	fmt.Fprint(fd, "// This file is generated by go generate\n\n")

	fmt.Fprintln(fd, "export interface SVCBKeyValue {}\n")
	fmt.Fprintln(fd, "export interface EDNS0 {}\n")

	// Helper function to quote strings containing special DNS characters
	fmt.Fprintln(fd, "// Helper function to quote strings containing special DNS characters (spaces, quotes, semicolons, parentheses)")
	fmt.Fprintln(fd, "function quoteStringIfNeeded(s: string): string {")
	fmt.Fprintln(fd, "    if (s.match(/[\\s;()\\\"]/)) {")
	fmt.Fprintln(fd, "        return JSON.stringify(s);")
	fmt.Fprintln(fd, "    }")
	fmt.Fprintln(fd, "    return s;")
	fmt.Fprintln(fd, "}\n")

	// dnsRR
	fmt.Fprint(fd, "export interface dnsRR {\n")
	dnsrr(fd)
	fmt.Fprint(fd, "}\n\n")

	// dnsType
	for _, ty := range getSortedTypes() {
		rr, ok := dns.TypeToRR[ty]
		if !ok {
			continue
		}

		t := reflect.TypeOf(rr()).Elem()

		fmt.Fprint(fd, "export interface dnsType"+strings.Replace(dns.TypeToString[ty], "-", "_", -1))

		if t.NumField() == 1 {
			// This is a redirection to another type
			t = t.Field(0).Type
		}

		fmt.Fprint(fd, " ", toTSType(t, 0))
		fmt.Fprintf(fd, "\n\n")
	}

	// dnsResource
	fmt.Fprint(fd, "export interface dnsResource {\n")
	for _, ty := range getSortedTypes() {
		rr, ok := dns.TypeToRR[ty]
		if !ok {
			continue
		}

		t := reflect.TypeOf(rr()).Elem()

		if t.NumField() == 1 {
			// This is a redirection to another type
			t = t.Field(0).Type
		}

		// Records that are typically used as arrays in service bodies
		if ty == dns.TypeCAA || ty == dns.TypeMX || ty == dns.TypeNS || ty == dns.TypeDS {
			fmt.Fprintf(fd, "    %s?: dnsType%s | Array<dnsType%s>;\n", strings.Replace(strings.ToLower(dns.TypeToString[ty]), "-", "_", -1), strings.Replace(dns.TypeToString[ty], "-", "_", -1), strings.Replace(dns.TypeToString[ty], "-", "_", -1))
		} else {
			fmt.Fprintf(fd, "    %s?: dnsType%s;\n", strings.Replace(strings.ToLower(dns.TypeToString[ty]), "-", "_", -1), strings.Replace(dns.TypeToString[ty], "-", "_", -1))
		}
	}
	fmt.Fprint(fd, "}\n\n")

	// getRrtype
	fmt.Fprint(fd, "export function getRrtype(input: string): number {\n")
	getRrtype(fd)
	fmt.Fprint(fd, "}\n\n")

	// newRR
	fmt.Fprint(fd, "export function newRR(dn: string, rrtype: number): dnsRR {\n")
	newRR(fd)
	fmt.Fprint(fd, "}\n\n")

	// nsrrtype
	fmt.Fprint(fd, "export function nsrrtype(input: number | string): string {\n")
	nsrrtype(fd)
	fmt.Fprint(fd, "}\n\n")

	// rdatatostr
	fmt.Fprint(fd, "export function rdatatostr(rr: dnsRR): string {\n")
	rdatatostr(fd)
	fmt.Fprint(fd, "}\n\n")

	// rdataFields
	fmt.Fprint(fd, "export function rdatafields(input: number | string): Array<string> {\n")
	rdataFields(fd)
	fmt.Fprint(fd, "}\n\n")
}
