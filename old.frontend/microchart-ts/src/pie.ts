import { el, makeSvg, makeTooltip, showTip, moveTip, hideTip,
         PALETTE, type TooltipOptions } from './core'

export interface PieSlice { label: string; value: number; color?: string }

export interface PieChartOptions {
  width?:   number
  height?:  number
  donut?:   boolean
  donutWidth?: number        // px, default 40
  padAngle?:   number        // gap between slices in radians, default 0.015
  tooltip?: TooltipOptions
  colors?:  string[]
  label?:   boolean | 'percent' | 'value'
}

function arc(
  cx: number, cy: number, r: number,
  startAngle: number, endAngle: number,
  innerR = 0
): string {
  const x1 = cx + r * Math.cos(startAngle), y1 = cy + r * Math.sin(startAngle)
  const x2 = cx + r * Math.cos(endAngle),   y2 = cy + r * Math.sin(endAngle)
  const large = endAngle - startAngle > Math.PI ? 1 : 0

  if (innerR === 0) {
    return `M${cx},${cy} L${x1},${y1} A${r},${r} 0 ${large},1 ${x2},${y2} Z`
  }
  const ix1 = cx + innerR * Math.cos(endAngle),   iy1 = cy + innerR * Math.sin(endAngle)
  const ix2 = cx + innerR * Math.cos(startAngle), iy2 = cy + innerR * Math.sin(startAngle)
  return `M${x1},${y1} A${r},${r} 0 ${large},1 ${x2},${y2}
          L${ix1},${iy1} A${innerR},${innerR} 0 ${large},0 ${ix2},${iy2} Z`
}

export function pieChart(
  container: HTMLElement,
  slices: PieSlice[],
  opts: PieChartOptions = {}
): () => void {
  const W = opts.width  ?? (container.clientWidth  || 260)
  const H = opts.height ?? (container.clientHeight || 220)
  const colors     = opts.colors  ?? PALETTE
  const donut      = opts.donut   ?? false
  const donutW     = opts.donutWidth ?? 40
  const padAngle   = opts.padAngle   ?? 0.018
  const showLabel  = opts.label !== false

  const svg  = makeSvg(container, W, H)
  const cx   = W / 2, cy = H / 2
  const r    = Math.min(cx, cy) - 8
  const ir   = donut ? r - donutW : 0
  const total = slices.reduce((s, sl) => s + sl.value, 0)

  const tip = opts.tooltip?.enabled !== false ? makeTooltip() : null

  let angle = -Math.PI / 2   // start at top

  for (let i = 0; i < slices.length; i++) {
    const sl      = slices[i]
    const pct     = sl.value / total
    const span    = pct * 2 * Math.PI - padAngle
    const color   = sl.color ?? colors[i % colors.length]
    const mid     = angle + span / 2

    const path = el('path', {
      d: arc(cx, cy, r, angle, angle + span, ir),
      fill: color, style: 'cursor:pointer',
      stroke: '#0d1117', 'stroke-width': 1,
    }, svg)

    // Label in slice
    if (showLabel && pct > 0.05) {
      const lr = ir ? (r + ir) / 2 : r * 0.65
      const lx = cx + lr * Math.cos(mid)
      const ly = cy + lr * Math.sin(mid)
      const labelTxt = opts.label === 'value'
        ? String(sl.value)
        : Math.round(pct * 100) + '%'
      const t = el('text', {
        x: lx, y: ly, 'text-anchor': 'middle',
        'dominant-baseline': 'middle', fill: '#fff',
        'font-size': 11, 'font-weight': 600,
        style: 'pointer-events:none',
      }, svg)
      t.textContent = labelTxt
    }

    if (tip) {
      const pctStr = Math.round(pct * 100) + '%'
      path.addEventListener('mouseenter', e => showTip(tip,
        `<b style="color:${color}">${sl.label}</b><br/>${pctStr} (${sl.value})`, e as MouseEvent))
      path.addEventListener('mousemove',  e => moveTip(tip, e as MouseEvent))
      path.addEventListener('mouseleave', () => hideTip(tip))

      // Hover: léger scale
      path.addEventListener('mouseenter', () => path.setAttribute('transform',
        `translate(${Math.cos(mid) * 4},${Math.sin(mid) * 4})`))
      path.addEventListener('mouseleave', () => path.removeAttribute('transform'))
    }

    angle += span + padAngle
  }

  // Centre label for donut
  if (donut) {
    const t = el('text', {
      x: cx, y: cy, 'text-anchor': 'middle',
      'dominant-baseline': 'middle', fill: '#e6edf3',
      'font-size': 14, 'font-weight': 700,
    }, svg)
    t.textContent = total.toLocaleString()
  }

  return () => tip?.remove()
}
