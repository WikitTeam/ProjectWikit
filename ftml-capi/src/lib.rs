use std::borrow::Cow;
use std::collections::HashMap;
use std::fmt::{self, Debug, Formatter};
use std::os::raw::{c_char, c_int};
use std::panic::{catch_unwind, AssertUnwindSafe};
use std::rc::Rc;
use std::slice;

use ftml::data::{ExpressionResult, PageCallbacks, PageInfo, PageRef, PartialPageInfo};
use ftml::includes::{FetchedPage, IncludeRef, Includer, NullIncluder};
use ftml::info::VERSION;
use ftml::prelude::remove_noincludes;
use ftml::render::html::HtmlRender;
use ftml::render::text::TextRender;
use ftml::render::Render;
use ftml::settings::{WikitextMode, WikitextSettings};
use ftml::{include, parse, preprocess, tokenize};

#[repr(C)]
#[derive(Clone, Copy)]
pub struct FtmlStr {
    pub ptr: *const c_char,
    pub len: usize,
}

impl FtmlStr {
    // An empty Rust str carries a dangling-but-aligned pointer, 0x1 for u8.
    // Go's collector scans this struct's pointer field and rejects 0x1, so
    // empty has to become a real null.
    fn borrow(s: &str) -> Self {
        if s.is_empty() {
            return FtmlStr::empty();
        }
        FtmlStr {
            ptr: s.as_ptr() as *const c_char,
            len: s.len(),
        }
    }

    fn empty() -> Self {
        FtmlStr {
            ptr: std::ptr::null(),
            len: 0,
        }
    }

    // Invalid UTF-8 becomes empty rather than a panic: the host is across an
    // ABI, so its bytes are input, not an invariant.
    unsafe fn as_str<'a>(&self) -> &'a str {
        if self.ptr.is_null() || self.len == 0 {
            return "";
        }
        let bytes = slice::from_raw_parts(self.ptr as *const u8, self.len);
        std::str::from_utf8(bytes).unwrap_or("")
    }
}

#[repr(C)]
#[derive(Clone, Copy)]
pub struct FtmlKeyValue {
    pub key: FtmlStr,
    pub value: FtmlStr,
}

#[repr(C)]
pub struct FtmlIncludeRef {
    pub full_name: FtmlStr,
    pub variables: *const FtmlKeyValue,
    pub variable_count: usize,
}

pub const FTML_EXPR_NONE: c_int = 0;
pub const FTML_EXPR_STRING: c_int = 1;
pub const FTML_EXPR_BOOL: c_int = 2;
pub const FTML_EXPR_FLOAT: c_int = 3;
pub const FTML_EXPR_INT: c_int = 4;

#[repr(C)]
pub struct FtmlExpressionResult {
    pub kind: c_int,
    pub int_value: i64,
    pub float_value: f64,
}

pub struct FtmlStringSink {
    value: String,
}

pub struct FtmlPageInfoSink {
    items: Vec<PartialPageInfo<'static>>,
}

pub struct FtmlFetchedPageSink {
    items: Vec<(String, Option<String>)>,
}

#[no_mangle]
pub unsafe extern "C" fn ftml_sink_string(sink: *mut FtmlStringSink, value: FtmlStr) {
    if let Some(sink) = sink.as_mut() {
        sink.value = value.as_str().to_owned();
    }
}

#[no_mangle]
pub unsafe extern "C" fn ftml_sink_page_info(
    sink: *mut FtmlPageInfoSink,
    full_name: FtmlStr,
    title: FtmlStr,
    has_title: c_int,
    exists: c_int,
) {
    let sink = match sink.as_mut() {
        Some(sink) => sink,
        None => return,
    };
    let page_ref = match PageRef::parse(full_name.as_str()) {
        Ok(page_ref) => page_ref.to_owned(),
        Err(_) => return,
    };
    sink.items.push(PartialPageInfo {
        page_ref,
        title: if has_title != 0 {
            Some(Cow::Owned(title.as_str().to_owned()))
        } else {
            None
        },
        exists: exists != 0,
    });
}

