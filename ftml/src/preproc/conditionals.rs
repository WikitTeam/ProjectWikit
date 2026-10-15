/*
 * preproc/conditionals.rs
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

use crate::data::{ExpressionResult, PageCallbacks};
use std::borrow::Cow;

pub fn if_condition_holds(condition: &str) -> bool {
    let condition = condition.trim().to_ascii_lowercase();
    !(condition.is_empty()
        || condition == "0"
        || condition == "false"
        || condition == "null"
        || (condition.starts_with("{$") && condition.ends_with('}'))
        || (condition.starts_with("%%") && condition.ends_with("%%")))
}

pub fn expression_holds(result: ExpressionResult) -> bool {
    match result {
        ExpressionResult::Bool(b) => b,
        ExpressionResult::Int(i) => i != 0,
        ExpressionResult::Float(f) => f != 0.0,
        ExpressionResult::String(s) => !s.is_empty(),
        ExpressionResult::None => false,
    }
}

pub fn expand_head_conditionals(text: &mut String, callbacks: &dyn PageCallbacks) {
    if !text.contains("[[#if") && !text.contains("[[#IF") && !text.contains("[[#If") {
        return;
    }

    let source = std::mem::take(text);
    let mut out = String::with_capacity(source.len());
    let mut rest = source.as_str();

    while !rest.is_empty() {
        if rest.starts_with("@@") {
            let end = rest[2..].find("@@").map(|i| i + 4).unwrap_or(rest.len());
            out.push_str(&rest[..end]);
            rest = &rest[end..];
            continue;
        }
        if rest.starts_with("[[[") {
            out.push_str("[[[");
            rest = &rest[3..];
            continue;
        }
        if rest.starts_with("[[") && starts_head(&rest[2..]) {
            if block_name(&rest[2..]).eq_ignore_ascii_case("code") {
                let end = find_ignore_case(rest, "[[/code]]")
                    .map(|i| i + "[[/code]]".len())
                    .unwrap_or(rest.len());
                out.push_str(&rest[..end]);
                rest = &rest[end..];
                continue;
            }
            match head_end(rest) {
                Some(end) => {
                    out.push_str(&expand_within(&rest[..end], callbacks));
                    rest = &rest[end..];
                }
                None => {
                    out.push_str(rest);
                    rest = "";
                }
            }
            continue;
        }
        let step = rest.chars().next().map(char::len_utf8).unwrap_or(1);
        out.push_str(&rest[..step]);
        rest = &rest[step..];
    }

    *text = out;
}

fn starts_head(after: &str) -> bool {
    after
        .chars()
        .next()
        .map(|c| c.is_ascii_alphabetic() || c == '*' || c == '=' || c == '<' || c == '>')
        .unwrap_or(false)
}

fn block_name(after: &str) -> &str {
    let end = after
        .find(|c: char| c.is_whitespace() || c == ']')
        .unwrap_or(after.len());
    &after[..end]
}

fn find_ignore_case(haystack: &str, needle: &str) -> Option<usize> {
    haystack.to_ascii_lowercase().find(&needle.to_ascii_lowercase())
}

fn head_end(s: &str) -> Option<usize> {
    let bytes = s.as_bytes();
    let mut depth = 0usize;
    let mut i = 0;
    while i + 1 < bytes.len() {
        if bytes[i] == b'[' && bytes[i + 1] == b'[' {
            depth += 1;
            i += 2;
            continue;
        }
        if bytes[i] == b']' && bytes[i + 1] == b']' {
            depth -= 1;
            i += 2;
            if depth == 0 {
                return Some(i);
            }
            continue;
        }
        if bytes[i] == b'\n' && depth == 1 {
            return None;
        }
        i += 1;
    }
    None
}

fn conditional_at(s: &str) -> Option<bool> {
    let lower = s.get(..9).unwrap_or(s).to_ascii_lowercase();
    if lower.starts_with("[[#ifexpr") {
        Some(true)
    } else if lower.starts_with("[[#if") && s[5..].starts_with(|c: char| c.is_whitespace()) {
        Some(false)
    } else {
        None
    }
}

fn expand_within(head: &str, callbacks: &dyn PageCallbacks) -> String {
    let mut out = String::with_capacity(head.len());
    let mut rest = &head[2..];
    out.push_str("[[");

    while !rest.is_empty() {
        if let Some(is_expr) = conditional_at(rest) {
            if let Some(end) = head_end(rest) {
                let inner = &rest[if is_expr { 9 } else { 5 }..end - 2];
                let parts = split_top_level(inner);
                let holds = match is_expr {
                    true => expression_holds(callbacks.evaluate_expression(Cow::from(parts[0].trim()))),
                    false => if_condition_holds(parts[0]),
                };
                let chosen = match holds {
                    true => parts.get(1).copied().unwrap_or(""),
                    false => parts.get(2).copied().unwrap_or(""),
                };
                let expanded = expand_within(&format!("[[{}", chosen.trim()), callbacks);
                out.push_str(&expanded[2..]);
                rest = &rest[end..];
                continue;
            }
        }
        let step = rest.chars().next().map(char::len_utf8).unwrap_or(1);
        out.push_str(&rest[..step]);
        rest = &rest[step..];
    }
    out
}

fn split_top_level(inner: &str) -> Vec<&str> {
    let bytes = inner.as_bytes();
    let mut parts = Vec::new();
    let mut depth = 0usize;
    let mut start = 0;
    let mut i = 0;
    while i < bytes.len() {
        if i + 1 < bytes.len() && bytes[i] == b'[' && bytes[i + 1] == b'[' {
            depth += 1;
            i += 2;
            continue;
        }
        if i + 1 < bytes.len() && bytes[i] == b']' && bytes[i + 1] == b']' && depth > 0 {
            depth -= 1;
            i += 2;
            continue;
        }
        if bytes[i] == b'|' && depth == 0 {
            parts.push(&inner[start..i]);
            start = i + 1;
        }
        i += 1;
    }
    parts.push(&inner[start..]);
    parts
}

#[cfg(test)]
mod test {
    use super::*;
    use crate::data::NullPageCallbacks;

    fn expand(input: &str) -> String {
        let mut text = str!(input);
        expand_head_conditionals(&mut text, &NullPageCallbacks {});
        text
    }

    #[test]
    fn conditions() {
        assert!(if_condition_holds("1"));
        assert!(if_condition_holds("yes"));
        assert!(!if_condition_holds("0"));
        assert!(!if_condition_holds(" "));
        assert!(!if_condition_holds("False"));
        assert!(!if_condition_holds("{$missing}"));
    }

    #[test]
    fn heads() {
        assert_eq!(
            expand(r#"[[li class="[[#if 1 | folded | unfolded ]] [[#if 0 | alt ]]"]]x"#),
            r#"[[li class="folded "]]x"#,
        );
        assert_eq!(
            expand(r#"[[div [[#if 1 |  | class="end" ]] ]]"#),
            "[[div  ]]",
        );
        assert_eq!(
            expand(r#"[[div [[#if 0 |  | class="end" ]] ]]"#),
            r#"[[div class="end" ]]"#,
        );
        assert_eq!(
            expand(r#"[[span class="[[#if 1 | [[#if 0 | a | b ]] | c ]]"]]"#),
            r#"[[span class="b"]]"#,
        );
    }

    #[test]
    fn leaves_body_and_raw_text() {
        let body = "[# a][[#if 0 | | [# b] ]]";
        assert_eq!(expand(body), body);
        let raw = r#"@@[[div class="[[#if 1 | a | b ]]"]]@@"#;
        assert_eq!(expand(raw), raw);
        let code = "[[code]]\n[[div class=\"[[#if 1 | a | b ]]\"]]\n[[/code]]";
        assert_eq!(expand(code), code);
    }
}
