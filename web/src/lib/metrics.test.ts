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

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";

vi.mock("$lib/stores/adminsession", () => ({
    getAdminToken: vi.fn(),
}));

import { getAdminToken } from "$lib/stores/adminsession";
import {
    parsePrometheusText,
    fetchMetrics,
    singleValue,
    sumValues,
    firstLabel,
    type Metrics,
} from "./metrics";

describe("parsePrometheusText", () => {
    it("returns an empty object for empty input", () => {
        expect(parsePrometheusText("")).toEqual({});
    });

    it("ignores blank lines and whitespace-only lines", () => {
        expect(parsePrometheusText("\n   \n\t\n")).toEqual({});
    });

    it("ignores HELP and TYPE comment lines", () => {
        const text = [
            "# HELP http_requests_total Total requests",
            "# TYPE http_requests_total counter",
            "http_requests_total 42",
        ].join("\n");
        expect(parsePrometheusText(text)).toEqual({
            http_requests_total: [{ name: "http_requests_total", labels: {}, value: 42 }],
        });
    });

    it("parses a simple metric without labels", () => {
        const out = parsePrometheusText("go_goroutines 12");
        expect(out).toEqual({
            go_goroutines: [{ name: "go_goroutines", labels: {}, value: 12 }],
        });
    });

    it("parses a metric with a single label", () => {
        const out = parsePrometheusText('build_info{version="1.2.3"} 1');
        expect(out).toEqual({
            build_info: [{ name: "build_info", labels: { version: "1.2.3" }, value: 1 }],
        });
    });

    it("parses a metric with multiple labels", () => {
        const out = parsePrometheusText('http_requests_total{method="GET",status="200"} 5');
        expect(out).toEqual({
            http_requests_total: [
                { name: "http_requests_total", labels: { method: "GET", status: "200" }, value: 5 },
            ],
        });
    });

    it("aggregates multiple samples of the same metric name", () => {
        const text = [
            'http_requests_total{method="GET"} 5',
            'http_requests_total{method="POST"} 3',
        ].join("\n");
        const out = parsePrometheusText(text);
        expect(out.http_requests_total).toHaveLength(2);
        expect(out.http_requests_total).toEqual([
            { name: "http_requests_total", labels: { method: "GET" }, value: 5 },
            { name: "http_requests_total", labels: { method: "POST" }, value: 3 },
        ]);
    });

    it("keeps distinct metric names separate", () => {
        const text = ["metric_a 1", "metric_b 2"].join("\n");
        const out = parsePrometheusText(text);
        expect(Object.keys(out).sort()).toEqual(["metric_a", "metric_b"]);
    });

    it("parses negative and floating point values", () => {
        const text = ["temp_celsius -3.5", "ratio 0.001"].join("\n");
        const out = parsePrometheusText(text);
        expect(singleValue(out, "temp_celsius")).toBe(-3.5);
        expect(singleValue(out, "ratio")).toBe(0.001);
    });

    it("parses scientific notation values", () => {
        const out = parsePrometheusText("big_number 1.5e3");
        expect(singleValue(out, "big_number")).toBe(1500);
    });

    it("skips non-finite values such as +Inf, Infinity and NaN", () => {
        // Number.isFinite() rejects Infinity and NaN, so all of these are dropped.
        const text = ["a_metric +Inf", "b_metric Infinity", "c_metric NaN"].join("\n");
        const out = parsePrometheusText(text);
        expect(out).toEqual({});
    });

    it("skips lines with a non-numeric value", () => {
        const out = parsePrometheusText("broken_metric not_a_number");
        expect(out).toEqual({});
    });

    it("skips lines with no whitespace separator and no labels", () => {
        const out = parsePrometheusText("just_a_name_with_no_value");
        expect(out).toEqual({});
    });

    it("trims leading/trailing whitespace on each line", () => {
        const out = parsePrometheusText("   spaced_metric   7   ");
        expect(singleValue(out, "spaced_metric")).toBe(7);
    });

    it("handles a timestamp suffix after the value", () => {
        const out = parsePrometheusText("metric_with_ts 100 1633024800000");
        expect(singleValue(out, "metric_with_ts")).toBe(100);
    });

    it("unescapes escaped quotes and backslashes in label values", () => {
        const out = parsePrometheusText('quoted_metric{msg="he said \\"hi\\""} 1');
        expect(firstLabel(out, "quoted_metric", "msg")).toBe('he said "hi"');
    });

    it("unescapes backslashes in label values", () => {
        const out = parsePrometheusText('path_metric{path="C:\\\\Users\\\\bob"} 1');
        expect(firstLabel(out, "path_metric", "path")).toBe("C:\\Users\\bob");
    });

    it("handles empty label braces", () => {
        const out = parsePrometheusText("empty_labels{} 9");
        expect(out.empty_labels).toEqual([{ name: "empty_labels", labels: {}, value: 9 }]);
    });

    it("handles an empty label value", () => {
        const out = parsePrometheusText('empty_val{key=""} 1');
        expect(firstLabel(out, "empty_val", "key")).toBe("");
    });

    it("parses a full multi-line Prometheus document", () => {
        const text = [
            "# HELP go_goroutines Number of goroutines",
            "# TYPE go_goroutines gauge",
            "go_goroutines 15",
            "",
            "# HELP http_requests_total Total HTTP requests",
            "# TYPE http_requests_total counter",
            'http_requests_total{method="GET",code="200"} 1027',
            'http_requests_total{method="POST",code="500"} 3',
            "",
            "# HELP build_info Build information",
            "# TYPE build_info gauge",
            'build_info{version="4.5.6",revision="abc123"} 1',
        ].join("\n");

        const out = parsePrometheusText(text);

        expect(singleValue(out, "go_goroutines")).toBe(15);
        expect(sumValues(out, "http_requests_total")).toBe(1030);
        expect(firstLabel(out, "build_info", "version")).toBe("4.5.6");
        expect(firstLabel(out, "build_info", "revision")).toBe("abc123");
    });
});

