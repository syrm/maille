import gleam/int
import lustre/event
import lustre/element/html.{div, h1, p, text, button}
import model.{type Model}
import update.{UserClickedRefresh}

pub fn view(model: Model) {
  div([], [
    h1([], [text("Maille !")]),
    p([], [text(int.to_string(model.stats.total_transaction)), text(" transaction")]),
    button([event.on_click(UserClickedRefresh)], [text("Refresh")])
  ])
}
