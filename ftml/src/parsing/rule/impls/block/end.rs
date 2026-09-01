/*
 * parsing/rule/impls/block/end.rs
 *
 * ftml - Library to parse Wikidot text
 * Copyright (C) 2019-2022 Wikijump Team
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <http://www.gnu.org/licenses/>.
 */

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
