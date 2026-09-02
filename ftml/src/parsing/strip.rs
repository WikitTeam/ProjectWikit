/*
 * parsing/strip.rs
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

use crate::tree::Element;

pub fn strip_newlines(elements: &mut Vec<Element>) {
    let removed_start = elements.iter().position(|x| !matches!(x, Element::LineBreak)).unwrap_or(0);
    let removed_end = elements.iter().rev().position(|x| !matches!(x, Element::LineBreak)).unwrap_or(0);

    elements.drain(0..removed_start);
    elements.drain(elements.len()-removed_end..);
}

// Wikidot writes the newline between two block elements out as plain whitespace, so
// a line break element there becomes a stray <br> and, inside a grid, a stray cell.
pub fn strip_block_newlines(elements: &mut Vec<Element>) {
    let mut i = 0;
    while i < elements.len() {
        if !matches!(elements[i], Element::LineBreak) {
            i += 1;
            continue;
        }

        let prev_block = i > 0 && is_block(&elements[i - 1]);
        let next_block = i + 1 < elements.len() && is_block(&elements[i + 1]);

        if prev_block || next_block {
            elements.remove(i);
        } else {
            i += 1;
        }
    }
}

// paragraph_safe() panics on a partial, which a block body still holds when it caught
// a stray table row or cell.
fn is_block(element: &Element) -> bool {
    match element {
        Element::Partial(_) => false,
        _ => !element.paragraph_safe(),
    }
}

pub fn strip_whitespace(elements: &mut Vec<Element>) {
    let removed_start = elements.iter().position(|x| !x.is_whitespace()).unwrap_or(0);
    let removed_end = elements.iter().rev().position(|x| !x.is_whitespace()).unwrap_or(0);

    elements.drain(0..removed_start);
    elements.drain(elements.len()-removed_end..);
}
