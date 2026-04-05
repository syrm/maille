// Demo.res

let months = ["Jan","Fév","Mar","Avr","Mai","Juin","Juil","Août","Sep","Oct","Nov","Déc"]

let walk = (n, base, vol) => {
  let v = ref(base)
  Array.fromInitializer(~length=n, _ => {
    v := v.contents + (Math.random() - 0.49) * vol
    Float.fromInt(Float.toInt(v.contents * 100.)) / 100.
  })
}

// Binding DOMContentLoaded pour être sûr que le DOM est prêt
@val @scope("window")
external addEventListener: (string, unit => unit) => unit = "addEventListener"

let init = () => {
  let el = Core.getElementById

  // ── Bar
  let _bar = Bar.make(
    el("bar"),
    [
      { name: "Produit A", data: [42.,58.,71.,63.,89.,74.,95.,82.,68.,77.,91.,105.], color: None },
      { name: "Produit B", data: [28.,35.,41.,55.,48.,62.,58.,71.,83.,69.,74.,88.],  color: None },
      { name: "Produit C", data: [15.,22.,18.,31.,27.,34.,29.,38.,42.,35.,41.,55.],  color: None },
    ],
    { width: Some(760.), height: Some(260.), labels: Some(months),
      yTicks: None, gap: None, colors: None }
  )

  // ── Line area
  let _line = Line.make(
    el("line"),
    [{ name: "EUR/USD", data: walk(12, 1.085, 0.008), color: None, area: true, dashed: false }],
    { width: Some(370.), height: Some(220.), labels: Some(months),
      yTicks: None, smooth: true, points: true, colors: None }
  )

  // ── Pie
  let _pie = Pie.make(
    el("pie"),
    [
      { label: "Infrastructure", value: 34., color: None },
      { label: "Marketing",      value: 22., color: None },
      { label: "R&D",            value: 28., color: None },
      { label: "RH",             value: 10., color: None },
      { label: "Divers",         value:  6., color: None },
    ],
    { width: Some(370.), height: Some(220.), donut: false, donutWidth: None,
      padAngle: None, showLabel: true, colors: None }
  )

  // ── Multi line
  let _multi = Line.make(
    el("multi"),
    [
      { name: "BTC/1k",  data: walk(12, 42., 3.),  color: None, area: false, dashed: false },
      { name: "ETH/100", data: walk(12, 22., 2.),  color: None, area: false, dashed: true  },
      { name: "SOL",     data: walk(12, 95., 7.),  color: None, area: false, dashed: true  },
    ],
    { width: Some(370.), height: Some(220.), labels: Some(months),
      yTicks: None, smooth: true, points: false, colors: None }
  )

  // ── Donut
  let _donut = Pie.make(
    el("donut"),
    [
      { label: "Complété",   value: 63., color: None },
      { label: "En cours",   value: 20., color: None },
      { label: "En attente", value: 17., color: None },
    ],
    { width: Some(370.), height: Some(220.), donut: true, donutWidth: Some(45.),
      padAngle: None, showLabel: true, colors: None }
  )

  ignore((_bar, _line, _pie, _multi, _donut))
}

addEventListener("DOMContentLoaded", init)
