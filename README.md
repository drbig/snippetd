# snippetd

Snippetd is a micro-service that takes POST requests and schedules them to be
printed on an ESC/POS-compatible ("receipt") thermal printer.

Main things it does:

- Wraps each snippet in a nice header and footer (can be turned off per req.)
- Does basic length sanity checks, and has text and image modes
- Queues messages for printing, in case you really want to kill your printer
- Has `expvar`ed basic statistics
- In general tries to do the least and be reliable

## Showcase

None for now, given nobody will use this anyway :P

But this is a part of a larger project, with the second element being the
[soup2esc](https://github.com/drbig/soup2esc). Been using this tandem for less
than a month but so far I'm getting what I wanted, duh.

## Contributing

Follow the usual GitHub development model:

1. Clone the repository
2. Make your changes on a separate branch
3. Make sure you run `gofmt` and `go test` before committing
4. Make a pull request

See licensing for legalese.

## Licensing

Standard two-clause BSD license, see LICENSE.txt for details.

Any contributions will be licensed under the same conditions.

Copyright (c) 2019 Piotr S. Staszewski
