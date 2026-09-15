# ProjectWikit

A Wikidot-compatible wiki engine and migration target.

ProjectWikit began as a fork of [RuFoundation](https://github.com/scpru/rufoundation),
the wiki software of the Russian SCP branch. It has since been redesigned
and rewritten in Go with its own architecture. The goal is a wiki engine
that any Wikidot community can move to without losing its pages, history,
and keep the way it works today, while gaining what Wikidot never offered
and continuing where Wikidot has stood still.

The project owes its existence and much of its early direction to
RuFoundation, and some parts are still derived from it (see [NOTICE](NOTICE)).

Wikidot syntax parsing uses [FTML-WIKIT](https://github.com/WikitTeam/ProjectWikit/tree/main/ftml), a hard fork of Wikijump's [FTML](https://github.com/scpwiki/ftml). Upstream FTML focuses on the subset of Wikidot syntax considered well-formed, and as a result, some syntax constructs do not remain compatible. The fork is maintained separately and is not expected to merge back. Our goal is to achieve exact compatibility with Wikidot's output.

## What it does

- Imports Wikidot sites (users, pages, history, votes, forums) from backups
- Renders Wikidot syntax with output matched against real Wikidot HTML
- Wiki Farm, one instance serves many wikis
- Wikidot account claiming with external verification

## Install

## Documentation


