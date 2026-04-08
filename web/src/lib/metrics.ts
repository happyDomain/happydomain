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

export type MetricSample = {
    name: string;
    labels: Record<string, string>;
    value: number;
};

export type Metrics = Record<string, MetricSample[]>;

// Minimal Prometheus text format parser. Handles standard lines of the form
// `metric_name{label="value",...} number` and ignores HELP/TYPE/comment lines.
export function parsePrometheusText(text: string): Metrics {
    const out: Metrics = {};
    for (const rawLine of text.split("\n")) {
        const line = rawLine.trim();
        if (!line || line.startsWith("#")) continue;

        // Split name(+labels) from value (last whitespace-separated token before optional timestamp).
        const braceEnd = line.indexOf("}");
        let head: string;
        let rest: string;
        if (braceEnd >= 0) {
            head = line.slice(0, braceEnd + 1);
            rest = line.slice(braceEnd + 1).trim();
        } else {
            const sp = line.indexOf(" ");
            if (sp < 0) continue;
            head = line.slice(0, sp);
            rest = line.slice(sp + 1).trim();
        }

        const valueToken = rest.split(/\s+/)[0];
        const value = Number(valueToken);
        if (!Number.isFinite(value)) continue;

        let name = head;
        const labels: Record<string, string> = {};
        const lb = head.indexOf("{");
        if (lb >= 0) {
            name = head.slice(0, lb);
            const labelStr = head.slice(lb + 1, head.lastIndexOf("}"));
            // Naive label parser; sufficient for values without escaped quotes/commas
            const re = /([a-zA-Z_][a-zA-Z0-9_]*)="((?:[^"\\]|\\.)*)"/g;
            let m: RegExpExecArray | null;
            while ((m = re.exec(labelStr)) !== null) {
                labels[m[1]] = m[2].replace(/\\"/g, '"').replace(/\\\\/g, "\\");
            }
        }

        (out[name] ||= []).push({ name, labels, value });
    }
    return out;
}

export async function fetchMetrics(): Promise<Metrics> {
    const res = await fetch("/metrics", { headers: { Accept: "text/plain" } });
    if (!res.ok) {
        throw new Error(`Failed to fetch /metrics: ${res.status} ${res.statusText}`);
    }
    return parsePrometheusText(await res.text());
}

// Returns the single value of a metric, or undefined if absent.
export function singleValue(metrics: Metrics, name: string): number | undefined {
    const samples = metrics[name];
    if (!samples || samples.length === 0) return undefined;
    return samples[0].value;
}

// Sums all samples of a metric (useful for *_total counters with labels).
export function sumValues(metrics: Metrics, name: string): number | undefined {
    const samples = metrics[name];
    if (!samples || samples.length === 0) return undefined;
    return samples.reduce((acc, s) => acc + s.value, 0);
}

// Returns the first label value found for a metric (e.g. build version).
export function firstLabel(metrics: Metrics, name: string, label: string): string | undefined {
    const samples = metrics[name];
    if (!samples || samples.length === 0) return undefined;
    return samples[0].labels[label];
}
