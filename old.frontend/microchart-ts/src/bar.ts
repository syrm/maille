import { el, makeSvg, txt, makeTooltip, showTip, moveTip, hideTip,
         niceMax, linspace, fmtVal, PALETTE, type TooltipOptions } from './core'

export interface BarSeries { name: string; data: number[]; color?: string }

export interface BarChartOptions {
  width?:   number
  height?:  number
  labels?:  string[]
  padding?: { top?: number; right?: number; bottom?: number; left?: number }
  yTicks?:  number
  gap?:     number        // gap between groups (0-1)
  tooltip?: TooltipOptions
  colors?:  string[]
}

export function barChart(
  container: HTMLElement,
  series: BarSeries[],
  opts: BarChartOptions = {}
): () => void {
  const W = opts.width  ?? (container.clientWidth  || 400)
  const H = opts.height ?? (container.clientHeight || 260)
  const pad = { top: 20, right: 16, bottom: 36, left: 44, ...opts.padding }
  const colors = opts.colors ?? PALETTE
  const yTicks = opts.yTicks ?? 5
  const gap    = opts.gap    ?? 0.25

  const nGroups = Math.max(...series.map(s => s.data.length))
  const labels  = opts.labels ?? Array.from({ length: nGroups }, (_, i) => String(i + 1))
  const allVals = series.flatMap(s => s.data)
  const maxVal  = niceMax(allVals)

  const plotW = W - pad.left - pad.right
  const plotH = H - pad.top  - pad.bottom

  const svg = makeSvg(container, W, H)
  const g   = el('g', { transform: `translate(${pad.left},${pad.top})` }, svg)

  // Grid + Y axis
  const ticks = linspace(0, maxVal, yTicks)
  for (const v of ticks) {
    const y = plotH - (v / maxVal) * plotH
    el('line', { x1: 0, y1: y, x2: plotW, y2: y,
      stroke: '#21262d', 'stroke-width': 1 }, g)
    txt(g, fmtVal(v), { x: -6, y: y + 4, 'text-anchor': 'end',
      fill: '#8b949e', 'font-size': 11 })
  }

  // Bars
  const groupW   = plotW / nGroups
  const barTotal = groupW * (1 - gap)
  const barW     = barTotal / series.length

  const tip = opts.tooltip?.enabled !== false ? makeTooltip() : null

  for (let gi = 0; gi < nGroups; gi++) {
    const gx = gi * groupW + (groupW * gap) / 2

    // X label
    txt(g, labels[gi], {
      x: gx + barTotal / 2, y: plotH + 18,
      'text-anchor': 'middle', fill: '#8b949e', 'font-size': 11,
    })

    for (let si = 0; si < series.length; si++) {
      const val    = series[si].data[gi] ?? 0
      const bx     = gx + si * barW
      const bh     = (val / maxVal) * plotH
      const color  = series[si].color ?? colors[si % colors.length]
      const name   = series[si].name

      const rect = el('rect', {
        x: bx, y: plotH - bh, width: barW - 1, height: bh,
        fill: color, rx: 2, style: 'cursor:pointer',
      }, g)

      if (tip) {
        rect.addEventListener('mouseenter', e => showTip(tip,
          `<b style="color:${color}">${name}</b><br/>${labels[gi]}: <b>${fmtVal(val)}</b>`, e as MouseEvent))
        rect.addEventListener('mousemove',  e => moveTip(tip, e as MouseEvent))
        rect.addEventListener('mouseleave', () => hideTip(tip))
      }
    }
  }

  // X axis line
  el('line', { x1: 0, y1: plotH, x2: plotW, y2: plotH,
    stroke: '#30363d', 'stroke-width': 1 }, g)

  return () => tip?.remove()
}
