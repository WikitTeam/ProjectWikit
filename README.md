# ProjectWikit

A Wikidot-compatible wiki engine and migration target.

## What it does

- Imports Wikidot sites (users, pages, history, votes, forums) from backups
- Renders Wikidot syntax with output matched against real Wikidot HTML
- Wiki Farm: one instance serves many wikis
- Wikidot account claiming with external verification

## Why it exists

Wikidot is frozen. ProjectWikit exists to preserve existing sites as they are — the goal is byte-level fidelity to Wikidot's own output, so a migrated site behaves like the original rather than like a reinterpretation of it.

## Install


## Documentation


## History and credits

ProjectWikit began as a fork of [RuFoundation](...), the site software written for the Russian SCP branch. The current implementation is a ground-up Go rewrite and shares no code with it, but the project owes its existence and much of its early direction to that work.

Wikidot syntax parsing uses [FTML-WIKIT](https://github.com/WikitTeam/ProjectWikit/tree/major/ftml), a hard fork of Wikijump's [FTML](https://github.com/scpwiki/ftml). Upstream FTML focuses on the subset of Wikidot syntax considered well-formed, and as a result, some syntax constructs may not remain compatible. The fork is maintained separately and is not expected to merge back. Our goal is to achieve exact compatibility with Wikidot's output.