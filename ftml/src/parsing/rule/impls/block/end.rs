//! Rule for an end block that closes nothing.
//!
//! A block that is open consumes its own end block while gathering its body,
//! so anything reaching this rule was never opened. Wikidot drops such a tag,
//! and pages carrying one have been written against that for years.

use super::super::prelude::*;

pub const RULE_BLOCK_END: Rule = Rule {
    name: "block-end",
    position: LineRequirement::Any,
    try_consume_fn,
};

fn try_consume_fn<'p, 'r, 't>(
    parser: &'p mut Parser<'r, 't>,
) -> ParseResult<'r, 't, Elements<'t>> {
    info!("Consuming an end block that closes nothing");

    // A malformed tag is left to the fallback so it still shows up as text.
    parser.get_end_block()?;

    ok!(Elements::None)
}
