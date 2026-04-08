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

// gen_instrumented_storage generates internal/app/instrumented_storage_generated.go,
// a metrics-instrumented wrapper for every method of storage.Storage.
package main

import (
	"bytes"
	"fmt"
	"go/format"
	"go/types"
	"log"
	"os"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

// entityMap maps each embedded interface type name to the Prometheus entity label.
var entityMap = map[string]string{
	"AuthUserStorage":          "authuser",
	"CheckPlanStorage":         "check_plan",
	"CheckerOptionsStorage":    "check_config",
	"CheckEvaluationStorage":   "check_evaluation",
	"ExecutionStorage":         "execution",
	"ObservationCacheStorage":  "observation_cache",
	"ObservationSnapshotStorage": "observation_snapshot",
	"SchedulerStateStorage":    "scheduler_state",
	"DomainStorage":            "domain",
	"DomainLogStorage":         "domain_log",
	"InsightStorage":           "insight",
	"ProviderStorage":          "provider",
	"SessionStorage":           "session",
	"UserStorage":              "user",
	"ZoneStorage":              "zone",
}

// operationOverrides maps method names that don't follow the prefix convention.
var operationOverrides = map[string]string{
	"AuthUserExists":    "get",
	"InsightsRun":       "run",
	"LastInsightsRun":   "get",
	"CreateOrUpdateUser": "update",
}

// skipMethods lists methods that should be passed through without instrumentation.
var skipMethods = map[string]bool{
	"SchemaVersion": true,
	"MigrateSchema": true,
	"Close":         true,
}

func main() {
	cfg := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedImports | packages.NeedDeps,
	}
	pkgs, err := packages.Load(cfg, "git.happydns.org/happyDomain/internal/storage")
	if err != nil {
		log.Fatalf("loading storage package: %v", err)
	}
	if len(pkgs) == 0 {
		log.Fatal("no packages loaded")
	}
	if len(pkgs[0].Errors) > 0 {
		for _, e := range pkgs[0].Errors {
			log.Println(e)
		}
		log.Fatal("package has errors")
	}

	storageObj := pkgs[0].Types.Scope().Lookup("Storage")
	if storageObj == nil {
		log.Fatal("Storage type not found")
	}
	storageIface := storageObj.Type().Underlying().(*types.Interface)

	// Build method → entity mapping by walking embedded interfaces.
	methodEntity := map[string]string{}
	for i := 0; i < storageIface.NumEmbeddeds(); i++ {
		embedded := storageIface.EmbeddedType(i)
		named, ok := embedded.(*types.Named)
		if !ok {
			continue
		}
		ifaceName := named.Obj().Name()
		entity, ok := entityMap[ifaceName]
		if !ok {
			log.Fatalf("unknown embedded interface %q — add it to entityMap", ifaceName)
		}
		iface := named.Underlying().(*types.Interface)
		for j := 0; j < iface.NumMethods(); j++ {
			methodEntity[iface.Method(j).Name()] = entity
		}
	}

	// Track imports needed by the generated code.
	imports := map[string]string{} // path → alias (empty = no alias)

	// qualifier returns the package qualifier for types.TypeString. It also
	// records each referenced package so we can emit the right imports.
	qualifier := func(pkg *types.Package) string {
		path := pkg.Path()
		name := pkg.Name()
		switch path {
		case "git.happydns.org/happyDomain/model":
			imports[path] = "happydns"
			return "happydns"
		default:
			imports[path] = ""
			return name
		}
	}

	type methodInfo struct {
		Name      string
		Entity    string
		Operation string
		Skip      bool // passthrough without observe
		Sig       *types.Signature
	}

	var methods []methodInfo
	for i := 0; i < storageIface.NumMethods(); i++ {
		m := storageIface.Method(i)
		name := m.Name()
		sig := m.Type().(*types.Signature)

		if skipMethods[name] {
			methods = append(methods, methodInfo{Name: name, Skip: true, Sig: sig})
			continue
		}

		entity, ok := methodEntity[name]
		if !ok {
			log.Fatalf("method %q has no entity mapping (not in any embedded interface?)", name)
		}

		op := operationOverrides[name]
		if op == "" {
			op = inferOperation(name)
		}

		methods = append(methods, methodInfo{
			Name:      name,
			Entity:    entity,
			Operation: op,
			Sig:       sig,
		})
	}

	// Pre-resolve all type strings so the qualifier captures all needed imports.
	type renderedMethod struct {
		info     methodInfo
		params   string // "p1 T1, p2 T2"
		results  string // "(r1 R1, r2 R2)"
		callArgs string // "p1, p2"
		retNames []string
		hasErr   bool
	}
	var rendered []renderedMethod
	for _, m := range methods {
		rm := renderedMethod{info: m}
		rm.params = renderParams(m.Sig.Params(), qualifier)
		rm.results = renderResults(m.Sig.Results(), qualifier, m.Skip)
		rm.callArgs = renderCallArgs(m.Sig.Params())
		rm.retNames, rm.hasErr = resultNames(m.Sig.Results(), m.Skip)
		rendered = append(rendered, rm)
	}

	// Generate code. Imports are written after the qualifier has been called
	// for every method signature, so the imports map is fully populated.
	var buf bytes.Buffer
	buf.WriteString(`// Code generated by go run tools/gen_instrumented_storage.go; DO NOT EDIT.

package app

import (
	"time"

	"git.happydns.org/happyDomain/internal/metrics"
	"git.happydns.org/happyDomain/internal/storage"
`)
	extraImports := map[string]string{}
	for path, alias := range imports {
		switch path {
		case "time", "git.happydns.org/happyDomain/internal/metrics", "git.happydns.org/happyDomain/internal/storage":
			continue
		default:
			extraImports[path] = alias
		}
	}
	paths := make([]string, 0, len(extraImports))
	for p := range extraImports {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		alias := extraImports[p]
		if alias != "" {
			fmt.Fprintf(&buf, "\t%s %q\n", alias, p)
		} else {
			fmt.Fprintf(&buf, "\t%q\n", p)
		}
	}
	buf.WriteString(")\n\n")

	buf.WriteString(`// instrumentedStorage wraps a storage.Storage to record Prometheus metrics for
// every operation.
type instrumentedStorage struct {
	inner storage.Storage
}

// newInstrumentedStorage wraps the given Storage with metrics instrumentation.
func newInstrumentedStorage(s storage.Storage) storage.Storage {
	return &instrumentedStorage{inner: s}
}

// observe starts a timer and returns a closure that, when called with a
// pointer to the named return error, records the operation outcome. Use as:
//
//	defer observe("get", "user")(&err)
//
// The closure reads *err at defer-execution time, so it captures the final
// value of the named return.
func observe(operation, entity string) func(err *error) {
	start := time.Now()
	return func(err *error) {
		status := "success"
		if *err != nil {
			status = "error"
		}
		metrics.StorageOperationsTotal.WithLabelValues(operation, entity, status).Inc()
		metrics.StorageOperationDuration.WithLabelValues(operation, entity).Observe(time.Since(start).Seconds())
	}
}

`)

	// Write method implementations.
	for _, rm := range rendered {
		m := rm.info
		if m.Skip {
			// Passthrough: one-liner.
			retType := types.TypeString(m.Sig.Results().At(0).Type(), qualifier)
			if m.Sig.Results().Len() == 0 {
				fmt.Fprintf(&buf, "func (s *instrumentedStorage) %s(%s) { s.inner.%s(%s) }\n\n",
					m.Name, rm.params, m.Name, rm.callArgs)
			} else if m.Sig.Results().Len() == 1 {
				fmt.Fprintf(&buf, "func (s *instrumentedStorage) %s(%s) %s { return s.inner.%s(%s) }\n\n",
					m.Name, rm.params, retType, m.Name, rm.callArgs)
			} else {
				fmt.Fprintf(&buf, "func (s *instrumentedStorage) %s(%s) %s { return s.inner.%s(%s) }\n\n",
					m.Name, rm.params, rm.results, m.Name, rm.callArgs)
			}
			continue
		}

		fmt.Fprintf(&buf, "func (s *instrumentedStorage) %s(%s) %s {\n",
			m.Name, rm.params, rm.results)
		fmt.Fprintf(&buf, "\tdefer observe(%q, %q)(&err)\n", m.Operation, m.Entity)
		fmt.Fprintf(&buf, "\treturn s.inner.%s(%s)\n", m.Name, rm.callArgs)
		buf.WriteString("}\n\n")
	}

	// Format the generated code.
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// Write unformatted for debugging.
		os.WriteFile("internal/app/instrumented_storage_generated.go", buf.Bytes(), 0644)
		log.Fatalf("gofmt failed: %v", err)
	}

	if err := os.WriteFile("internal/app/instrumented_storage_generated.go", formatted, 0644); err != nil {
		log.Fatalf("writing output: %v", err)
	}
	log.Println("wrote internal/app/instrumented_storage_generated.go")
}