#[no_mangle]
pub unsafe extern "C" fn ftml_sink_fetched_page(
    sink: *mut FtmlFetchedPageSink,
    full_name: FtmlStr,
    content: FtmlStr,
    has_content: c_int,
) {
    if let Some(sink) = sink.as_mut() {
        sink.items.push((
            full_name.as_str().to_owned(),
            if has_content != 0 {
                Some(content.as_str().to_owned())
            } else {
                None
            },
        ));
    }
}

#[repr(C)]
pub struct FtmlCallbacks {
    pub module_has_body: Option<extern "C" fn(usize, FtmlStr) -> c_int>,
    pub module_is_inline: Option<extern "C" fn(usize, FtmlStr) -> c_int>,
    pub render_module: Option<
        extern "C" fn(usize, FtmlStr, *const FtmlKeyValue, usize, FtmlStr, *mut FtmlStringSink),
    >,
    pub render_user: Option<extern "C" fn(usize, FtmlStr, c_int, *mut FtmlStringSink)>,
    pub get_i18n_message: Option<extern "C" fn(usize, FtmlStr, *mut FtmlStringSink)>,
    pub get_html_injected_code: Option<extern "C" fn(usize, FtmlStr, *mut FtmlStringSink)>,
    pub get_page_info:
        Option<extern "C" fn(usize, *const FtmlStr, usize, *mut FtmlPageInfoSink)>,
    pub evaluate_expression: Option<
        extern "C" fn(usize, FtmlStr, *mut FtmlExpressionResult, *mut FtmlStringSink),
    >,
    pub normalize_page_name: Option<extern "C" fn(usize, FtmlStr, *mut FtmlStringSink)>,
    pub include_pages:
        Option<extern "C" fn(usize, *const FtmlIncludeRef, usize, *mut FtmlFetchedPageSink)>,
    pub no_such_include: Option<extern "C" fn(usize, FtmlStr, *mut FtmlStringSink)>,
    pub next_include_level: Option<extern "C" fn(usize) -> c_int>,
}

#[repr(C)]
pub struct FtmlPageInfo {
    pub page: FtmlStr,
    pub category: FtmlStr,
    pub site: FtmlStr,
    pub title: FtmlStr,
    pub domain: FtmlStr,
    pub media_domain: FtmlStr,
    pub rating: f64,
    pub tags: *const FtmlStr,
    pub tag_count: usize,
    pub language: FtmlStr,
}

impl FtmlPageInfo {
    unsafe fn to_page_info(&self) -> PageInfo<'static> {
        let tags = if self.tags.is_null() || self.tag_count == 0 {
            vec![]
        } else {
            slice::from_raw_parts(self.tags, self.tag_count)
                .iter()
                .map(|tag| Cow::Owned(tag.as_str().to_owned()))
                .collect()
        };
        PageInfo {
            page: Cow::Owned(self.page.as_str().to_owned()),
            category: Some(Cow::Owned(self.category.as_str().to_owned())),
            site: Cow::Owned(self.site.as_str().to_owned()),
            domain: Cow::Owned(self.domain.as_str().to_owned()),
            media_domain: Cow::Owned(self.media_domain.as_str().to_owned()),
            title: Cow::Owned(self.title.as_str().to_owned()),
            alt_title: None,
            rating: self.rating,
            tags,
            language: Cow::Owned(self.language.as_str().to_owned()),
        }
    }
}

struct HostBridge {
    callbacks: *const FtmlCallbacks,
    host: usize,
}

impl Debug for HostBridge {
    fn fmt(&self, f: &mut Formatter<'_>) -> fmt::Result {
        write!(f, "<HostBridge>")
    }
}

impl HostBridge {
    unsafe fn vtable(&self) -> &FtmlCallbacks {
        &*self.callbacks
    }

    unsafe fn collect<F>(&self, call: F) -> String
    where
        F: FnOnce(*mut FtmlStringSink),
    {
        let mut sink = FtmlStringSink {
            value: String::new(),
        };
        call(&mut sink);
        sink.value
    }

    fn next_include_level(&self) -> bool {
        unsafe {
            match self.vtable().next_include_level {
                Some(f) => f(self.host) != 0,
                None => false,
            }
        }
    }
}

