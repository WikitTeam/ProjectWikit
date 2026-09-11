/*
 * parsing/rule/impls/block/blocks/math.rs
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

use super::prelude::*;

pub const BLOCK_MATH: BlockRule = BlockRule {
    name: "block-math",
    accepts_names: &["math"],
    accepts_star: false,
    accepts_score: false,
    accepts_newlines: true,
    accepts_partial: AcceptsPartial::None,
    parse_fn,
};

pub const BLOCK_EQUATION_REFERENCE: BlockRule = BlockRule {
    name: "block-equation-reference",
    accepts_names: &["eref"],
    accepts_star: false,
    accepts_score: false,
    accepts_newlines: false,
    accepts_partial: AcceptsPartial::None,
    parse_fn: parse_equation_reference,
};

fn parse_equation_reference<'r, 't>(
    parser: &mut Parser<'r, 't>,
    name: &'t str,
    flag_star: bool,
    flag_score: bool,
    in_head: bool,
) -> ParseResult<'r, 't, Elements<'t>> {
    info!("Parsing equation reference block (name '{name}', in-head {in_head})");
    assert!(!flag_star, "Equation reference doesn't allow star flag");
    assert!(!flag_score, "Equation reference doesn't allow score flag");
    assert_block_name(&BLOCK_EQUATION_REFERENCE, name);

    let label = parser.get_head_value(
        &BLOCK_EQUATION_REFERENCE,
        in_head,
        |parser, value| match value {
            Some(label) if !label.trim().is_empty() => Ok(label.trim()),
            _ => Err(parser.make_warn(ParseWarningKind::BlockMissingArguments)),
        },
    )?;

    ok!(Element::EquationReference(cow!(label)))
}

fn parse_fn<'r, 't>(
    parser: &mut Parser<'r, 't>,
    name: &'t str,
    flag_star: bool,
    flag_score: bool,
    in_head: bool,
) -> ParseResult<'r, 't, Elements<'t>> {
    info!("Parsing math block (name '{name}', in-head {in_head})");
    assert!(!flag_star, "User doesn't allow star flag");
    assert!(!flag_score, "User doesn't allow score flag");
    assert_block_name(&BLOCK_MATH, name);

    let block_name = name;

    let name = parser.get_head_value(&BLOCK_MATH, in_head, |_, value| {
        Ok(value.map(|s| cow!(s.trim())))
    })?;

    let latex_source = parser.get_body_text(&BLOCK_MATH, block_name)?.trim();
    if latex_source.is_empty() {
        return Err(parser.make_warn(ParseWarningKind::RuleFailed));
    }

    let element = Element::Math {
        name,
        latex_source: cow!(latex_source),
    };

    ok!(element)
}