// inferOperation derives the Prometheus operation label from a method name
// using prefix matching.
func inferOperation(name string) string {
	prefixes := []struct {
		prefix string
		op     string
	}{
		{"ListAll", "list"},
		{"List", "list"},
		{"Get", "get"},
		{"Count", "count"},
		{"Create", "create"},
		{"Update", "update"},
		{"Delete", "delete"},
		{"Clear", "delete"},
		{"Set", "set"},
		{"Put", "put"},
		{"Tidy", "tidy"},
	}
	for _, p := range prefixes {
		if strings.HasPrefix(name, p.prefix) {
			return p.op
		}
	}
	log.Fatalf("cannot infer operation for method %q — add it to operationOverrides", name)
	return ""
}

// renderParams renders the parameter list of a signature as "(name Type, ...)" for use in
// a method declaration.
func renderParams(params *types.Tuple, qual types.Qualifier) string {
	if params.Len() == 0 {
		return ""
	}
	var parts []string
	for i := 0; i < params.Len(); i++ {
		p := params.At(i)
		parts = append(parts, fmt.Sprintf("%s %s", p.Name(), types.TypeString(p.Type(), qual)))
	}
	return strings.Join(parts, ", ")
}

// renderResults renders the result list with named returns. For instrumented
// methods, the last result (error) is always named "err". Other results get
// placeholder names (ret, ret2, ...) to enable `defer observe(...)(&err)`.
func renderResults(results *types.Tuple, qual types.Qualifier, skip bool) string {
	if results.Len() == 0 {
		return ""
	}
	if skip {
		// For skipped methods, use the raw type list.
		if results.Len() == 1 {
			return types.TypeString(results.At(0).Type(), qual)
		}
		var parts []string
		for i := 0; i < results.Len(); i++ {
			parts = append(parts, types.TypeString(results.At(i).Type(), qual))
		}
		return "(" + strings.Join(parts, ", ") + ")"
	}

	names, _ := resultNames(results, skip)
	var parts []string
	for i := 0; i < results.Len(); i++ {
		parts = append(parts, fmt.Sprintf("%s %s", names[i], types.TypeString(results.At(i).Type(), qual)))
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

// resultNames returns synthetic names for each result variable. The last
// error-typed result is always "err"; others get "ret", "ret2", etc.
func resultNames(results *types.Tuple, skip bool) ([]string, bool) {
	names := make([]string, results.Len())
	hasErr := false
	retIdx := 0
	for i := 0; i < results.Len(); i++ {
		r := results.At(i)
		if i == results.Len()-1 && r.Type().String() == "error" {
			names[i] = "err"
			hasErr = true
		} else {
			// Use the original name if present, otherwise generate one.
			if r.Name() != "" && r.Name() != "_" {
				names[i] = r.Name()
			} else if retIdx == 0 {
				names[i] = "ret"
				retIdx++
			} else {
				names[i] = fmt.Sprintf("ret%d", retIdx+1)
				retIdx++
			}
		}
	}
	return names, hasErr
}

// renderCallArgs renders just the argument names for forwarding to s.inner.
func renderCallArgs(params *types.Tuple) string {
	if params.Len() == 0 {
		return ""
	}
	var parts []string
	for i := 0; i < params.Len(); i++ {
		parts = append(parts, params.At(i).Name())
	}
	return strings.Join(parts, ", ")
}