impl PageCallbacks for HostBridge {
    fn module_has_body(&self, module_name: Cow<str>) -> bool {
        unsafe {
            match self.vtable().module_has_body {
                Some(f) => f(self.host, FtmlStr::borrow(&module_name)) != 0,
                None => false,
            }
        }
    }

    fn module_is_inline(&self, module_name: Cow<str>) -> bool {
        unsafe {
            match self.vtable().module_is_inline {
                Some(f) => f(self.host, FtmlStr::borrow(&module_name)) != 0,
                None => false,
            }
        }
    }

    fn render_module(
        &self,
        module_name: Cow<str>,
        params: HashMap<Cow<str>, Cow<str>>,
        body: Cow<str>,
    ) -> Cow<'static, str> {
        unsafe {
            let f = match self.vtable().render_module {
                Some(f) => f,
                None => return Cow::from(""),
            };
            let pairs: Vec<FtmlKeyValue> = params
                .iter()
                .map(|(key, value)| FtmlKeyValue {
                    key: FtmlStr::borrow(key),
                    value: FtmlStr::borrow(value),
                })
                .collect();
            Cow::Owned(self.collect(|sink| {
                f(
                    self.host,
                    FtmlStr::borrow(&module_name),
                    pairs.as_ptr(),
                    pairs.len(),
                    FtmlStr::borrow(&body),
                    sink,
                )
            }))
        }
    }

    fn render_user(&self, user: Cow<str>, avatar: bool) -> Cow<'static, str> {
        unsafe {
            let f = match self.vtable().render_user {
                Some(f) => f,
                None => return Cow::from(""),
            };
            Cow::Owned(
                self.collect(|sink| f(self.host, FtmlStr::borrow(&user), avatar as c_int, sink)),
            )
        }
    }

    fn get_i18n_message(&self, message_id: Cow<str>) -> Cow<'static, str> {
        unsafe {
            let f = match self.vtable().get_i18n_message {
                Some(f) => f,
                None => return Cow::from("?"),
            };
            Cow::Owned(self.collect(|sink| f(self.host, FtmlStr::borrow(&message_id), sink)))
        }
    }

    fn get_html_injected_code(&self, html_id: Cow<str>) -> Cow<'static, str> {
        unsafe {
            let f = match self.vtable().get_html_injected_code {
                Some(f) => f,
                None => return Cow::from("?"),
            };
            Cow::Owned(self.collect(|sink| f(self.host, FtmlStr::borrow(&html_id), sink)))
        }
    }

    fn get_page_info<'a>(&self, page_refs: &Vec<PageRef<'a>>) -> Vec<PartialPageInfo<'static>> {
        unsafe {
            let f = match self.vtable().get_page_info {
                Some(f) => f,
                None => return vec![],
            };
            let names: Vec<String> = page_refs.iter().map(|r| r.to_string()).collect();
            let borrowed: Vec<FtmlStr> = names.iter().map(|n| FtmlStr::borrow(n)).collect();
            let mut sink = FtmlPageInfoSink { items: vec![] };
            f(self.host, borrowed.as_ptr(), borrowed.len(), &mut sink);
            sink.items
        }
    }

    fn evaluate_expression(&self, expression: Cow<str>) -> ExpressionResult<'static> {
        unsafe {
            let f = match self.vtable().evaluate_expression {
                Some(f) => f,
                None => return ExpressionResult::None,
            };
            let mut out = FtmlExpressionResult {
                kind: FTML_EXPR_NONE,
                int_value: 0,
                float_value: 0.0,
            };
            let text =
                self.collect(|sink| f(self.host, FtmlStr::borrow(&expression), &mut out, sink));
            match out.kind {
                FTML_EXPR_STRING => ExpressionResult::String(Cow::Owned(text)),
                FTML_EXPR_BOOL => ExpressionResult::Bool(out.int_value != 0),
                FTML_EXPR_FLOAT => ExpressionResult::Float(out.float_value),
                FTML_EXPR_INT => ExpressionResult::Int(out.int_value),
                _ => ExpressionResult::None,
            }
        }
    }

    fn normalize_page_name(&self, full_name: Cow<str>) -> Cow<'static, str> {
        unsafe {
            let f = match self.vtable().normalize_page_name {
                Some(f) => f,
                None => return Cow::Owned(full_name.into_owned()),
            };
            Cow::Owned(self.collect(|sink| f(self.host, FtmlStr::borrow(&full_name), sink)))
        }
    }
}

