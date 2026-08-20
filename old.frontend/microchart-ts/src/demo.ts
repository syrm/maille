import { barChart, lineChart, pieChart, PALETTE } from './index'

const MONTHS = ['Jan','Fév','Mar','Avr','Mai','Juin','Juil','Août','Sep','Oct','Nov','Déc']

function walk(n: number, base: number, vol: number): number[] {
  let v = base
  return Array.from({ length: n }, () => { v += (Math.random() - 0.49) * vol; return +v.toFixed(2) })
}

// ── Bar ───────────────────────────────────────────────────────────
const barSeries = [
  { name: 'Produit A', data: [42,58,71,63,89,74,95,82,68,77,91,105] },
  { name: 'Produit B', data: [28,35,41,55,48,62,58,71,83,69,74,88]  },
  { name: 'Produit C', data: [15,22,18,31,27,34,29,38,42,35,41,55]  },
]
barChart(document.getElementById('bar')!, barSeries, { labels: MONTHS, height: 240 })

const barLegend = document.getElementById('bar-legend')!
barSeries.forEach((s, i) => {
  barLegend.innerHTML += `<span class="legend-item"><i style="background:${PALETTE[i]}"></i>${s.name}</span>`
})

// ── Line area ─────────────────────────────────────────────────────
lineChart(document.getElementById('line')!, [
  { name: 'EUR/USD', data: walk(12, 1.085, 0.008), area: true },
], { labels: MONTHS, height: 220 })

// ── Pie ───────────────────────────────────────────────────────────
const pieSlices = [
  { label: 'Infrastructure', value: 34 },
  { label: 'Marketing',      value: 22 },
  { label: 'R&D',            value: 28 },
  { label: 'RH',             value: 10 },
  { label: 'Divers',         value: 6  },
]
pieChart(document.getElementById('pie')!, pieSlices, { height: 220 })

const pieLegend = document.getElementById('pie-legend')!
pieSlices.forEach((s, i) => {
  pieLegend.innerHTML += `<span class="legend-item"><i style="background:${PALETTE[i]}"></i>${s.label} (${s.value}%)</span>`
})

// ── Multi line ────────────────────────────────────────────────────
lineChart(document.getElementById('multi')!, [
  { name: 'BTC/1k', data: walk(12, 42, 3) },
  { name: 'ETH/100',data: walk(12, 22, 2), dashed: true },
  { name: 'SOL',    data: walk(12, 95, 7), dashed: true },
], { labels: MONTHS, height: 220, points: false })

// ── Donut ─────────────────────────────────────────────────────────
const donutSlices = [
  { label: 'Complété',   value: 63 },
  { label: 'En cours',   value: 20 },
  { label: 'En attente', value: 17 },
]
pieChart(document.getElementById('donut')!, donutSlices, { donut: true, donutWidth: 45, height: 220 })

const donutLegend = document.getElementById('donut-legend')!
donutSlices.forEach((s, i) => {
  donutLegend.innerHTML += `<span class="legend-item"><i style="background:${PALETTE[i]}"></i>${s.label} (${s.value}%)</span>`
})

// ── Bundle size badge ─────────────────────────────────────────────
// On affiche les tailles injectées au build
const badges = document.getElementById('badges')!
declare const __BUNDLE_MIN__: string
declare const __BUNDLE_GZ__: string
try {
  badges.innerHTML = `
    <span class="badge g">✓ ${__BUNDLE_MIN__} min</span>
    <span class="badge b">✓ ${__BUNDLE_GZ__} gzip</span>
    <span class="badge y">SVG · 0 deps</span>
  `
} catch {
  badges.innerHTML = `<span class="badge y">SVG · 0 deps</span>`
}
