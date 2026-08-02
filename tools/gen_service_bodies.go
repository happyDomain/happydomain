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

//go:build ignore
// +build ignore

package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"reflect"
	"slices"
	"sort"
	"strings"
	"unicode"

	"github.com/miekg/dns"

	intsvc "git.happydns.org/happyDomain/internal/serviceanalyzer"
	_ "git.happydns.org/happyDomain/services"
	_ "git.happydns.org/happyDomain/services/abstract"
	_ "git.happydns.org/happyDomain/services/providers/google"
)

// bodyInterfaceName turns a service type ("svcs.DMARCReport") into the name of
// the TypeScript interface describing its body ("SvcsDMARCReportBody"). The
// family is kept as a prefix so two services of different families sharing a
// name cannot collide.
func bodyInterfaceName(svctype string) string {
	var sb strings.Builder
	for _, part := range strings.FieldsFunc(svctype, func(r rune) bool { return r == '.' || r == '_' || r == '-' }) {
		sb.WriteRune(unicode.ToUpper(rune(part[0])))
		sb.WriteString(part[1:])
	}
	sb.WriteString("Body")
	return sb.String()
}

// rrTypeName returns the name of the DNS record type t describes, if it
// describes one. Both the miekg/dns records and the abstractions happyDomain
// defines over them (happydns.TXT) marshal as the record they are named after,
// so they map to the same dnsType* interface.
func rrTypeName(t reflect.Type) (string, bool) {
	name := t.Name()
	if name == "" {
		return "", false
	}

	if _, ok := dns.StringToType[name]; !ok {
		return "", false
	}

	pkg := t.PkgPath()
	if pkg != "github.com/miekg/dns" && pkg != "git.happydns.org/happyDomain/model" {
		return "", false
	}

	return "dnsType" + strings.Replace(name, "-", "_", -1), true
}

// jsonName returns the key the field takes in JSON, and whether it may be
// missing. It mirrors what encoding/json does with the tag.
func jsonName(f reflect.StructField) (name string, omitempty bool, skip bool) {
	tag := f.Tag.Get("json")
	if tag == "-" {
		return "", false, true
	}

	parts := strings.Split(tag, ",")
	name = parts[0]
	if name == "" {
		name = f.Name
	}

	return name, slices.Contains(parts[1:], "omitempty"), false
}

// toTSType renders the TypeScript type a Go type marshals to. Record types stop
// the recursion: they already have an interface of their own in dns_rr.ts.
func toTSType(t reflect.Type, indent int, imports map[string]bool) string {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if name, ok := rrTypeName(t); ok {
		imports[name] = true
		return name
	}

	switch t.Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Interface:
		// The only interface a body holds is happydns.Record, ie. any record.
		imports["dnsRR"] = true
		return "dnsRR"
	case reflect.Slice, reflect.Array:
		if t.Elem().Kind() == reflect.Uint8 && t.Name() == "IP" {
			return "string"
		}
		return "Array<" + toTSType(t.Elem(), indent, imports) + ">"
	case reflect.Map:
		return "Record<string, " + toTSType(t.Elem(), indent, imports) + ">"
	case reflect.Struct:
		return structToTSType(t, indent, imports)
	default:
		return "unknown"
	}
}

func structToTSType(t reflect.Type, indent int, imports map[string]bool) string {
	pad := strings.Repeat("    ", indent+1)

	var sb strings.Builder
	sb.WriteString("{\n")
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}

		name, omitempty, skip := jsonName(f)
		if skip {
			continue
		}

		opt := ""
		if omitempty {
			opt = "?"
		}

		fmt.Fprintf(&sb, "%s%s%s: %s;\n", pad, quoteKeyIfNeeded(name), opt, toTSType(f.Type, indent+1, imports))
	}
	sb.WriteString(strings.Repeat("    ", indent))
	sb.WriteString("}")

	return sb.String()
}

func quoteKeyIfNeeded(name string) string {
	for i, r := range name {
		if r == '_' || unicode.IsLetter(r) || (i > 0 && unicode.IsDigit(r)) {
			continue
		}
		return fmt.Sprintf("%q", name)
	}
	return name
}

func writeBodies(fd io.Writer) int {
	services := *intsvc.ListServices()

	svctypes := make([]string, 0, len(services))
	for svctype := range services {
		svctypes = append(svctypes, svctype)
	}
	sort.Strings(svctypes)

	imports := map[string]bool{}
	var bodies strings.Builder

	for _, svctype := range svctypes {
		t := reflect.TypeOf(services[svctype].Creator())
		for t.Kind() == reflect.Ptr {
			t = t.Elem()
		}

		fmt.Fprintf(&bodies, "export interface %s ", bodyInterfaceName(svctype))
		if t.Kind() == reflect.Struct {
			bodies.WriteString(structToTSType(t, 0, imports))
		} else {
			// A body that is not a struct marshals to a bare value, which
			// has no fields to describe.
			bodies.WriteString("{}")
		}
		bodies.WriteString("\n\n")
	}

	fmt.Fprint(fd, "// This file is generated by go generate\n\n")

	if len(imports) > 0 {
		names := make([]string, 0, len(imports))
		for name := range imports {
			names = append(names, name)
		}
		sort.Strings(names)
		fmt.Fprintf(fd, "import type { %s } from \"$lib/dns_rr\";\n\n", strings.Join(names, ", "))
	}

	fmt.Fprint(fd, "// The body each service carries, as the API serializes it.\n")
	fmt.Fprint(fd, bodies.String())

	// Lookup by service type, for the places holding a service type as a string.
	fmt.Fprint(fd, "export interface ServiceBodies {\n")
	for _, svctype := range svctypes {
		fmt.Fprintf(fd, "    %q: %s;\n", svctype, bodyInterfaceName(svctype))
	}
	fmt.Fprint(fd, "}\n\n")

	fmt.Fprint(fd, "// The body of any service, for the places holding one before knowing\n")
	fmt.Fprint(fd, "// which service it belongs to.\n")
	fmt.Fprint(fd, "export type ServiceBody = ServiceBodies[keyof ServiceBodies];\n")

	return len(svctypes)
}

func main() {
	output := flag.String("o", "", "output file path")
	flag.Parse()

	if *output == "" {
		fmt.Fprintf(os.Stderr, "Error: output file path is required\n")
		fmt.Fprintf(os.Stderr, "Usage: %s -o <output-file>\n", os.Args[0])
		os.Exit(1)
	}

	fd, err := os.Create(*output)
	if err != nil {
		panic(err)
	}
	defer fd.Close()

	count := writeBodies(fd)

	fmt.Printf("Generated %s with %d Service bodies\n", *output, count)
}
