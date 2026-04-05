import lustre/effect
import rsvp

import model.{type Stats, Model, type Model}
import api/stats

pub type Msg {
  UserClickedRefresh
  GotStats(Result(Stats, rsvp.Error))
}

pub fn update(model: Model, msg: Msg) -> #(Model, effect.Effect(Msg)) {
  case msg {
    UserClickedRefresh -> #(model, stats.fetch(GotStats))
    GotStats(Ok(stats)) -> #(Model(..model, stats:), effect.none())
    GotStats(Error(err)) -> #(Model(..model, msg: error_to_string(err)), effect.none())
  }
}

fn error_to_string(err: rsvp.Error) -> String {
  case err {
    rsvp.NetworkError -> "network error"
    rsvp.UnhandledResponse(_) -> "unhandled response"
    rsvp.JsonError(_) -> "json error"
    rsvp.HttpError(_) -> "http error"
    rsvp.BadUrl(_) -> "bad url"
    rsvp.BadBody -> "bad body"
  }
}