struct BridgeIncluder<'b>(&'b HostBridge);

impl<'t, 'b> Includer<'t> for BridgeIncluder<'b> {
    type Error = ();

    fn include_pages(
        &mut self,
        includes: &[IncludeRef<'t>],
    ) -> Result<Vec<FetchedPage<'t>>, Self::Error> {
        unsafe {
            let f = match self.0.vtable().include_pages {
                Some(f) => f,
                None => return Err(()),
            };

            // These owned strings have to outlive the call: the FtmlStr values
            // below borrow from them, not from the IncludeRef.
            let names: Vec<String> = includes
                .iter()
                .map(|inc| inc.page_ref().to_string())
                .collect();
            let variables: Vec<Vec<(String, String)>> = includes
                .iter()
                .map(|inc| {
                    let vars = inc.variables();
                    vars.keys()
                        .map(|k| (k.to_string(), vars.get(k).unwrap().to_string()))
                        .collect()
                })
                .collect();
            let borrowed: Vec<Vec<FtmlKeyValue>> = variables
                .iter()
                .map(|pairs| {
                    pairs
                        .iter()
                        .map(|(k, v)| FtmlKeyValue {
                            key: FtmlStr::borrow(k),
                            value: FtmlStr::borrow(v),
                        })
                        .collect()
                })
                .collect();
            let refs: Vec<FtmlIncludeRef> = names
                .iter()
                .zip(borrowed.iter())
                .map(|(name, vars)| FtmlIncludeRef {
                    full_name: FtmlStr::borrow(name),
                    variables: vars.as_ptr(),
                    variable_count: vars.len(),
                })
                .collect();

            let mut sink = FtmlFetchedPageSink { items: vec![] };
            f(self.0.host, refs.as_ptr(), refs.len(), &mut sink);

            // ftml wants one FetchedPage per request, in request order, while the
            // host may answer a subset -- the Python binding's host does exactly
            // that for pages it cannot find.
            let mut out = Vec::with_capacity(includes.len());
            for (index, include) in includes.iter().enumerate() {
                let content = sink
                    .items
                    .iter()
                    .find(|(name, _)| name == &names[index])
                    .and_then(|(_, content)| content.clone())
                    .map(Cow::Owned);
                out.push(FetchedPage {
                    page_ref: include.page_ref().clone(),
                    content,
                });
            }
            Ok(out)
        }
    }

    fn no_such_include(&mut self, page_ref: &PageRef<'t>) -> Result<Cow<'t, str>, Self::Error> {
        unsafe {
            let f = match self.0.vtable().no_such_include {
                Some(f) => f,
                None => return Err(()),
            };
            let name = page_ref.to_string();
            Ok(Cow::Owned(
                self.0
                    .collect(|sink| f(self.0.host, FtmlStr::borrow(&name), sink)),
            ))
        }
    }
}

pub struct FtmlResult {
    body: String,
    included_pages: Vec<String>,
    linked_pages: Vec<String>,
    code: Vec<(String, String)>,
    html: Vec<String>,
}

fn page_refs_to_string(refs: &Vec<PageRef>) -> Vec<String> {
    refs.iter().map(|r| r.to_string()).collect()
}

fn page_refs_to_owned(refs: &Vec<PageRef>) -> Vec<PageRef<'static>> {
    refs.iter().map(|r| r.to_owned()).collect()
}

fn mode_from(mode: &str) -> WikitextMode {
    match mode {
        "article" => WikitextMode::Page,
        "message" => WikitextMode::ForumPost,
        "inline" => WikitextMode::Inline,
        "system" => WikitextMode::System,
        "system-with-modules" => WikitextMode::SystemWithModules,
        _ => WikitextMode::Page,
    }
}

