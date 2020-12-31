# URY Time Machine

<img alt="The Time Machine Show logo" src="https://ury.org.uk/media/image_meta/ShowImageMetadata/288.jpeg" width="150"/>

This is The Time Machine, named after a (pretty good) show on URY! What does it do? It let's listeners rewind time, so they can listen back to an earlier part of the on air programme.

## Running It

To install the bits you need, `go get`. Usual folder structures of Go apply. You'll then want to make sure you've setup https://github.com/UniversityRadioYork/myradio-go before continuing. (Basically, create a file called `.myradio.key` containing an API key.)

To build the Time Machine, run `go build`.

You can then run the server with `./TimeMachine`.

The UI is served from `/` on the server, by default on port `3958`.
