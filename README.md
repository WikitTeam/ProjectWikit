# ProjectWikit

A Wikidot-compatible wiki engine and migration target.

ProjectWikit began as a fork of [RuFoundation](https://github.com/scpru/rufoundation), the wiki software written for the Russian SCP branch. The current implementation is a ground-up Go rewrite, with the core implementation developed independently. But the project owes its existence and much of its early direction to that work.

Wikidot syntax parsing uses [FTML-WIKIT](https://github.com/WikitTeam/ProjectWikit/tree/major/ftml), a hard fork of Wikijump's [FTML](https://github.com/scpwiki/ftml). Upstream FTML focuses on the subset of Wikidot syntax considered well-formed, and as a result, some syntax constructs not remain compatible. The fork is maintained separately and is not expected to merge back. Our goal is to achieve exact compatibility with Wikidot's output.

## What it does

- Imports Wikidot sites (users, pages, history, votes, forums) from backups
- Renders Wikidot syntax with output matched against real Wikidot HTML
- Wiki Farm: one instance serves many wikis
- Wikidot account claiming with external verification

## Install

## Documentation


