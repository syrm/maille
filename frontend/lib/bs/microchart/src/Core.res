// Core.res — SVG helpers, palette, scale utils

let svgNS = "http://www.w3.org/2000/svg"

// ─── Raw DOM bindings ────────────────────────────────────────────────────────
@val @scope("document")
external createElementNS: (string, string) => Dom.element = "createElementNS"

@val @scope("document")
external createElement_: string => Dom.element = "createElement"

@send external getElementById: (Dom.document, string) => Nullable.t<Dom.element> = "getElementById"

@send external setAttribute: (Dom.element, string, string) => unit = "setAttribute"
@send external appendChild: (Dom.element, Dom.element) => Dom.element = "appendChild"
@send external remove: Dom.element => unit = "remove"
@set  external setInnerHTML: (Dom.element, string) => unit = "innerHTML"
@send external addEventListener: (Dom.element, string, 'a => unit) => unit = "addEventListener"
@send external removeAttribute: (Dom.element, string) => unit = "removeAttribute"

@val @scope("document") external body: Dom.element = "body"

// style proxy — untyped on purpose (@get car c'est une propriété, pas une méthode)
@get external getStyle: Dom.element => {..} = "style"

@get external clientX: Dom.mouseEvent => int = "clientX"
@get external clientY: Dom.mouseEvent => int = "clientY"

// ── Helper ───────────────────────────────────────────────────────────────────
type invalidJson =
  | InvalidData(string)
  | InvalidOption(string)
  | InvalidChart(string)
  | InvalidString(string)
  | InvalidFloat(string)
  | InvalidInt(string)
  | InvalidStringArray(string)

exception InvalidJson(invalidJson)

let optField = (v: option<JSON.t>, decode: JSON.t => 'a): option<'a> =>
  switch v {
  | None | Some(Null) => None
  | Some(x) => Some(decode(x))
  }

let asString = v => switch v {
| JSON.String(n) => n
| _ => throw(InvalidJson(InvalidString("")))
}

let asFloat = v => switch v {
| JSON.Number(n) => n
| _ => throw(InvalidJson(InvalidFloat("")))
}

let asInt = v => switch v {
| JSON.Number(n) => Float.toInt(n)
| _ => throw(InvalidJson(InvalidInt("")))
}

let asStringArray = v => switch v {
| JSON.Array(arr) =>
  arr->Array.map(value =>
    switch value {
    | String(s) => s
    | _ => throw(InvalidJson(InvalidString("in array")))
    }
  )
| _ => throw(InvalidJson(InvalidStringArray("")))
}

// ─── Palette ─────────────────────────────────────────────────────────────────

let palette = [
  "#58a6ff", "#3fb950", "#f0883e", "#a371f7",
  "#f85149", "#39d353", "#ffa657", "#79c0ff",
]

let getColor = (~colors=palette, i) =>
  Array.get(colors, mod(i, Array.length(colors)))->Option.getOr("#58a6ff")

// ─── SVG builder ─────────────────────────────────────────────────────────────

let el = (tag, attrs) => {
  let e = createElementNS(svgNS, tag)
  Array.forEach(attrs, ((k, v)) => setAttribute(e, k, v))
  e
}

// Crée un élément SVG et l'insère dans parent — ordre correct
let elIn = (tag, attrs, parent) => {
  let e = el(tag, attrs)
  appendChild(parent, e)->ignore
  e
}

let svgEl = (w, h, container) => {
  setInnerHTML(container, "")
  elIn("svg", [
    ("viewBox", `0 0 ${Float.toString(w)} ${Float.toString(h)}`),
    ("width",   "100%"),
    ("height",  "100%"),
    ("xmlns",   svgNS),
    ("style",   "display:block;overflow:visible"),
  ], container)
}

let group = (~transform="", parent) =>
  elIn("g", transform === "" ? [] : [("transform", transform)], parent)

let f2 = v => Float.toFixed(v, ~digits=2)

let line = (parent, x1, y1, x2, y2, ~stroke="#21262d", ~width="1") =>
  elIn("line", [
    ("x1", f2(x1)), ("y1", f2(y1)),
    ("x2", f2(x2)), ("y2", f2(y2)),
    ("stroke", stroke), ("stroke-width", width),
  ], parent)->ignore

let rect = (parent, x, y, w, h, ~fill, ~rx="2") =>
  elIn("rect", [
    ("x", f2(x)), ("y", f2(y)),
    ("width", f2(w)), ("height", f2(h)),
    ("fill", fill), ("rx", rx),
  ], parent)->ignore

let path = (parent, d, attrs) =>
  elIn("path", Array.concat([("d", d)], attrs), parent)->ignore

let circle = (parent, cx, cy, r, attrs) =>
  elIn("circle", Array.concat([
    ("cx", f2(cx)), ("cy", f2(cy)), ("r", f2(r)),
  ], attrs), parent)->ignore

let text = (parent, x, y, content, attrs) => {
  let t = elIn("text", Array.concat([("x", f2(x)), ("y", f2(y))], attrs), parent)
  setInnerHTML(t, content)
  t
}

// ─── Tooltip ─────────────────────────────────────────────────────────────────

type tooltip = { el: Dom.element }

let makeTooltip = () => {
  let d = createElement_("div")
  let s = getStyle(d)
  s["position"]      = "fixed"
  s["pointerEvents"] = "none"
  s["opacity"]       = "0"
  s["background"]    = "#1c2128"
  s["border"]        = "1px solid #30363d"
  s["borderRadius"]  = "6px"
  s["padding"]       = "5px 10px"
  s["fontSize"]      = "12px"
  s["color"]         = "#e6edf3"
  s["transition"]    = "opacity .15s"
  s["zIndex"]        = "9999"
  s["whiteSpace"]    = "nowrap"
  appendChild(body, d)->ignore
  { el: d }
}

let showTip = (tip, html, e: Dom.mouseEvent) => {
  setInnerHTML(tip.el, html)
  let s = getStyle(tip.el)
  s["opacity"] = "1"
  s["left"]    = `${Int.toString(clientX(e) + 12)}px`
  s["top"]     = `${Int.toString(clientY(e) - 28)}px`
}

let moveTip = (tip, e: Dom.mouseEvent) => {
  let s = getStyle(tip.el)
  s["left"] = `${Int.toString(clientX(e) + 12)}px`
  s["top"]  = `${Int.toString(clientY(e) - 28)}px`
}

let hideTip = tip => {
  let s = getStyle(tip.el)
  s["opacity"] = "0"
}

let removeTip = tip => remove(tip.el)

// ─── Scale helpers ────────────────────────────────────────────────────────────

let niceMax = (vals: array<float>) => {
  let max = Array.reduce(vals, 0., (acc, v) => v > acc ? v : acc)
  if max == 0. {
    10.
  } else {
    let mag = 10. ** Float.fromInt(Float.toInt(Math.log10(max)))
    Math.ceil(max / mag) * mag
  }
}

let linspace = (min: float, max: float, n: int) =>
  Array.fromInitializer(~length=n, i =>
    min + (max - min) * Float.fromInt(i) / Float.fromInt(n - 1))

let fmtVal = (v: float) =>
  if Math.abs(v) >= 1_000_000. {
    `${Float.toFixed(v / 1_000_000., ~digits=1)}M`
  } else if Math.abs(v) >= 1_000. {
    `${Float.toFixed(v / 1_000., ~digits=1)}K`
  } else if v == Float.fromInt(Float.toInt(v)) {
    Int.toString(Float.toInt(v))
  } else {
    Float.toFixed(v, ~digits=2)
  }
