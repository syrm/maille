import { el, makeSvg, txt, makeTooltip, showTip, moveTip, hideTip,
         niceMax, linspace, fmtVal, PALETTE, type TooltipOptions } from './core'

export interface LineSeries { name: string; data: number[]; color?: string; area?: boolean; dashed?: boolean }

export interface LineChartOptions {
  width?:   number
  height?:  number
  labels?:  string[]
  padding?: { top?: number; right?: number; bottom?: number; left?: number }
  yTicks?:  number
  smooth?:  boolean
  points?:  boolean
  tooltip?: TooltipOptions
  colors?:  string[]
}

export function lineChart(
  container: HTMLElement,
  series: LineSeries[],
  opts: LineChartOptions = {}
): () => void {
  const W = opts.width  ?? (container.clientWidth  || 400)
  const H = opts.height ?? (container.clientHeight || 220)
  const pad = { top: 20, right: 16, bottom: 36, left: 44, ...opts.padding }
  const colors  = opts.colors  ?? PALETTE
  const yTicks  = opts.yTicks  ?? 5
  const smooth  = opts.smooth  ?? true
  const showPts = opts.points  ?? true

  const nPoints = Math.max(...series.map(s => s.data.length))
  const labels  = opts.labels ?? Array.from({ length: nPoints }, (_, i) => String(i + 1))
  const allVals = series.flatMap(s => s.data)
  const maxVal  = niceMax(allVals)

  const plotW = W - pad.left - pad.right
  const plotH = H - pad.top  - pad.bottom

  const svg = makeSvg(container, W, H)
  const g   = el('g', { transform: `translate(${pad.left},${pad.top})` }, svg)

  // Grid + Y axis
  for (const v of linspace(0, maxVal, yTicks)) {
    const y = plotH - (v / maxVal) * plotH
    el('line', { x1: 0, y1: y, x2: plotW, y2: y,
      stroke: '#21262d', 'stroke-width': 1 }, g)
    txt(g, fmtVal(v), { x: -6, y: y + 4, 'text-anchor': 'end',
      fill: '#8b949e', 'font-size': 11 })
  }

  // X labels
  const step = plotW / (nPoints - 1)
  for (let i = 0; i < nPoints; i++) {
    txt(g, labels[i], { x: i * step, y: plotH + 18,
      'text-anchor': 'middle', fill: '#8b949e', 'font-size': 11 })
  }

  const tip = opts.tooltip?.enabled !== false ? makeTooltip() : null

  // Series
  for (let si = 0; si < series.length; si++) {
    const s     = series[si]
    const color = s.color ?? colors[si % colors.length]
    const pts   = s.data.map((v, i) => [i * step, plotH - (v / maxVal) * plotH] as [number, number])

    // Build path
    let d = ''
    if (smooth && pts.length > 2) {
      d = `M${pts[0][0]},${pts[0][1]}`
      for (let i = 1; i < pts.length; i++) {
        const [px, py] = pts[i - 1], [cx, cy] = pts[i]
        const cpx = (px + cx) / 2
        d += ` C${cpx},${py} ${cpx},${cy} ${cx},${cy}`
      }
    } else {
      d = pts.map((p, i) => `${i ? 'L' : 'M'}${p[0]},${p[1]}`).join(' ')
    }

    // Area fill
    if (s.area) {
      el('path', {
        d: `${d} L${pts.at(-1)![0]},${plotH} L${pts[0][0]},${plotH} Z`,
        fill: color, opacity: 0.12, stroke: 'none',
      }, g)
    }

    // Line
    el('path', {
      d, fill: 'none', stroke: color, 'stroke-width': 2,
      ...(s.dashed ? { 'stroke-dasharray': '5,3' } : {}),
    }, g)

    // Points + tooltip targets
    if (showPts || tip) {
      for (let i = 0; i < pts.length; i++) {
        const [px, py] = pts[i]
        const val  = s.data[i]
        const name = s.name

        if (showPts) el('circle', { cx: px, cy: py, r: 4,
          fill: color, stroke: '#0d1117', 'stroke-width': 2 }, g)

        if (tip) {
          const target = el('circle', { cx: px, cy: py, r: 8,
            fill: 'transparent', style: 'cursor:pointer' }, g)
          target.addEventListener('mouseenter', e => showTip(tip,
            `<b style="color:${color}">${name}</b><br/>${labels[i]}: <b>${fmtVal(val)}</b>`, e as MouseEvent))
          target.addEventListener('mousemove',  e => moveTip(tip, e as MouseEvent))
          target.addEventListener('mouseleave', () => hideTip(tip))
        }
      }
    }
  }

  // X axis line
  el('line', { x1: 0, y1: plotH, x2: plotW, y2: plotH,
    stroke: '#30363d', 'stroke-width': 1 }, g)

  return () => tip?.remove()
}
