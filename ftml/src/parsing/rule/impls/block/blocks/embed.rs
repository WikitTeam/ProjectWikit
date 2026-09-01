//! Rule for a block holding one iframe.
//!
//! The frame goes straight into the page rather than into a sandbox of its
//! own, so a site stylesheet can give an embedded player its size. Only an
//! iframe is allowed in, because anything else would be running in the page.

use std::borrow::Cow;
use std::collections::HashMap;

use regex::Regex;
use unicase::UniCase;

use super::prelude::*;
use crate::tree::AttributeMap;

pub const BLOCK_EMBED: BlockRule = BlockRule {
    name: "block-embed",
    accepts_names: &["embed"],
    accepts_star: false,
    accepts_score: false,
    accepts_newlines: true,
    accepts_partial: AcceptsPartial::None,
    parse_fn,
};

lazy_static! {
    static ref IFRAME_TAG: Regex =
        Regex::new(r"(?is)^\s*<iframe\b([^>]*?)/?>\s*(?:</iframe>\s*)?$").unwrap();
    static ref ATTRIBUTE: Regex =
        Regex::new("(?is)([a-z_:][a-z0-9_:.-]*)\\s*(?:=\\s*(?:\"([^\"]*)\"|'([^']*)'|([^\\s\"'>]+)))?")
            .unwrap();
}

fn parse_fn<'r, 't>(
    parser: &mut Parser<'r, 't>,
    name: &'t str,
    flag_star: bool,
    flag_score: bool,
    in_head: bool,
) -> ParseResult<'r, 't, Elements<'t>> {
    info!("Parsing embed block (in-head {in_head})");
    assert!(!flag_star, "Embed doesn't allow star flag");
    assert!(!flag_score, "Embed doesn't allow score flag");
    assert_block_name(&BLOCK_EMBED, name);

    parser.get_head_none(&BLOCK_EMBED, in_head)?;
    let body = parser.get_body_text(&BLOCK_EMBED, name)?;

    let captures = match IFRAME_TAG.captures(body) {
        Some(captures) => captures,
        None => return Err(parser.make_warn(ParseWarningKind::RuleFailed)),
    };

    let mut arguments: HashMap<UniCase<&'t str>, Cow<'t, str>> = HashMap::new();
    let mut url = "";
    for capture in ATTRIBUTE.captures_iter(captures.get(1).unwrap().as_str()) {
        let key = capture.get(1).unwrap().as_str();
        let value = capture
            .get(2)
            .or_else(|| capture.get(3))
            .or_else(|| capture.get(4))
            .map(|m| m.as_str())
            .unwrap_or("");

        if key.eq_ignore_ascii_case("src") {
            url = value;
            continue;
        }
        arguments.insert(UniCase::ascii(key), cow!(value));
    }

    // An embedded frame runs in the page, so its source is held to an allowlist
    // rather than validate_href's blocklist, which lets vbscript and the like
    // through. Only the schemes a real embed uses are accepted.
    if !embeddable_src(url) {
        return Err(parser.make_warn(ParseWarningKind::RuleFailed));
    }

    let element = Element::Iframe {
        url: cow!(url),
        attributes: AttributeMap::from_arguments(&arguments),
    };

    ok!(element)
}

fn embeddable_src(url: &str) -> bool {
    let lowered = url.trim().to_ascii_lowercase();
    lowered.starts_with("https://") || lowered.starts_with("http://") || lowered.starts_with("//")
}
