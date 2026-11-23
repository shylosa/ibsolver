#!/usr/bin/env python3
"""Generate a self-contained viewer.html from series.csv and manifest.json.

Does not modify source CSV/JSON. Open the resulting viewer.html via file://.
"""

from __future__ import annotations

import csv
import json
from collections import defaultdict
from pathlib import Path

ROOT = Path(__file__).resolve().parent
SERIES_CSV = ROOT / "series.csv"
MANIFEST_JSON = ROOT / "manifest.json"
OUT_HTML = ROOT / "viewer.html"

SERIES_COLORS = [
    "#5ba3d9",
    "#e7a35a",
    "#7bc67e",
    "#d97b9a",
    "#c4b05a",
    "#9b8fd9",
    "#6ec4c0",
    "#d98b6b",
]


def load_points() -> dict[int, dict[int, list[dict]]]:
    grouped: dict[int, dict[int, list[dict]]] = defaultdict(lambda: defaultdict(list))
    with SERIES_CSV.open(encoding="utf-8", newline="") as f:
        for row in csv.DictReader(f):
            ci = int(row["chart_index"])
            si = int(row["series_index"])
            grouped[ci][si].append(
                {
                    "i": int(row["point_index"]),
                    "x": row["x"],  # original CSV text
                    "y": row["y"],  # original CSV text
                    "xa": row["x_address"],
                    "ya": row["y_address"],
                    "workbook": row["workbook"],
                    "series_name": row["series"],
                    "values_ref": row["values_ref"],
                    "categories_ref": row["categories_ref"],
                }
            )
    for ci in grouped:
        for si in grouped[ci]:
            grouped[ci][si].sort(key=lambda p: p["i"])
    return grouped


def build_payload() -> dict:
    with MANIFEST_JSON.open(encoding="utf-8") as f:
        manifest = json.load(f)

    points = load_points()
    charts_out = []
    total_series = 0
    total_points = 0
    mismatches = []

    for wb in manifest["workbooks"]:
        for ch in wb.get("charts") or []:
            ci = int(ch["chart_index"])
            series_out = []
            for si, s in enumerate(ch["series"]):
                pts = points.get(ci, {}).get(si, [])
                man_n = s.get("point_count")
                if man_n is not None and man_n != len(pts):
                    mismatches.append(
                        {
                            "chart_index": ci,
                            "series_index": si,
                            "csv_points": len(pts),
                            "manifest_point_count": man_n,
                        }
                    )
                name = s.get("name")
                if not name and pts:
                    name = pts[0]["series_name"]
                if not name:
                    name = s.get("values_ref") or f"series_{si}"

                series_out.append(
                    {
                        "series_index": si,
                        "name": name,
                        "values_ref": s.get("values_ref")
                        or (pts[0]["values_ref"] if pts else ""),
                        "categories_ref": s.get("categories_ref")
                        or (pts[0]["categories_ref"] if pts else ""),
                        "title_ref": s.get("title_ref"),
                        "point_count": len(pts),
                        "color": SERIES_COLORS[si % len(SERIES_COLORS)],
                        "points": [
                            {
                                "i": p["i"],
                                "x": p["x"],
                                "y": p["y"],
                                "xa": p["xa"],
                                "ya": p["ya"],
                            }
                            for p in pts
                        ],
                    }
                )
                total_series += 1
                total_points += len(pts)

            display_name = ch.get("title") or ch.get("name")
            if not display_name:
                display_name = f"Діаграма №{ci} — {ch.get('host_sheet') or '?'}"

            charts_out.append(
                {
                    "chart_index": ci,
                    "workbook": wb["file_name"],
                    "host_sheet": ch.get("host_sheet") or "",
                    "name": ch.get("name") or display_name,
                    "title": ch.get("title"),
                    "display_name": display_name,
                    "x_axis_title": ch.get("x_axis_title") or "X",
                    "y_axis_title": ch.get("y_axis_title") or "Y",
                    "chart_types": ch.get("chart_types") or [],
                    "series": series_out,
                }
            )

    charts_out.sort(key=lambda c: c["chart_index"])
    return {
        "meta": {
            "source_series": SERIES_CSV.name,
            "source_manifest": MANIFEST_JSON.name,
            "chart_count": len(charts_out),
            "series_count": total_series,
            "point_count": total_points,
            "mismatches": mismatches,
        },
        "charts": charts_out,
    }


