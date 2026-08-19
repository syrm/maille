type slice = {
  label: string,
  value: float,
  color: option<string>,
}

type options = {
  width:      option<float>,
  height:     option<float>,
  donut:      bool,
  donutWidth: option<float>,
  padAngle:   option<float>,
  showLabel:  bool,
  colors:     option<array<string>>,
}

type chart = {
  slices: array<slice>,
  options: option<options>,
}

let parseSlice = (json: JSON.t): slice =>
  switch json {
  | Object(dict{
      "label": JSON.String(label),
      "value": JSON.Number(value),
      "color": ?color,
    }) =>
    let color = Core.optField(color, Core.asString)
    {label, value, color}
  | _ => throw(Core.InvalidJson(Core.InvalidData("")))
  }

let parseOptions = (json: JSON.t): options =>
  switch json {
  | Object(dict{
      "width":  ?width,
      "height": ?height,
      "donut": JSON.Boolean(donut),
      "donutWidth": ?donutWidth,
      "padAngle": ?padAngle,
      "showLabel": JSON.Boolean(showLabel),
      "colors": ?colors,
    }) => {
      width:  Core.optField(width,  Core.asFloat),
      height: Core.optField(height, Core.asFloat),
      donut: donut,
      donutWidth: Core.optField(donutWidth, Core.asFloat),
      padAngle: Core.optField(padAngle, Core.asFloat),
      showLabel: showLabel,
      colors: Core.optField(colors, Core.asStringArray),
    }
  | _ => throw(Core.InvalidJson(Core.InvalidOption("")))
  }

let parseChart = (json: JSON.t): chart =>
  switch json {
  | Object(dict{
    "slices": JSON.Array(rawSlices),
    "options": ?rawOpts
    }) =>
    let slices = rawSlices->Array.map(parseSlice)
    let options = switch rawOpts {
    | None | Some(Null) => None
    | Some(v) => Some(parseOptions(v))
    }
    {slices, options}
  | _ => throw(Core.InvalidJson(Core.InvalidChart("")))
  }

let arcPath = (cx, cy, r, ir, start, end_) => {
  let x1    = cx + r * Math.cos(start)
  let y1    = cy + r * Math.sin(start)
  let x2    = cx + r * Math.cos(end_)
  let y2    = cy + r * Math.sin(end_)
  let large = end_ - start > Math.Constants.pi ? "1" : "0"
  let f     = Core.f2

  if ir == 0. {
    `M${f(cx)},${f(cy)} L${f(x1)},${f(y1)} A${f(r)},${f(r)} 0 ${large},1 ${f(x2)},${f(y2)} Z`
  } else {
    let ix1 = cx + ir * Math.cos(end_)
    let iy1 = cy + ir * Math.sin(end_)
    let ix2 = cx + ir * Math.cos(start)
    let iy2 = cy + ir * Math.sin(start)
    `M${f(x1)},${f(y1)} A${f(r)},${f(r)} 0 ${large},1 ${f(x2)},${f(y2)} ` ++
    `L${f(ix1)},${f(iy1)} A${f(ir)},${f(ir)} 0 ${large},0 ${f(ix2)},${f(iy2)} Z`
  }
}

let make = (container: Dom.element, slices: array<slice>, opts: options) => {
  let w      = Option.getOr(opts.width,      280.)
  let h      = Option.getOr(opts.height,     240.)
  let colors = Option.getOr(opts.colors,     Core.palette)
  let donutW = Option.getOr(opts.donutWidth, 40.)
  let padAng = Option.getOr(opts.padAngle,   0.018)

  let cx    = w / 2.
  let cy    = h / 2.
  let r     = min(cx, cy) - 8.
  let ir    = opts.donut ? r - donutW : 0.
  let total = Array.reduce(slices, 0., (acc, s) => acc + s.value)

  let svg   = Core.svgEl(w, h, container)
  let tip   = Core.makeTooltip()
  let angle = ref(-. Math.Constants.pi / 2.)

  Array.forEachWithIndex(slices, (sl, i) => {
    let pct  = sl.value / total
    let span = pct * 2. * Math.Constants.pi - padAng
    let c    = Option.getOr(sl.color, Core.getColor(~colors, i))
    let mid  = angle.contents + span / 2.

    let d = arcPath(cx, cy, r, ir, angle.contents, angle.contents + span)
    let p = Core.elIn("path", [
      ("d", d), ("fill", c),
      ("stroke", "#0d1117"), ("stroke-width", "1"),
      ("style", "cursor:pointer"),
    ], svg)

    if opts.showLabel && pct > 0.05 {
      let lr = ir > 0. ? (r + ir) / 2. : r * 0.65
      let lx = cx + lr * Math.cos(mid)
      let ly = cy + lr * Math.sin(mid)
      Core.text(svg, lx, ly, `${Float.toFixed(pct * 100., ~digits=0)}%`, [
        ("text-anchor",       "middle"),
        ("dominant-baseline", "middle"),
        ("fill",              "#fff"),
        ("font-size",         "11"),
        ("font-weight",       "600"),
        ("style",             "pointer-events:none"),
      ])->ignore
    }

    let pctStr = `${Float.toFixed(pct * 100., ~digits=0)}%`
    let lbl    = sl.label
    let val_   = sl.value
    Core.addEventListener(p, "mouseenter", (e: Dom.mouseEvent) => {
      Core.showTip(tip, `<b style="color:${c}">${lbl}</b><br/>${pctStr} (${Core.fmtVal(val_)})`, e)
      Core.setAttribute(p, "transform",
        `translate(${Core.f2(Math.cos(mid) * 4.)},${Core.f2(Math.sin(mid) * 4.)})`)
    })
    Core.addEventListener(p, "mousemove",  (e: Dom.mouseEvent) => Core.moveTip(tip, e))
    Core.addEventListener(p, "mouseleave", (_: Dom.mouseEvent) => {
      Core.hideTip(tip)
      Core.removeAttribute(p, "transform")
    })

    angle := angle.contents + span + padAng
  })

  if opts.donut {
    Core.text(svg, cx, cy, Core.fmtVal(total), [
      ("text-anchor",       "middle"),
      ("dominant-baseline", "middle"),
      ("fill",              "#e6edf3"),
      ("font-size",         "14"),
      ("font-weight",       "700"),
    ])->ignore
  }

  () => Core.removeTip(tip)
}
