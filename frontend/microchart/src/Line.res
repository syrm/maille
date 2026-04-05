// Line.res

type series = {
  name:   string,
  data:   array<float>,
  color:  option<string>,
  area:   bool,
  dashed: bool,
}

type options = {
  width:  option<float>,
  height: option<float>,
  labels: option<array<string>>,
  yTicks: option<int>,
  smooth: bool,
  points: bool,
  colors: option<array<string>>,
}

let make = (container: Core.element, series: array<series>, opts: options) => {
  let w       = Option.getOr(opts.width,  600.)
  let h       = Option.getOr(opts.height, 240.)
  let yTicks  = Option.getOr(opts.yTicks, 5)
  let colors  = Option.getOr(opts.colors, Core.palette)
  let padTop    = 20.
  let padRight  = 16.
  let padBottom = 36.
  let padLeft   = 50.

  let nPoints = Array.reduce(series, 0, (acc, s) => max(acc, Array.length(s.data)))
  let labels  = Option.getOr(opts.labels,
    Array.fromInitializer(~length=nPoints, i => Int.toString(i + 1)))

  let allVals = Array.flatMap(series, s => s.data)
  let maxVal  = Core.niceMax(allVals)
  let plotW   = w - padLeft - padRight
  let plotH   = h - padTop  - padBottom
  let step    = plotW / Float.fromInt(nPoints - 1)

  let svg = Core.svgEl(w, h, container)
  let g   = Core.group(~transform=`translate(${Core.f2(padLeft)},${Core.f2(padTop)})`, svg)

  Array.forEach(Core.linspace(0., maxVal, yTicks), v => {
    let y = plotH - (v / maxVal) * plotH
    Core.line(g, 0., y, plotW, y)
    Core.text(g, -6., y + 4., Core.fmtVal(v),
      [("text-anchor","end"),("fill","#8b949e"),("font-size","11")])->ignore
  })

  Array.forEachWithIndex(labels, (lbl, i) => {
    let x = Float.fromInt(i) * step
    Core.text(g, x, plotH + 18., lbl,
      [("text-anchor","middle"),("fill","#8b949e"),("font-size","11")])->ignore
  })

  let tip = Core.makeTooltip()

  Array.forEachWithIndex(series, (s, si) => {
    let c   = Option.getOr(s.color, Core.getColor(~colors, si))
    let pts = Array.mapWithIndex(s.data, (v, i) =>
      (Float.fromInt(i) * step, plotH - (v / maxVal) * plotH))

    // Build path
    let d = if opts.smooth && Array.length(pts) > 2 {
      let (x0, y0) = Array.getUnsafe(pts, 0)
      let init = `M${Core.f2(x0)},${Core.f2(y0)}`
      Array.reduceWithIndex(pts, init, (acc, (cx, cy), i) =>
        if i == 0 { acc }
        else {
          let (px, py) = Array.getUnsafe(pts, i - 1)
          let cpx = (px + cx) / 2.
          acc ++ ` C${Core.f2(cpx)},${Core.f2(py)} ${Core.f2(cpx)},${Core.f2(cy)} ${Core.f2(cx)},${Core.f2(cy)}`
        })
    } else {
      Array.mapWithIndex(pts, ((x, y), i) =>
        `${i == 0 ? "M" : "L"}${Core.f2(x)},${Core.f2(y)}`
      )->Array.join(" ")
    }

    if s.area {
      let (lx, _) = Array.getUnsafe(pts, Array.length(pts) - 1)
      let (x0, _) = Array.getUnsafe(pts, 0)
      Core.path(g,
        `${d} L${Core.f2(lx)},${Core.f2(plotH)} L${Core.f2(x0)},${Core.f2(plotH)} Z`,
        [("fill", c), ("opacity", "0.12"), ("stroke", "none")])
    }

    let dashAttr = s.dashed ? [("stroke-dasharray","5,3")] : []
    Core.path(g, d, Array.concat(
      [("fill","none"),("stroke",c),("stroke-width","2")], dashAttr))

    Array.forEachWithIndex(pts, ((px, py), i) => {
      if opts.points {
        Core.circle(g, px, py, 4.,
          [("fill",c),("stroke","#0d1117"),("stroke-width","2")])
      }
      let target = Core.elIn("circle", [
        ("cx", Core.f2(px)), ("cy", Core.f2(py)),
        ("r", "8"), ("fill","transparent"), ("style","cursor:pointer"),
      ], g)
      let val_  = Array.getUnsafe(s.data, i)
      let lbl   = Option.getOr(Array.get(labels, i), "")
      let name  = s.name
      Core.addEventListener(target, "mouseenter", (e: Core.mouseEvent) =>
        Core.showTip(tip, `<b style="color:${c}">${name}</b><br/>${lbl}: <b>${Core.fmtVal(val_)}</b>`, e))
      Core.addEventListener(target, "mousemove",  (e: Core.mouseEvent) => Core.moveTip(tip, e))
      Core.addEventListener(target, "mouseleave", (_: Core.mouseEvent) => Core.hideTip(tip))
    })
  })

  Core.line(g, 0., plotH, plotW, plotH, ~stroke="#30363d")
  () => Core.removeTip(tip)
}
