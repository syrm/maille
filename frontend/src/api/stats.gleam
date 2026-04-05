import gleam/dynamic/decode.{type Decoder}
import lustre/effect.{type Effect}
import rsvp

import model

pub fn fetch(on_response handle_response: fn(Result(model.Stats, rsvp.Error)) -> msg) -> Effect(msg) {
  let url = "http://localhost:13000/stats"

  let decoder = stats_decoder()
  let handler = rsvp.expect_json(decoder, handle_response)

  rsvp.get(url, handler)
}

fn stats_decoder() -> Decoder(model.Stats) {
  use total_transaction <- decode.field("TotalTransaction", decode.int)

  decode.success(model.Stats(total_transaction:))
}