fn render_with<R: Render>(
    source: &str,
    renderer: &R,
    page_info: PageInfo,
    bridge: Rc<HostBridge>,
    mode: WikitextMode,
) -> (R::Output, Vec<String>, Vec<String>, Vec<(String, String)>, Vec<String>) {
    let mut settings = WikitextSettings::from_mode(mode);
    settings.use_include_compatibility = true;

    let mut included_text = source.to_owned();
    let mut included_pages = vec![];
    loop {
        bridge.next_include_level();
        let includer = BridgeIncluder(&bridge);
        let mut current_text = included_text;
        preprocess(&mut current_text);
        let text_with_allowed_no_include = remove_noincludes(&current_text);
        let (next_text, pulled_in) = include(
            &text_with_allowed_no_include,
            &settings,
            includer,
            || panic!("Bad includer return"),
        )
        .unwrap_or((current_text.to_owned(), vec![]));
        included_text = next_text;
        included_pages.append(&mut page_refs_to_owned(&pulled_in));
        if pulled_in.is_empty() {
            break;
        }
    }

    let text = &mut included_text.clone();
    let tokens = tokenize(text);
    let (tree, _warnings) = parse(&tokens, &page_info, bridge.clone(), &settings).into();
    let output = renderer.render(&tree, &page_info, bridge, &settings);

    (
        output,
        page_refs_to_string(&included_pages),
        page_refs_to_string(&tree.internal_links),
        tree.code,
        tree.html,
    )
}

unsafe fn bridge_from(
    callbacks: *const FtmlCallbacks,
    host: usize,
) -> Option<Rc<HostBridge>> {
    if callbacks.is_null() {
        return None;
    }
    Some(Rc::new(HostBridge { callbacks, host }))
}

// A panic must not unwind past an `extern "C"` frame: that is UB.
fn guard<F, T>(body: F) -> *mut T
where
    F: FnOnce() -> Option<T>,
{
    match catch_unwind(AssertUnwindSafe(body)) {
        Ok(Some(value)) => Box::into_raw(Box::new(value)),
        _ => std::ptr::null_mut(),
    }
}

#[no_mangle]
pub unsafe extern "C" fn ftml_render_html(
    source: FtmlStr,
    callbacks: *const FtmlCallbacks,
    host: usize,
    page_info: *const FtmlPageInfo,
    mode: FtmlStr,
) -> *mut FtmlResult {
    guard(|| {
        let bridge = bridge_from(callbacks, host)?;
        let info = page_info.as_ref()?.to_page_info();
        let (output, included_pages, linked_pages, code, html) = render_with(
            source.as_str(),
            &HtmlRender,
            info,
            bridge,
            mode_from(mode.as_str()),
        );
        Some(FtmlResult {
            body: output.body,
            included_pages,
            linked_pages,
            code,
            html,
        })
    })
}

#[no_mangle]
pub unsafe extern "C" fn ftml_render_text(
    source: FtmlStr,
    callbacks: *const FtmlCallbacks,
    host: usize,
    page_info: *const FtmlPageInfo,
    mode: FtmlStr,
) -> *mut FtmlResult {
    guard(|| {
        let bridge = bridge_from(callbacks, host)?;
        let info = page_info.as_ref()?.to_page_info();
        let (body, included_pages, linked_pages, code, html) = render_with(
            source.as_str(),
            &TextRender,
            info,
            bridge,
            mode_from(mode.as_str()),
        );
        Some(FtmlResult {
            body,
            included_pages,
            linked_pages,
            code,
            html,
        })
    })
}

