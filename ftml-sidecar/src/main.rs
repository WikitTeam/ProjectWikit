use std::borrow::Cow;
use std::cell::RefCell;
use std::collections::HashMap;
use std::fmt::{self, Debug, Formatter};
use std::io::{self, BufReader, Read, Stdin, Stdout, Write};
use std::rc::Rc;

use ftml::data::{ExpressionResult, PageCallbacks, PageInfo, PageRef, PartialPageInfo};
use ftml::includes::{FetchedPage, IncludeRef, Includer};
use ftml::prelude::remove_noincludes;
use ftml::render::html::HtmlRender;
use ftml::render::text::TextRender;
use ftml::render::Render;
use ftml::settings::{WikitextMode, WikitextSettings};
use ftml::{include, parse, preprocess, tokenize};
use serde_json::{json, Map, Value};

fn write_msg<W: Write>(w: &mut W, v: &Value) -> io::Result<()> {
    let buf = serde_json::to_vec(v).map_err(io::Error::other)?;
    let len = u32::try_from(buf.len()).map_err(io::Error::other)?;
    w.write_all(&len.to_be_bytes())?;
    w.write_all(&buf)?;
    w.flush()
}

fn read_msg<R: Read>(r: &mut R) -> io::Result<Option<Value>> {
    let mut len = [0u8; 4];
    match r.read_exact(&mut len) {
        Ok(()) => {}
        Err(e) if e.kind() == io::ErrorKind::UnexpectedEof => return Ok(None),
        Err(e) => return Err(e),
    }
    let mut buf = vec![0u8; u32::from_be_bytes(len) as usize];
    r.read_exact(&mut buf)?;
    serde_json::from_slice(&buf)
        .map(Some)
        .map_err(io::Error::other)
}

struct Bridge {
    input: RefCell<BufReader<Stdin>>,
    output: RefCell<Stdout>,
}

impl Debug for Bridge {
    fn fmt(&self, f: &mut Formatter<'_>) -> fmt::Result {
        write!(f, "<StdioBridge>")
    }
}

impl Bridge {
    fn new() -> Self {
        Self {
            input: RefCell::new(BufReader::new(io::stdin())),
            output: RefCell::new(io::stdout()),
        }
    }

    fn recv(&self) -> Option<Value> {
        read_msg(&mut *self.input.borrow_mut()).expect("read inbound message")
    }

    fn send(&self, v: &Value) {
        write_msg(&mut *self.output.borrow_mut(), v).expect("write outbound message");
    }

    fn call(&self, method: &str, args: Value) -> Value {
        self.send(&json!({"type": "callback", "method": method, "args": args}));
        let reply = self.recv().expect("host closed the connection before answering the callback");
        match reply.get("type").and_then(Value::as_str) {
            Some("callback_result") => reply.get("value").cloned().unwrap_or(Value::Null),
            other => panic!("callback {method} got type = {other:?}, want callback_result"),
        }
    }

    fn call_str(&self, method: &str, args: Value) -> Cow<'static, str> {
        Cow::Owned(
            self.call(method, args)
                .as_str()
                .unwrap_or_default()
                .to_owned(),
        )
    }

    fn next_include_level(&self) {
        self.call("next_include_level", json!({}));
    }
}

impl PageCallbacks for Bridge {
    fn module_has_body(&self, module_name: Cow<str>) -> bool {
        self.call("module_has_body", json!({ "name": module_name }))
            .as_bool()
            .unwrap_or(false)
    }

    fn module_is_inline(&self, module_name: Cow<str>) -> bool {
        self.call("module_is_inline", json!({ "name": module_name }))
            .as_bool()
            .unwrap_or(false)
    }

