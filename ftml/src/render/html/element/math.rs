/*
 * render/html/element/math.rs
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
use cfg_if::cfg_if;
use std::num::NonZeroUsize;

cfg_if! {
    if #[cfg(feature = "mathml")] {
        use latex2mathml::{latex_to_mathml, DisplayStyle};

        const MAX_LATEX_BYTES: usize = 8 * 1024;
        const MAX_NESTING_ON_CALLER_STACK: usize = 128;
        const CONVERTER_STACK_BYTES: usize = 64 * 1024 * 1024;

        // Running out of stack takes the whole process down, so a formula that could
        // nest deeply is converted on a thread with room for the longest source allowed.
        fn convert(latex_source: &str, display: DisplayStyle) -> Result<String, Option<String>> {
            if latex_source.len() > MAX_LATEX_BYTES {
                return Err(None);
            }
            let nesting = latex_source
                .bytes()
                .filter(|byte| matches!(byte, b'{' | b'^' | b'_' | b'\\'))
                .count();
            if nesting <= MAX_NESTING_ON_CALLER_STACK {
                return latex_to_mathml(latex_source, display).map_err(|error| Some(str!(error)));
            }

            let source = str!(latex_source);
            std::thread::Builder::new()
                .stack_size(CONVERTER_STACK_BYTES)
                .spawn(move || latex_to_mathml(&source, display).map_err(|error| str!(error)))
                .map_err(|_| None)?
                .join()
                .map_err(|_| None)?
                .map_err(Some)
        }
    } else {
        /// Mocked version of the enum from `latex2mathml`.
        #[derive(Debug, Copy, Clone, PartialEq, Eq)]
        enum DisplayStyle {
            Block,
            Inline,
        }
    }
}

pub fn render_math_block(ctx: &mut HtmlContext, name: Option<&str>, latex_source: &str) {
    info!(
        "Rendering math block (name '{}', source '{}')",
        name.unwrap_or("<none>"),
        latex_source,
    );

    let index = ctx.next_equation_index();
    if let Some(name) = name {
        ctx.name_equation(name, index);
    }

    render_latex(ctx, name, Some(index), latex_source, DisplayStyle::Block);
}

pub fn render_math_inline(ctx: &mut HtmlContext, latex_source: &str) {
    info!("Rendering math inline (source '{latex_source}'");
    render_latex(ctx, None, None, latex_source, DisplayStyle::Inline);
}

fn render_latex(
    ctx: &mut HtmlContext,
    name: Option<&str>,
    index: Option<NonZeroUsize>,
    latex_source: &str,
    display: DisplayStyle,
) {
    // error_type is unused if MathML is disabled
    let (html_tag, wj_type, _error_type) = match display {
        DisplayStyle::Block => ("div", "wj-math-block", "wj-error-block"),
        DisplayStyle::Inline => ("span", "wj-math-inline", "wj-error-inline"),
    };

    // Outer container
    ctx.html()
        .tag(html_tag)
        .attr(attr!(
            "class" => "wj-math " wj_type,
            "id" => &format!("equation-{}", index.map_or(0, NonZeroUsize::get)); if index.is_some(),
            "data-name" => name.unwrap_or(""); if name.is_some(),
        ))
        .contents(|ctx| {
            // Add equation index
            if let Some(index) = index {
                ctx.html()
                    .span()
                    .attr(attr!("class" => "wj-equation-number equation-number"))
                    .contents(|ctx| {
                        // Open parenthesis
                        ctx.html()
                            .span()
                            .attr(attr!(
                                "class" => "wj-equation-paren wj-equation-paren-open",
                            ))
                            .inner("(");

                        str_write!(ctx, "{index}");

                        // Close parenthesis
                        ctx.html()
                            .span()
                            .attr(attr!(
                                "class" => "wj-equation-paren wj-equation-paren-close",
                            ))
                            .inner(")");
                    });
            }

            // Add LaTeX source (hidden)
            // Can't use a pre tag because that won't work for inline tags
            ctx.html()
                .code()
                .attr(attr!(
                    "class" => "wj-math-source wj-hidden",
                    "aria-hidden" => "true",
                    "hidden" => "",
                ))
                .inner(latex_source);

            // Add generated MathML
            cfg_if! {
                if #[cfg(feature = "mathml")] {
                    match convert(latex_source, display) {
                        Ok(mathml) => {
                            info!("Processed LaTeX -> MathML");

                            // Inject MathML elements
                            ctx.html()
                                .element("wj-math-ml")
                                .attr(attr!("class" => "wj-math-ml"))
                                .contents(|ctx| ctx.push_raw_str(&mathml));
                        }
                        Err(error) => {
                            let error = match error {
                                Some(error) => error,
                                None => ctx.handle().get_message("math-too-complex"),
                            };
                            warn!("Error processing LaTeX -> MathML: {error}");

                            ctx.html()
                                .span()
                                .attr(attr!("class" => _error_type))
                                .inner(error);
                        }
                    }
                }
            }
        });
}

pub fn render_equation_reference(ctx: &mut HtmlContext, name: &str) {
    info!("Rendering equation reference (name '{name}')");

    let slot = ctx.equation_reference_slot(name);
    ctx.push_raw_str(&slot);
}
