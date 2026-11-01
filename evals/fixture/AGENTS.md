# Rules for this repo

- A service function that cannot find a record raises `app.errors.NotFound`; it never returns `None`.
- Tests use `unittest` and live in `tests/`, one file per module. Run them with `python3 -m unittest`.