    fn render_module(
        &self,
        module_name: Cow<str>,
        params: HashMap<Cow<str>, Cow<str>>,
        body: Cow<str>,
    ) -> Cow<'static, str> {
        let params: Map<String, Value> = params
            .iter()
            .map(|(k, v)| (k.to_string(), Value::String(v.to_string())))
            .collect();
        self.call_str(
            "render_module",
            json!({"name": module_name, "params": params, "body": body}),
        )
    }

    fn render_user(&self, user: Cow<str>, avatar: bool) -> Cow<'static, str> {
        self.call_str("render_user", json!({"user": user, "avatar": avatar}))
    }

    fn get_i18n_message(&self, message_id: Cow<str>) -> Cow<'static, str> {
        self.call_str("get_i18n_message", json!({ "id": message_id }))
    }

    fn get_html_injected_code(&self, html_id: Cow<str>) -> Cow<'static, str> {
        self.call_str("get_html_injected_code", json!({ "id": html_id }))
    }

    fn get_page_info(&self, page_refs: &Vec<PageRef<'_>>) -> Vec<PartialPageInfo<'static>> {
        let names: Vec<String> = page_refs.iter().map(PageRef::to_string).collect();
        let reply = self.call("get_page_info", json!({ "refs": names }));

        let mut out = Vec::new();
        for item in reply.as_array().map(Vec::as_slice).unwrap_or_default() {
            let full_name = item.get("full_name").and_then(Value::as_str).unwrap_or("");
            let Some(idx) = names.iter().position(|n| n == full_name) else {
                continue;
            };
            out.push(PartialPageInfo {
                page_ref: page_refs[idx].to_owned(),
                title: item
                    .get("title")
                    .and_then(Value::as_str)
                    .map(|s| Cow::Owned(s.to_owned())),
                exists: item.get("exists").and_then(Value::as_bool).unwrap_or(false),
            });
        }
        out
    }

    fn evaluate_expression(&self, expression: Cow<str>) -> ExpressionResult<'static> {
        let reply = self.call("evaluate_expression", json!({ "expr": expression }));
        match reply.get("kind").and_then(Value::as_str) {
            Some("string") => ExpressionResult::String(Cow::Owned(
                reply
                    .get("str")
                    .and_then(Value::as_str)
                    .unwrap_or_default()
                    .to_owned(),
            )),
            Some("bool") => {
                ExpressionResult::Bool(reply.get("bool").and_then(Value::as_bool).unwrap_or(false))
            }
            Some("float") => {
                ExpressionResult::Float(reply.get("float").and_then(Value::as_f64).unwrap_or(0.0))
            }
            Some("int") => {
                ExpressionResult::Int(reply.get("int").and_then(Value::as_i64).unwrap_or(0))
            }
            _ => ExpressionResult::None,
        }
    }

    fn normalize_page_name(&self, full_name: Cow<str>) -> Cow<'static, str> {
        self.call_str("normalize_page_name", json!({ "full_name": full_name }))
    }
}

struct BridgeIncluder<'b>(&'b Bridge);

impl<'t, 'b> Includer<'t> for BridgeIncluder<'b> {
    type Error = io::Error;

    fn include_pages(
        &mut self,
        includes: &[IncludeRef<'t>],
    ) -> Result<Vec<FetchedPage<'t>>, Self::Error> {
        let refs: Vec<Value> = includes
            .iter()
            .map(|inc| {
                let vars: Map<String, Value> = inc
                    .variables()
                    .iter()
                    .map(|(k, v)| (k.to_string(), Value::String(v.to_string())))
                    .collect();
                json!({"full_name": inc.page_ref().to_string(), "variables": vars})
            })
            .collect();

        let reply = self.0.call("include_pages", json!({ "includes": refs }));
        let items = reply.as_array().map(Vec::as_slice).unwrap_or_default();

        let mut out = Vec::with_capacity(includes.len());
        for inc in includes {
            let name = inc.page_ref().to_string();
            let found = items
                .iter()
                .find(|it| it.get("full_name").and_then(Value::as_str) == Some(name.as_str()));
            out.push(FetchedPage {
                page_ref: inc.page_ref().clone(),
                content: found
                    .and_then(|it| it.get("content"))
                    .and_then(Value::as_str)
                    .map(|s| Cow::Owned(s.to_owned())),
            });
        }
        Ok(out)
    }

    fn no_such_include(&mut self, page_ref: &PageRef<'t>) -> Result<Cow<'t, str>, Self::Error> {
        Ok(Cow::Owned(
            self.0
                .call_str("no_such_include", json!({"full_name": page_ref.to_string()}))
                .into_owned(),
        ))
    }
}

fn page_info_from(v: &Value) -> PageInfo<'static> {
    let s = |k: &str| -> String {
        v.get(k)
            .and_then(Value::as_str)
            .unwrap_or_default()
            .to_owned()
    };
    let opt = |k: &str| -> Option<String> {
        v.get(k)
            .and_then(Value::as_str)
            .filter(|x| !x.is_empty())
            .map(str::to_owned)
    };