describe("singleValue", () => {
    it("returns the value of a metric with one sample", () => {
        const metrics: Metrics = { foo: [{ name: "foo", labels: {}, value: 42 }] };
        expect(singleValue(metrics, "foo")).toBe(42);
    });

    it("returns the first sample's value when there are several", () => {
        const metrics: Metrics = {
            foo: [
                { name: "foo", labels: { a: "1" }, value: 1 },
                { name: "foo", labels: { a: "2" }, value: 2 },
            ],
        };
        expect(singleValue(metrics, "foo")).toBe(1);
    });

    it("returns undefined when the metric is absent", () => {
        expect(singleValue({}, "missing")).toBeUndefined();
    });

    it("returns undefined when the metric has an empty sample array", () => {
        expect(singleValue({ foo: [] }, "foo")).toBeUndefined();
    });
});

describe("sumValues", () => {
    it("sums all samples of a metric", () => {
        const metrics: Metrics = {
            foo: [
                { name: "foo", labels: { a: "1" }, value: 1 },
                { name: "foo", labels: { a: "2" }, value: 2 },
                { name: "foo", labels: { a: "3" }, value: 3 },
            ],
        };
        expect(sumValues(metrics, "foo")).toBe(6);
    });

    it("returns the single value for a metric with one sample", () => {
        const metrics: Metrics = { foo: [{ name: "foo", labels: {}, value: 9 }] };
        expect(sumValues(metrics, "foo")).toBe(9);
    });

    it("returns undefined when the metric is absent", () => {
        expect(sumValues({}, "missing")).toBeUndefined();
    });

    it("returns undefined when the metric has an empty sample array", () => {
        expect(sumValues({ foo: [] }, "foo")).toBeUndefined();
    });

    it("handles negative values correctly", () => {
        const metrics: Metrics = {
            foo: [
                { name: "foo", labels: {}, value: 5 },
                { name: "foo", labels: {}, value: -2 },
            ],
        };
        expect(sumValues(metrics, "foo")).toBe(3);
    });
});

describe("firstLabel", () => {
    it("returns the label value from the first sample", () => {
        const metrics: Metrics = {
            build_info: [{ name: "build_info", labels: { version: "1.0.0" }, value: 1 }],
        };
        expect(firstLabel(metrics, "build_info", "version")).toBe("1.0.0");
    });

    it("returns undefined when the label is missing on the first sample", () => {
        const metrics: Metrics = {
            build_info: [{ name: "build_info", labels: {}, value: 1 }],
        };
        expect(firstLabel(metrics, "build_info", "version")).toBeUndefined();
    });

    it("returns undefined when the metric is absent", () => {
        expect(firstLabel({}, "missing", "version")).toBeUndefined();
    });

    it("returns undefined when the metric has an empty sample array", () => {
        expect(firstLabel({ foo: [] }, "foo", "bar")).toBeUndefined();
    });

    it("only looks at the first sample, ignoring labels on later ones", () => {
        const metrics: Metrics = {
            foo: [
                { name: "foo", labels: {}, value: 1 },
                { name: "foo", labels: { bar: "baz" }, value: 2 },
            ],
        };
        expect(firstLabel(metrics, "foo", "bar")).toBeUndefined();
    });
});

describe("fetchMetrics", () => {
    const mockedGetAdminToken = vi.mocked(getAdminToken);

    beforeEach(() => {
        mockedGetAdminToken.mockReturnValue(null);
        vi.stubGlobal("fetch", vi.fn());
    });

    afterEach(() => {
        vi.unstubAllGlobals();
        vi.restoreAllMocks();
    });

    it("fetches /metrics without an Authorization header when there is no admin token", async () => {
        vi.mocked(fetch).mockResolvedValue(
            new Response("go_goroutines 5", { status: 200, statusText: "OK" }),
        );

        const result = await fetchMetrics();

        expect(fetch).toHaveBeenCalledWith("/metrics", {
            headers: { Accept: "text/plain" },
        });
        expect(singleValue(result, "go_goroutines")).toBe(5);
    });

    it("includes a Bearer Authorization header when an admin token is present", async () => {
        mockedGetAdminToken.mockReturnValue("secret-token");
        vi.mocked(fetch).mockResolvedValue(new Response("foo 1", { status: 200 }));

        await fetchMetrics();

        expect(fetch).toHaveBeenCalledWith("/metrics", {
            headers: { Accept: "text/plain", Authorization: "Bearer secret-token" },
        });
    });

    it("throws an error with status and statusText when the response is not ok", async () => {
        vi.mocked(fetch).mockResolvedValue(
            new Response("", { status: 500, statusText: "Internal Server Error" }),
        );

        await expect(fetchMetrics()).rejects.toThrow(
            "Failed to fetch /metrics: 500 Internal Server Error",
        );
    });

    it("throws on a 401 unauthorized response", async () => {
        vi.mocked(fetch).mockResolvedValue(
            new Response("", { status: 401, statusText: "Unauthorized" }),
        );

        await expect(fetchMetrics()).rejects.toThrow("Failed to fetch /metrics: 401 Unauthorized");
    });

    it("parses the fetched text body into Metrics", async () => {
        const body = [
            'http_requests_total{method="GET"} 10',
            'http_requests_total{method="POST"} 5',
        ].join("\n");
        vi.mocked(fetch).mockResolvedValue(new Response(body, { status: 200 }));

        const result = await fetchMetrics();

        expect(sumValues(result, "http_requests_total")).toBe(15);
    });
});
