// ─── Types ────────────────────────────────────────────────────────────────────

export interface ChartPadding { top: number; right: number; bottom: number; left: number }

export interface TooltipOptions {
  enabled?: boolean
  format?: (value: number, label: string) => string
}

// ─── SVG helpers ──────────────────────────────────────────────────────────────

const NS = 'http://www.w3.org/2000/svg'

export function el<K extends keyof SVGElementTagNameMap>(
  tag: K,
  attrs: Record<string, string | number> = {},
  parent?: SVGElement | SVGSVGElement
): SVGElementTagNameMap[K] {
  const e = document.createElementNS(NS, tag)
  for (const [k, v] of Object.entries(attrs)) e.setAttribute(k, String(v))
  parent?.appendChild(e)
  return e
}

export function makeSvg(container: HTMLElement, w: number, h: number): SVGSVGElement {
  container.innerHTML = ''
  return el('svg', { viewBox: `0 0 ${w} ${h}`, width: '100%', height: '100%',
    style: 'display:block;overflow:visible' }, container as unknown as SVGElement)
}

export function txt(
  parent: SVGElement, content: string,
  attrs: Record<string, string | number> = {}
): SVGTextElement {
  const t = el('text', attrs, parent)
  t.textContent = content
  return t
}

// ─── Tooltip ──────────────────────────────────────────────────────────────────

export function makeTooltip(): HTMLDivElement {
  const d = document.createElement('div')
  Object.assign(d.style, {
    position: 'fixed', pointerEvents: 'none', opacity: '0',
    background: '#1c2128', border: '1px solid #30363d', borderRadius: '6px',
    padding: '5px 10px', fontSize: '12px', color: '#e6edf3',
    transition: 'opacity .15s', zIndex: '9999', whiteSpace: 'nowrap',
  })
  document.body.appendChild(d)
  return d
}

export function showTip(tip: HTMLDivElement, html: string, e: MouseEvent): void {
  tip.innerHTML = html
  tip.style.opacity = '1'
  moveTip(tip, e)
}

export function moveTip(tip: HTMLDivElement, e: MouseEvent): void {
  const x = e.clientX + 12, y = e.clientY - 28
  tip.style.left = x + 'px'
  tip.style.top  = y + 'px'
}

export function hideTip(tip: HTMLDivElement): void {
  tip.style.opacity = '0'
}

// ─── Scale helpers ────────────────────────────────────────────────────────────

export function niceMax(values: number[]): number {
  const max = Math.max(...values)
  if (max === 0) return 10
  const mag = Math.pow(10, Math.floor(Math.log10(max)))
  return Math.ceil(max / mag) * mag
}

export function linspace(min: number, max: number, n: number): number[] {
  return Array.from({ length: n }, (_, i) => min + (i / (n - 1)) * (max - min))
}

export function fmtVal(v: number): string {
  if (Math.abs(v) >= 1_000_000) return (v / 1_000_000).toFixed(1) + 'M'
  if (Math.abs(v) >= 1_000)     return (v / 1_000).toFixed(1) + 'K'
  return Number.isInteger(v) ? String(v) : v.toFixed(2)
}

// ─── Default palette ──────────────────────────────────────────────────────────

export const PALETTE = [
  '#58a6ff', '#3fb950', '#f0883e', '#a371f7',
  '#f85149', '#39d353', '#ffa657', '#79c0ff',
]
