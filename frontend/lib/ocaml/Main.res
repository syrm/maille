let months = ["Jan","Fév","Mar","Avr","Mai","Juin","Juil","Août","Sep","Oct","Nov","Déc"]

@val external addEventListener: (string, unit => unit) => unit = "addEventListener"
@get external textContent: Dom.element => Nullable.t<string> = "textContent"

let init = () => {
  let getEl = id => document->Core.getElementById(id)->Nullable.toOption

  let appData: option<JSON.t> =
    getEl("app-data")
    ->Option.flatMap(el => el->textContent->Nullable.toOption)
    ->Option.flatMap(s =>
        try Some(JSON.parseOrThrow(s)) catch { | _ => None }
      )

  switch (appData, getEl("net-worth-history-chart")) {
  | (Some(json), Some(el)) =>
    let chartData = Line.parseChart(json)

    let cleanup = Line.make(
      el,
      chartData.series,
      chartData.options
    )
  | _ => ()
  }
}

addEventListener("DOMContentLoaded", init)
