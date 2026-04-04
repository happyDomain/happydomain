<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2022-2026 happyDomain
     Authors: Pierre-Olivier Mercier, et al.

     This program is offered under a commercial and under the AGPL license.
     For commercial licensing, contact us at <contact@happydomain.org>.

     For AGPL licensing:
     This program is free software: you can redistribute it and/or modify
     it under the terms of the GNU Affero General Public License as published by
     the Free Software Foundation, either version 3 of the License, or
     (at your option) any later version.

     This program is distributed in the hope that it will be useful,
     but WITHOUT ANY WARRANTY; without even the implied warranty of
     MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
     GNU Affero General Public License for more details.

     You should have received a copy of the GNU Affero General Public License
     along with this program.  If not, see <https://www.gnu.org/licenses/>.
-->

<script lang="ts">
    import { onDestroy } from "svelte";
    import {
        Chart,
        LineController,
        LineElement,
        PointElement,
        LinearScale,
        TimeScale,
        Legend,
        Tooltip,
        Filler,
    } from "chart.js";
    import "chartjs-adapter-date-fns";
    import type { CheckMetric } from "$lib/api/checkers";

    Chart.register(
        LineController,
        LineElement,
        PointElement,
        LinearScale,
        TimeScale,
        Legend,
        Tooltip,
        Filler,
    );

    interface Props {
        metrics: CheckMetric[];
    }

    let { metrics }: Props = $props();

    let canvas: HTMLCanvasElement;
    let chart: Chart | null = null;

    const COLORS = [
        "#0d6efd",
        "#dc3545",
        "#198754",
        "#ffc107",
        "#6610f2",
        "#0dcaf0",
        "#fd7e14",
        "#d63384",
    ];

    function buildChart() {
        if (chart) {
            chart.destroy();
            chart = null;
        }
        if (!canvas || !metrics?.length) return;

        // Group metrics into series by (name, labels)
        type SeriesKey = string;
        interface Series {
            name: string;
            unit: string;
            labels: Record<string, string>;
            points: { x: number; y: number }[];
        }

        const seriesMap = new Map<SeriesKey, Series>();
        const seriesOrder: SeriesKey[] = [];

        for (const m of metrics) {
            const labelStr = m.labels ? Object.entries(m.labels).sort().map(([k, v]) => `${k}=${v}`).join(",") : "";
            const key = `${m.name}{${labelStr}}`;
            if (!seriesMap.has(key)) {
                seriesMap.set(key, {
                    name: m.name,
                    unit: m.unit ?? "",
                    labels: m.labels ?? {},
                    points: [],
                });
                seriesOrder.push(key);
            }
            seriesMap.get(key)!.points.push({
                x: new Date(m.timestamp).getTime(),
                y: m.value,
            });
        }

        // Sort points by time within each series
        for (const series of seriesMap.values()) {
            series.points.sort((a, b) => a.x - b.x);
        }

        // Determine unique units for multi-axis support
        const units = [...new Set(Array.from(seriesMap.values()).map((s) => s.unit))];
        const hasRightAxis = units.length > 1;
        const rightUnit = hasRightAxis ? units[1] : null;

        const datasets = seriesOrder.map((key, i) => {
            const series = seriesMap.get(key)!;
            const labelParts = Object.entries(series.labels).map(([k, v]) => `${k}=${v}`);
            const displayLabel = labelParts.length > 0
                ? `${series.name} (${labelParts.join(", ")})`
                : series.name;

            return {
                label: displayLabel,
                data: series.points,
                borderColor: COLORS[i % COLORS.length],
                backgroundColor: COLORS[i % COLORS.length] + "20",
                borderWidth: 2,
                pointRadius: 3,
                pointHoverRadius: 5,
                tension: 0.3,
                yAxisID: hasRightAxis && series.unit === rightUnit ? "y1" : "y",
            };
        });

        const scales: Record<string, any> = {
            x: {
                type: "time" as const,
                time: { tooltipFormat: "PPpp" },
                title: { display: false },
            },
            y: {
                type: "linear" as const,
                position: "left" as const,
                title: { display: true, text: units[0] || "" },
                beginAtZero: true,
            },
        };

        if (hasRightAxis && rightUnit) {
            scales.y1 = {
                type: "linear" as const,
                position: "right" as const,
                title: { display: true, text: rightUnit },
                beginAtZero: true,
                grid: { drawOnChartArea: false },
            };
        }

        chart = new Chart(canvas, {
            type: "line",
            data: { datasets },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                interaction: { mode: "index", intersect: false },
                scales,
                plugins: {
                    legend: { position: "bottom" },
                    tooltip: { mode: "index", intersect: false },
                },
            },
        });
    }

    $effect(() => {
        if (metrics && canvas) {
            buildChart();
        }
    });

    onDestroy(() => {
        if (chart) {
            chart.destroy();
            chart = null;
        }
    });
</script>

<div class="chart-container" style="position: relative; height: 350px; width: 100%;">
    <canvas bind:this={canvas}></canvas>
</div>
