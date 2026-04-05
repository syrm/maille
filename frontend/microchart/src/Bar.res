// Bar.res

type series = {
  name:  string,
  data:  array<float>,
  color: option<string>,
}

type options = {
  width:  option<float>,
  height: option<float>,
  labels: option<array<string>>,
  yTicks: option<int>,
  gap:    option<float>,
  colors: option<array<string>>,
}

let make = (container: Core.element, series: array<series>, opts: options) => {
  let w       = Option.getOr(opts.width,  600.)
  let h       = Option.getOr(opts.height, 280.)
  let yTicks  = Option.getOr(opts.yTicks, 5)
  let gap     = Option.getOr(opts.gap,    0.25)
  let colors  = Option.getOr(opts.colors, Core.palette)
  let padTop    = 20.
  let padRight  = 16.
  let padBottom = 36.
  let padLeft   = 50.

  let nGroups = Array.reduce(series, 0, (acc, s) => max(acc, Array.length(s.data)))
  let labels  = Option.getOr(opts.labels,
    Array.fromInitializer(~length=nGroups, i => Int.toString(i + 1)))

  let allVals = Array.flatMap(series, s => s.data)
  let maxVal  = Core.niceMax(allVals)
  let plotW   = w - padLeft - padRight
  let plotH   = h - padTop  - padBottom

  let svg = Core.svgEl(w, h, container)
  let g   = Core.group(~transform=`translate(${Core.f2(padLeft)},${Core.f2(padTop)})`, svg)

  // Grid + Y labels
  Array.forEach(Core.linspace(0., maxVal, yTicks), v => {
    let y = plotH - (v / maxVal) * plotH
    Core.line(g, 0., y, plotW, y)
    Core.text(g, -6., y + 4., Core.fmtVal(v),
      [("text-anchor","end"),("fill","#8b949e"),("font-size","11")])->ignore
  })

  // Bars
  let groupW   = plotW / Float.fromInt(nGroups)
  let barTotal = groupW * (1. - gap)
  let barW     = barTotal / Float.fromInt(Array.length(series))
  let tip      = Core.makeTooltip()

  for gi in 0 to nGroups - 1 {
    let gx = Float.fromInt(gi) * groupW + (groupW * gap) / 2.
    let lbl = Option.getOr(Array.get(labels, gi), "")
    Core.text(g, gx + barTotal / 2., plotH + 18., lbl,
      [("text-anchor","middle"),("fill","#8b949e"),("font-size","11")])->ignore

    Array.forEachWithIndex(series, (s, si) => {
      let val_ = Option.getOr(Array.get(s.data, gi), 0.)
      let bx   = gx + Float.fromInt(si) * barW
      let bh   = (val_ / maxVal) * plotH
      let c    = Option.getOr(s.color, Core.getColor(~colors, si))
      let name = s.name
      let r    = Core.elIn("rect", [
        ("x",      Core.f2(bx)),
        ("y",      Core.f2(plotH - bh)),
        ("width",  Core.f2(barW - 1.)),
        ("height", Core.f2(bh)),
        ("fill",   c), ("rx", "2"), ("style", "cursor:pointer"),
      ], g)
      Core.addEventListener(r, "mouseenter", (e: Core.mouseEvent) =>
        Core.showTip(tip, `<b style="color:${c}">${name}</b><br/>${lbl}: <b>${Core.fmtVal(val_)}</b>`, e))
      Core.addEventListener(r, "mousemove",  (e: Core.mouseEvent) => Core.moveTip(tip, e))
      Core.addEventListener(r, "mouseleave", (_: Core.mouseEvent) => Core.hideTip(tip))
    })
  }

  Core.line(g, 0., plotH, plotW, plotH, ~stroke="#30363d")
  () => Core.removeTip(tip)
}
