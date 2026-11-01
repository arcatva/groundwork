# Support

What the change needs from around the code: migrations, dependencies, configuration, build.

**Rule.** A migration follows the repo's migration rules, keeps the migration history linear, holds
no lock longer than the change needs, and runs on the data production holds. A new dependency,
setting, environment variable, feature flag, build entry or job type is added in each place the repo
declares one, with the value each environment needs.

**Gate.** For each migration, setting and dependency the diff adds, name every environment it
reaches and what it does there on deploy.