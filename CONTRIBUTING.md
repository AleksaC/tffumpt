# Contributing

All contributions are welcome! There's no official contribution process. For small
changes you can open a pull request directly. For bigger changes and new features
open an issue to discuss it before doing any work, so you don't end up wasting your
time in case it doesn't get accepted.

Before a pull request gets merged it needs to pass all the checks, including tests,
lint and pre-commit. Please make sure that's the case before requesting a review.

If you'd like to contribute, but don't know where to start take a look at the [roadmap](./README.md#roadmap).
There isn't a clear direction at the moment, but there are some ideas that can serve
as a starting point for further development.

## Development Setup

The only requirement for hacking on this project is to have go toolchain installed.
Instructions on how to set it up are available on multiple websites or you can ask
your favorite LLM.

Additionally it's also recommended to use [pre-commit](https://pre-commit.com/)
to run other checks, such as linters and formatters. Note that you don't have to
use pre-commit in actual pre-commit hooks, instead its main purpose is to install
all the relevant tools and run them using a single command.

Provided that you have set up the tools mentioned above, you can run all the checks
using a single command:

```
make check
```
