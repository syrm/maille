// Currency.res

let formatElement = (el: DomBindings.element) => {
  let amount = el->DomBindings.getAttribute("data-bal-amount")->Nullable.toOption
  let currency  = el->DomBindings.getAttribute("data-bal-currency")->Nullable.toOption
  switch (amount, currency) {
  | (Some(amount), Some(currency)) =>
    let amount    = Belt.Float.fromString(amount)->Belt.Option.getWithDefault(0.0)
    let formatter = DomBindings.makeIntlNumberFormat(
        DomBindings.navigatorLanguage,
        {
        "style": "currency",
        "currency": currency,
        "maximumFractionDigits": 0,
        }
    )
    el->DomBindings.setTextContent(formatter->DomBindings.format(amount))
  | _ => ()
  }
}

let run = () =>
  DomBindings.document
  ->DomBindings.querySelectorAll("[data-bal-amount]")
  ->DomBindings.toArray
  ->Belt.Array.forEach(formatElement)
