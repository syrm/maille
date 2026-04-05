import lustre
import lustre/effect
import model.{type Model, Model, Stats}
import update
import view

fn init(_flags) -> #(Model, effect.Effect(update.Msg)) {
  let m = Model(Stats(total_transaction: 0), "")

  #(m, effect.none())
}


pub fn main() {
  let app = lustre.application(init, update.update, view.view)
  let assert Ok(_) = lustre.start(app, "#app", Nil)
}