HTML_TEMPLATE = r"""<!DOCTYPE html>
<html lang="uk">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>IBSolver — історичні діаграми Excel</title>
<style>
:root {
  color-scheme: dark;
  --bg: #131922;
  --panel: #19222e;
  --panel-2: #18212d;
  --border: #2d3949;
  --border-strong: #354254;
  --text: #dce5ef;
  --muted: #95a7ba;
  --accent: #6597bc;
  --accent-btn: #245c7d;
  --hover: #2c3e53;
  --grid: #263244;
  --axis: #6f8298;
}
* { box-sizing: border-box; }
html, body {
  margin: 0; height: 100%;
  font: 13px/1.4 'Segoe UI', Tahoma, sans-serif;
  color: var(--text); background: var(--bg);
}
body { display: flex; flex-direction: column; min-height: 100vh; }
header {
  flex: 0 0 auto;
  display: flex; align-items: center; gap: 16px;
  min-height: 48px; padding: 6px 18px;
  border-bottom: 1px solid var(--border); background: var(--panel);
}
header h1 { margin: 0; font-size: 15px; font-weight: 600; letter-spacing: 0.02em; }
header .meta { color: var(--muted); font-size: 12px; }
.layout {
  flex: 1 1 auto; min-height: 0;
  display: grid;
  grid-template-columns: 280px minmax(0, 1fr) 300px;
  gap: 0;
}
.sidebar, .info {
  background: var(--panel); overflow: auto;
}
.sidebar { border-right: 1px solid var(--border); padding: 10px 0; }
.info { border-left: 1px solid var(--border); padding: 14px 16px; }
.main {
  min-width: 0; min-height: 0;
  display: flex; flex-direction: column;
  background: var(--panel-2);
}
.chart-list { list-style: none; margin: 0; padding: 0; }
.chart-list button {
  width: 100%; text-align: left;
  border: 0; border-left: 3px solid transparent;
  background: transparent; color: var(--text);
  padding: 10px 14px 10px 12px; cursor: pointer;
  font: inherit; line-height: 1.35;
}
.chart-list button:hover { background: var(--hover); }
.chart-list button.active {
  background: #213042;
  border-left-color: var(--accent);
  color: #fff;
}
.chart-list .idx {
  display: inline-block; min-width: 1.6em;
  color: var(--muted); font-variant-numeric: tabular-nums;
}
.chart-list .sheet {
  display: block; margin-top: 2px;
  color: var(--muted); font-size: 11px;
}
.toolbar {
  display: flex; align-items: center; gap: 10px; flex-wrap: wrap;
  padding: 10px 14px 0; color: var(--muted); font-size: 12px;
}
.toolbar button {
  font: inherit; color: inherit;
  border: 1px solid #3c546e; border-radius: 4px;
  background: #213042; color: #bed5ea;
  padding: 5px 12px; cursor: pointer;
}
.toolbar button:hover { background: #2a3e55; border-color: #5d8ab5; color: #fff; }
.canvas-wrap {
  flex: 1 1 auto; min-height: 220px;
  position: relative; margin: 10px 14px 6px;
  border: 1px solid var(--border-strong); border-radius: 4px;
  background: #101820;
}
#chart {
  display: block; width: 100%; height: 100%;
  cursor: crosshair;
}
.legend {
  flex: 0 0 auto;
  display: flex; flex-wrap: wrap; gap: 8px 14px;
  padding: 8px 14px 12px; align-items: center;
}
.legend label {
  display: inline-flex; align-items: center; gap: 6px;
  cursor: pointer; user-select: none;
  padding: 3px 6px; border-radius: 3px;
}
.legend label:hover { background: #213042; }
.legend input { accent-color: var(--accent); }
.swatch {
  width: 12px; height: 12px; border-radius: 2px;
  border: 1px solid #0006; flex: 0 0 auto;
}
.info h2 {
  margin: 0 0 12px; font-size: 14px; font-weight: 600;
  word-break: break-word;
}
.info dl {
  margin: 0 0 16px;
  display: grid;
  grid-template-columns: 110px 1fr;
  gap: 6px 10px;
  font-size: 12px;
}
.info dt { color: var(--muted); }
.info dd { margin: 0; word-break: break-word; }
.info .section-title {
  margin: 0 0 8px; font-size: 11px; letter-spacing: 0.08em;
  text-transform: uppercase; color: var(--muted);
}
.hover-box {
  border: 1px solid var(--border-strong);
  border-radius: 4px;
  background: #151e28;
  padding: 10px 12px;
  min-height: 120px;
  font-size: 12px;
}
.hover-box .empty { color: var(--muted); }
.hover-box .row {
  display: grid; grid-template-columns: 88px 1fr;
  gap: 4px 8px; margin: 0 0 4px;
}
.hover-box .k { color: var(--muted); }
.hover-box .v { word-break: break-all; font-variant-numeric: tabular-nums; }
.hover-series {
  display: flex; align-items: center; gap: 8px;
  margin-bottom: 8px; font-weight: 600;
}
footer {
  flex: 0 0 auto;
  padding: 7px 18px;
  font-size: 11px; color: var(--muted);
  border-top: 1px solid #344254; background: var(--panel);
}
@media (max-width: 980px) {
  .layout { grid-template-columns: 220px minmax(0, 1fr); grid-template-rows: 1fr auto; }
  .info { grid-column: 1 / -1; border-left: 0; border-top: 1px solid var(--border); max-height: 240px; }
}
@media (max-width: 720px) {
  .layout { grid-template-columns: 1fr; }
  .sidebar { max-height: 180px; border-right: 0; border-bottom: 1px solid var(--border); }
}
</style>
</head>
<body>
<header>
  <h1>IBSolver — історичні діаграми Excel</h1>
  <div class="meta" id="header-meta"></div>
</header>
<div class="layout">
  <aside class="sidebar">
    <ul class="chart-list" id="chart-list"></ul>
  </aside>
  <main class="main">
    <div class="toolbar">
      <button type="button" id="btn-show-all">Показати все</button>
      <span id="chart-subtitle"></span>
    </div>
    <div class="canvas-wrap">
      <canvas id="chart"></canvas>
    </div>
    <div class="legend" id="legend"></div>
  </main>
  <aside class="info">
    <div class="section-title">Діаграма</div>
    <h2 id="info-title">—</h2>
    <dl id="info-dl"></dl>
    <div class="section-title">Курсор</div>
    <div class="hover-box" id="hover-box">
      <div class="empty">Наведіть курсор на точку графіка</div>
    </div>
  </aside>
</div>
<footer id="footer"></footer>
<script id="legacy-data" type="application/json">__DATA_JSON__</script>
<script>
(function () {
  'use strict';

  var DATA = JSON.parse(document.getElementById('legacy-data').textContent);
  var charts = DATA.charts;
  var state = {
    chartIndex: charts.length ? charts[0].chart_index : -1,
    visible: {},
    hover: null
  };

  var listEl = document.getElementById('chart-list');
  var legendEl = document.getElementById('legend');
  var canvas = document.getElementById('chart');
  var ctx = canvas.getContext('2d');
  var hoverBox = document.getElementById('hover-box');
  var infoTitle = document.getElementById('info-title');
  var infoDl = document.getElementById('info-dl');
  var subtitle = document.getElementById('chart-subtitle');

  document.getElementById('header-meta').textContent =
    DATA.meta.chart_count + ' діаграм · ' +
    DATA.meta.series_count + ' серій · ' +
    DATA.meta.point_count + ' точок';

  document.getElementById('footer').textContent =
    'Дані вбудовано з ' + DATA.meta.source_series + ' та ' +
    DATA.meta.source_manifest +
    '. Значення X/Y без змін. Масштаб лише для відображення.';

  function currentChart() {
    for (var i = 0; i < charts.length; i++) {
      if (charts[i].chart_index === state.chartIndex) return charts[i];
    }
    return null;
  }

  function ensureVisibility(chart) {
    var key = String(chart.chart_index);
    if (!state.visible[key]) {
      state.visible[key] = {};
      for (var i = 0; i < chart.series.length; i++) {
        state.visible[key][chart.series[i].series_index] = true;
      }
    }
  }

  function isVisible(chart, seriesIndex) {
    return !!state.visible[String(chart.chart_index)][seriesIndex];
  }

  function renderList() {
    listEl.innerHTML = '';
    for (var i = 0; i < charts.length; i++) {
      var ch = charts[i];
      var li = document.createElement('li');
      var btn = document.createElement('button');
      btn.type = 'button';
      btn.className = ch.chart_index === state.chartIndex ? 'active' : '';
      btn.innerHTML =
        '<span class="idx">' + ch.chart_index + '</span> ' +
        escapeHtml(ch.display_name) +
        '<span class="sheet">' + escapeHtml(ch.host_sheet) + '</span>';
      btn.addEventListener('click', (function (idx) {
        return function () { selectChart(idx); };
      })(ch.chart_index));
      li.appendChild(btn);
      listEl.appendChild(li);
    }
  }

  function escapeHtml(s) {
    return String(s == null ? '' : s)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;');
  }

  function selectChart(chartIndex) {
    state.chartIndex = chartIndex;
    state.hover = null;
    var chart = currentChart();
    if (!chart) return;
    ensureVisibility(chart);
    renderList();
    renderInfo(chart);
    renderLegend(chart);
    draw();
  }

  function renderInfo(chart) {
    var totalPts = 0;
    for (var i = 0; i < chart.series.length; i++) totalPts += chart.series[i].point_count;
    infoTitle.textContent = chart.display_name;
    subtitle.textContent = chart.workbook + ' / ' + chart.host_sheet;
    infoDl.innerHTML =
      row('Книга', chart.workbook) +
      row('Аркуш', chart.host_sheet) +
      row('Назва', chart.name) +
      row('Заголовок', chart.title || '—') +
      row('Вісь X', chart.x_axis_title) +
      row('Вісь Y', chart.y_axis_title) +
      row('Серій', String(chart.series.length)) +
      row('Точок', String(totalPts)) +
      row('Індекс', String(chart.chart_index));
  }

  function row(k, v) {
    return '<dt>' + escapeHtml(k) + '</dt><dd>' + escapeHtml(v) + '</dd>';
  }

  function renderLegend(chart) {
    legendEl.innerHTML = '';
    for (var i = 0; i < chart.series.length; i++) {
      var s = chart.series[i];
      var label = document.createElement('label');
      var cb = document.createElement('input');
      cb.type = 'checkbox';
      cb.checked = isVisible(chart, s.series_index);
      cb.addEventListener('change', (function (si) {
        return function (ev) {
          state.visible[String(chart.chart_index)][si] = ev.target.checked;
          state.hover = null;
          draw();
          renderHover(null);
        };
      })(s.series_index));
      var sw = document.createElement('span');
      sw.className = 'swatch';
      sw.style.background = s.color;
      var text = document.createElement('span');
      text.textContent = s.name + ' (' + s.point_count + ')';
      label.appendChild(cb);
      label.appendChild(sw);
      label.appendChild(text);
      label.title = 'Y: ' + s.values_ref + '\nX: ' + s.categories_ref;
      legendEl.appendChild(label);
    }
  }

  function visibleSeries(chart) {
    var out = [];
    for (var i = 0; i < chart.series.length; i++) {
      if (isVisible(chart, chart.series[i].series_index)) out.push(chart.series[i]);
    }
    return out;
  }

  function asNumber(text) {
    var n = Number(text);
    return isFinite(n) ? n : NaN;
  }

  function dataBounds(seriesList) {
    var minX = Infinity, maxX = -Infinity, minY = Infinity, maxY = -Infinity;
    var any = false;
    for (var i = 0; i < seriesList.length; i++) {
      var pts = seriesList[i].points;
      for (var j = 0; j < pts.length; j++) {
        var x = asNumber(pts[j].x);
        var y = asNumber(pts[j].y);
        if (!isFinite(x) || !isFinite(y)) continue;
        any = true;
        if (x < minX) minX = x;
        if (x > maxX) maxX = x;
        if (y < minY) minY = y;
        if (y > maxY) maxY = y;
      }
    }
    if (!any) return { minX: 0, maxX: 1, minY: 0, maxY: 1 };
    if (minX === maxX) { minX -= 1; maxX += 1; }
    if (minY === maxY) { minY -= 1; maxY += 1; }
    var padX = (maxX - minX) * 0.04;
    var padY = (maxY - minY) * 0.06;
    return {
      minX: minX - padX,
      maxX: maxX + padX,
      minY: minY - padY,
      maxY: maxY + padY
    };
  }

  function niceStep(span, target) {
    if (!(span > 0)) return 1;
    var raw = span / Math.max(target, 1);
    var pow = Math.pow(10, Math.floor(Math.log(raw) / Math.LN10));
    var n = raw / pow;
    var step;
    if (n <= 1) step = 1;
    else if (n <= 2) step = 2;
    else if (n <= 5) step = 5;
    else step = 10;
    return step * pow;
  }

  function formatTick(v) {
    if (!isFinite(v)) return '';
    var a = Math.abs(v);
    if (a !== 0 && (a >= 1e6 || a < 1e-3)) {
      return v.toExponential(2);
    }
    var s = String(Math.round(v * 1e6) / 1e6);
    return s;
  }

  var layout = null;

  function resizeCanvas() {
    var wrap = canvas.parentElement;
    var dpr = window.devicePixelRatio || 1;
    var w = Math.max(100, wrap.clientWidth);
    var h = Math.max(160, wrap.clientHeight);
    canvas.width = Math.round(w * dpr);
    canvas.height = Math.round(h * dpr);
    canvas.style.width = w + 'px';
    canvas.style.height = h + 'px';
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    return { w: w, h: h };
  }

  function draw() {
    var chart = currentChart();
    if (!chart) return;
    var size = resizeCanvas();
    var w = size.w, h = size.h;
    ctx.clearRect(0, 0, w, h);
    ctx.fillStyle = '#101820';
    ctx.fillRect(0, 0, w, h);

    var seriesList = visibleSeries(chart);
    var pad = { l: 64, r: 18, t: 18, b: 44 };
    var plotW = Math.max(10, w - pad.l - pad.r);
    var plotH = Math.max(10, h - pad.t - pad.b);
    var bounds = dataBounds(seriesList);

    layout = {
      pad: pad,
      plotW: plotW,
      plotH: plotH,
      bounds: bounds,
      chart: chart,
      seriesList: seriesList,
      w: w,
      h: h
    };

    function xToPx(x) {
      return pad.l + ((x - bounds.minX) / (bounds.maxX - bounds.minX)) * plotW;
    }
    function yToPx(y) {
      return pad.t + (1 - (y - bounds.minY) / (bounds.maxY - bounds.minY)) * plotH;
    }

    // grid + axes
    ctx.strokeStyle = '#263244';
    ctx.lineWidth = 1;
    var xStep = niceStep(bounds.maxX - bounds.minX, 6);
    var yStep = niceStep(bounds.maxY - bounds.minY, 6);
    var x0 = Math.ceil(bounds.minX / xStep) * xStep;
    var y0 = Math.ceil(bounds.minY / yStep) * yStep;

    ctx.beginPath();
    for (var xv = x0; xv <= bounds.maxX + xStep * 0.01; xv += xStep) {
      var xp = xToPx(xv);
      ctx.moveTo(xp, pad.t);
      ctx.lineTo(xp, pad.t + plotH);
    }
    for (var yv = y0; yv <= bounds.maxY + yStep * 0.01; yv += yStep) {
      var yp = yToPx(yv);
      ctx.moveTo(pad.l, yp);
      ctx.lineTo(pad.l + plotW, yp);
    }
    ctx.stroke();

    ctx.strokeStyle = '#3d4f63';
    ctx.beginPath();
    ctx.moveTo(pad.l, pad.t);
    ctx.lineTo(pad.l, pad.t + plotH);
    ctx.lineTo(pad.l + plotW, pad.t + plotH);
    ctx.stroke();

    ctx.fillStyle = '#6f8298';
    ctx.font = '11px Segoe UI, Tahoma, sans-serif';
    ctx.textAlign = 'center';
    ctx.textBaseline = 'top';
    for (xv = x0; xv <= bounds.maxX + xStep * 0.01; xv += xStep) {
      ctx.fillText(formatTick(xv), xToPx(xv), pad.t + plotH + 6);
    }
    ctx.textAlign = 'right';
    ctx.textBaseline = 'middle';
    for (yv = y0; yv <= bounds.maxY + yStep * 0.01; yv += yStep) {
      ctx.fillText(formatTick(yv), pad.l - 8, yToPx(yv));
    }

    // axis titles
    ctx.fillStyle = '#95a7ba';
    ctx.font = '12px Segoe UI, Tahoma, sans-serif';
    ctx.textAlign = 'center';
    ctx.textBaseline = 'bottom';
    ctx.fillText(chart.x_axis_title, pad.l + plotW / 2, h - 6);
    ctx.save();
    ctx.translate(14, pad.t + plotH / 2);
    ctx.rotate(-Math.PI / 2);
    ctx.textBaseline = 'top';
    ctx.fillText(chart.y_axis_title, 0, 0);
    ctx.restore();

    if (!seriesList.length) {
      ctx.fillStyle = '#95a7ba';
      ctx.textAlign = 'center';
      ctx.textBaseline = 'middle';
      ctx.fillText('Немає видимих серій', pad.l + plotW / 2, pad.t + plotH / 2);
      return;
    }

    // series lines + markers
    for (var i = 0; i < seriesList.length; i++) {
      var s = seriesList[i];
      var pts = s.points;
      ctx.strokeStyle = s.color;
      ctx.fillStyle = s.color;
      ctx.lineWidth = 1.6;
      ctx.beginPath();
      var started = false;
      for (var j = 0; j < pts.length; j++) {
        var x = asNumber(pts[j].x);
        var y = asNumber(pts[j].y);
        if (!isFinite(x) || !isFinite(y)) continue;
        var px = xToPx(x), py = yToPx(y);
        if (!started) { ctx.moveTo(px, py); started = true; }
        else ctx.lineTo(px, py);
      }
      ctx.stroke();

      var mark = pts.length <= 8 ? 3.5 : 2.2;
      for (j = 0; j < pts.length; j++) {
        x = asNumber(pts[j].x);
        y = asNumber(pts[j].y);
        if (!isFinite(x) || !isFinite(y)) continue;
        px = xToPx(x); py = yToPx(y);
        ctx.beginPath();
        ctx.arc(px, py, mark, 0, Math.PI * 2);
        ctx.fill();
      }
    }

    // hover highlight
    if (state.hover) {
      var hp = state.hover.point;
      var hx = asNumber(hp.x), hy = asNumber(hp.y);
      if (isFinite(hx) && isFinite(hy)) {
        var hpx = xToPx(hx), hpy = yToPx(hy);
        ctx.strokeStyle = '#ffffff';
        ctx.lineWidth = 1.5;
        ctx.beginPath();
        ctx.arc(hpx, hpy, 6, 0, Math.PI * 2);
        ctx.stroke();
        ctx.fillStyle = state.hover.series.color;
        ctx.beginPath();
        ctx.arc(hpx, hpy, 3.5, 0, Math.PI * 2);
        ctx.fill();
      }
    }
  }

  function findNearest(mx, my) {
    if (!layout) return null;
    var pad = layout.pad;
    var bounds = layout.bounds;
    var seriesList = layout.seriesList;
    if (!seriesList.length) return null;
    if (mx < pad.l - 8 || mx > pad.l + layout.plotW + 8 ||
        my < pad.t - 8 || my > pad.t + layout.plotH + 8) {
      return null;
    }

    function xToPx(x) {
      return pad.l + ((x - bounds.minX) / (bounds.maxX - bounds.minX)) * layout.plotW;
    }
    function yToPx(y) {
      return pad.t + (1 - (y - bounds.minY) / (bounds.maxY - bounds.minY)) * layout.plotH;
    }

    var best = null;
    var bestDist = 12 * 12; // px^2 threshold
    for (var i = 0; i < seriesList.length; i++) {
      var s = seriesList[i];
      for (var j = 0; j < s.points.length; j++) {
        var p = s.points[j];
        var x = asNumber(p.x), y = asNumber(p.y);
        if (!isFinite(x) || !isFinite(y)) continue;
        var dx = xToPx(x) - mx;
        var dy = yToPx(y) - my;
        var d2 = dx * dx + dy * dy;
        if (d2 < bestDist) {
          bestDist = d2;
          best = { series: s, point: p };
        }
      }
    }
    return best;
  }

  function renderHover(hit) {
    if (!hit) {
      hoverBox.innerHTML = '<div class="empty">Наведіть курсор на точку графіка</div>';
      return;
    }
    var s = hit.series;
    var p = hit.point;
    hoverBox.innerHTML =
      '<div class="hover-series"><span class="swatch" style="background:' +
      escapeHtml(s.color) + '"></span>' + escapeHtml(s.name) + '</div>' +
      '<div class="row"><span class="k">X</span><span class="v">' + escapeHtml(p.x) + '</span></div>' +
      '<div class="row"><span class="k">Y</span><span class="v">' + escapeHtml(p.y) + '</span></div>' +
      '<div class="row"><span class="k">Точка</span><span class="v">' + escapeHtml(String(p.i)) + '</span></div>' +
      '<div class="row"><span class="k">X-адреса</span><span class="v">' + escapeHtml(p.xa) + '</span></div>' +
      '<div class="row"><span class="k">Y-адреса</span><span class="v">' + escapeHtml(p.ya) + '</span></div>' +
      '<div class="row"><span class="k">values_ref</span><span class="v">' + escapeHtml(s.values_ref) + '</span></div>' +
      '<div class="row"><span class="k">categories_ref</span><span class="v">' + escapeHtml(s.categories_ref) + '</span></div>';
  }

  canvas.addEventListener('mousemove', function (ev) {
    var rect = canvas.getBoundingClientRect();
    var mx = ev.clientX - rect.left;
    var my = ev.clientY - rect.top;
    var hit = findNearest(mx, my);
    var prev = state.hover;
    var same = prev && hit &&
      prev.series.series_index === hit.series.series_index &&
      prev.point.i === hit.point.i;
    if (!hit && !prev) return;
    if (same) return;
    state.hover = hit;
    renderHover(hit);
    draw();
  });

  canvas.addEventListener('mouseleave', function () {
    if (!state.hover) return;
    state.hover = null;
    renderHover(null);
    draw();
  });

  document.getElementById('btn-show-all').addEventListener('click', function () {
    var chart = currentChart();
    if (!chart) return;
    var key = String(chart.chart_index);
    state.visible[key] = {};
    for (var i = 0; i < chart.series.length; i++) {
      state.visible[key][chart.series[i].series_index] = true;
    }
    state.hover = null;
    renderLegend(chart);
    renderHover(null);
    draw();
  });

  window.addEventListener('resize', function () { draw(); });

  if (charts.length) selectChart(charts[0].chart_index);
})();
</script>
</body>
</html>
"""


def main() -> None:
    payload = build_payload()
    data_json = json.dumps(payload, ensure_ascii=False, separators=(",", ":"))
    # Keep </script> from breaking out of the script tag if somehow present.
    data_json = data_json.replace("<", "\\u003c")
    html = HTML_TEMPLATE.replace("__DATA_JSON__", data_json)
    OUT_HTML.write_text(html, encoding="utf-8")
    meta = payload["meta"]
    print(f"Wrote {OUT_HTML}")
    print(
        f"charts={meta['chart_count']} series={meta['series_count']} "
        f"points={meta['point_count']} mismatches={meta['mismatches']}"
    )


if __name__ == "__main__":
    main()