// NullIncluder is deliberate: these collectors report what a page refers to,
// so resolving those references would change the answer.
unsafe fn collect_tree(
    source: FtmlStr,
    callbacks: *const FtmlCallbacks,
    host: usize,
    page_info: *const FtmlPageInfo,
    mode: FtmlStr,
) -> Option<FtmlResult> {
    let bridge = bridge_from(callbacks, host)?;
    let info = page_info.as_ref()?.to_page_info();

    let mut settings = WikitextSettings::from_mode(mode_from(mode.as_str()));
    settings.use_include_compatibility = true;

    let source = source.as_str().to_owned();
    let text = &mut source.clone();
    preprocess(text);
    let (included_text, included_pages) =
        include(text, &settings, NullIncluder {}, || {
            panic!("Mismatched includer page count")
        })
        .unwrap_or((source.clone(), vec![]));

    let text = &mut included_text.clone();
    let tokens = tokenize(text);
    let (tree, _warnings) = parse(&tokens, &info, bridge, &settings).into();

    Some(FtmlResult {
        body: String::new(),
        included_pages: page_refs_to_string(&included_pages),
        linked_pages: page_refs_to_string(&tree.internal_links),
        code: tree.code,
        html: tree.html,
    })
}

#[no_mangle]
pub unsafe extern "C" fn ftml_collect_backlinks(
    source: FtmlStr,
    callbacks: *const FtmlCallbacks,
    host: usize,
    page_info: *const FtmlPageInfo,
    mode: FtmlStr,
) -> *mut FtmlResult {
    guard(|| collect_tree(source, callbacks, host, page_info, mode))
}

#[no_mangle]
pub unsafe extern "C" fn ftml_collect_code_and_html(
    source: FtmlStr,
    callbacks: *const FtmlCallbacks,
    host: usize,
    page_info: *const FtmlPageInfo,
    mode: FtmlStr,
) -> *mut FtmlResult {
    guard(|| collect_tree(source, callbacks, host, page_info, mode))
}

#[no_mangle]
pub unsafe extern "C" fn ftml_result_free(result: *mut FtmlResult) {
    if !result.is_null() {
        drop(Box::from_raw(result));
    }
}

#[no_mangle]
pub unsafe extern "C" fn ftml_result_body(result: *const FtmlResult) -> FtmlStr {
    match result.as_ref() {
        Some(result) => FtmlStr::borrow(&result.body),
        None => FtmlStr::empty(),
    }
}

macro_rules! string_list_accessors {
    ($len:ident, $at:ident, $field:ident) => {
        #[no_mangle]
        pub unsafe extern "C" fn $len(result: *const FtmlResult) -> usize {
            match result.as_ref() {
                Some(result) => result.$field.len(),
                None => 0,
            }
        }

        #[no_mangle]
        pub unsafe extern "C" fn $at(result: *const FtmlResult, index: usize) -> FtmlStr {
            match result.as_ref().and_then(|result| result.$field.get(index)) {
                Some(value) => FtmlStr::borrow(value),
                None => FtmlStr::empty(),
            }
        }
    };
}

string_list_accessors!(
    ftml_result_included_len,
    ftml_result_included_at,
    included_pages
);
string_list_accessors!(ftml_result_linked_len, ftml_result_linked_at, linked_pages);
string_list_accessors!(ftml_result_html_len, ftml_result_html_at, html);

#[no_mangle]
pub unsafe extern "C" fn ftml_result_code_len(result: *const FtmlResult) -> usize {
    match result.as_ref() {
        Some(result) => result.code.len(),
        None => 0,
    }
}

#[no_mangle]
pub unsafe extern "C" fn ftml_result_code_language_at(
    result: *const FtmlResult,
    index: usize,
) -> FtmlStr {
    match result.as_ref().and_then(|result| result.code.get(index)) {
        Some((language, _)) => FtmlStr::borrow(language),
        None => FtmlStr::empty(),
    }
}

#[no_mangle]
pub unsafe extern "C" fn ftml_result_code_contents_at(
    result: *const FtmlResult,
    index: usize,
) -> FtmlStr {
    match result.as_ref().and_then(|result| result.code.get(index)) {
        Some((_, contents)) => FtmlStr::borrow(contents),
        None => FtmlStr::empty(),
    }
}

// The ABI version is part of the symbol name so a mismatched library fails at
// link time rather than at runtime. Bump it here, in ftml.h and in ABI_VERSION
// together whenever the C interface changes shape.
#[no_mangle]
pub extern "C" fn ftml_abi_1() {}

#[no_mangle]
pub extern "C" fn ftml_version() -> FtmlStr {
    FtmlStr::borrow(VERSION.as_str())
}