    let domain = s("domain");
    let page = s("page");
    let site = opt("site")
        .unwrap_or_else(|| domain.split('.').next().unwrap_or_default().to_owned());

    PageInfo {
        page: Cow::Owned(page.clone()),
        category: Some(Cow::Owned(s("category"))),
        site: Cow::Owned(site),
        domain: Cow::Owned(domain.clone()),
        media_domain: Cow::Owned(opt("media_domain").unwrap_or(domain)),
        title: Cow::Owned(opt("title").unwrap_or(page)),
        alt_title: None,
        rating: v.get("rating").and_then(Value::as_f64).unwrap_or(0.0),
        tags: v
            .get("tags")
            .and_then(Value::as_array)
            .map(|a| {
                a.iter()
                    .filter_map(Value::as_str)
                    .map(|t| Cow::Owned(t.to_owned()))
                    .collect()
            })
            .unwrap_or_default(),
        language: Cow::Owned(opt("language").unwrap_or_else(|| "default".to_owned())),
    }
}

fn wikitext_mode(mode: &str) -> WikitextMode {
    match mode {
        "article" => WikitextMode::Page,
        "message" => WikitextMode::ForumPost,
        "inline" => WikitextMode::Inline,
        "system" => WikitextMode::System,
        "system-with-modules" => WikitextMode::SystemWithModules,
        _ => WikitextMode::Page,
    }
}

struct Rendered<O> {
    output: O,
    included_pages: Vec<String>,
    linked_pages: Vec<String>,
    code: Vec<(String, String)>,
    html: Vec<String>,
}

fn drive<R: Render>(
    source: &str,
    renderer: &R,
    info: &PageInfo,
    bridge: Rc<Bridge>,
    mode: WikitextMode,
) -> Rendered<R::Output> {
    let mut settings = WikitextSettings::from_mode(mode);
    settings.use_include_compatibility = true;

    let mut included_text = source.to_owned();
    let mut included_pages: Vec<String> = Vec::new();
    loop {
        bridge.next_include_level();
        let mut current = included_text;
        preprocess(&mut current);
        let allowed = remove_noincludes(&current);
        let (text, pages) = include(&allowed, &settings, BridgeIncluder(&bridge), || {
            panic!("Bad includer return")
        })
        .unwrap_or((current.clone(), vec![]));
        included_text = text;
        included_pages.extend(pages.iter().map(PageRef::to_string));
        if pages.is_empty() {
            break;
        }
    }

    let mut text = included_text.clone();
    let tokens = tokenize(&mut text);
    let (tree, _warnings) = parse(&tokens, info, bridge.clone(), &settings).into();
    let output = renderer.render(&tree, info, bridge, &settings);

    Rendered {
        linked_pages: tree.internal_links.iter().map(PageRef::to_string).collect(),
        code: tree.code.clone(),
        html: tree.html.clone(),
        output,
        included_pages,
    }
}

fn main() {
    let bridge = Rc::new(Bridge::new());

    while let Some(msg) = bridge.recv() {
        if msg.get("type").and_then(Value::as_str) != Some("render") {
            bridge.send(&json!({"type": "error", "message": "want type = render"}));
            continue;
        }

        let source = msg.get("source").and_then(Value::as_str).unwrap_or_default();
        let info = page_info_from(msg.get("page_info").unwrap_or(&Value::Null));
        let mode = wikitext_mode(msg.get("mode").and_then(Value::as_str).unwrap_or("article"));
        let op = msg.get("op").and_then(Value::as_str).unwrap_or("html");

        let reply = match op {
            "text" => {
                let r = drive(source, &TextRender, &info, bridge.clone(), mode);
                json!({"type": "result", "body": r.output, "included_pages": r.included_pages,
                       "linked_pages": r.linked_pages, "code": r.code, "html": r.html})
            }
            _ => {
                let r = drive(source, &HtmlRender, &info, bridge.clone(), mode);
                json!({"type": "result", "body": r.output.body, "included_pages": r.included_pages,
                       "linked_pages": r.linked_pages, "code": r.code, "html": r.html})
            }
        };
        bridge.send(&reply);
    }
}
