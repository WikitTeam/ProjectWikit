#ifndef PWIKIT_FTML_H
#define PWIKIT_FTML_H

#include <stddef.h>
#include <stdint.h>

typedef struct {
  const char *ptr;
  size_t len;
} FtmlStr;

typedef struct {
  FtmlStr key;
  FtmlStr value;
} FtmlKeyValue;

typedef struct {
  FtmlStr full_name;
  const FtmlKeyValue *variables;
  size_t variable_count;
} FtmlIncludeRef;

#define FTML_EXPR_NONE 0
#define FTML_EXPR_STRING 1
#define FTML_EXPR_BOOL 2
#define FTML_EXPR_FLOAT 3
#define FTML_EXPR_INT 4

typedef struct {
  int kind;
  int64_t int_value;
  double float_value;
} FtmlExpressionResult;

typedef struct FtmlStringSink FtmlStringSink;
typedef struct FtmlPageInfoSink FtmlPageInfoSink;
typedef struct FtmlFetchedPageSink FtmlFetchedPageSink;
typedef struct FtmlResult FtmlResult;

void ftml_sink_string(FtmlStringSink *sink, FtmlStr value);
void ftml_sink_page_info(FtmlPageInfoSink *sink, FtmlStr full_name, FtmlStr title,
                         int has_title, int exists);
void ftml_sink_fetched_page(FtmlFetchedPageSink *sink, FtmlStr full_name, FtmlStr content,
                            int has_content);

/* The host is a Go cgo.Handle, which is a small integer and not a pointer.
   Declaring it uintptr_t keeps Go's garbage collector from scanning it as one. */
typedef struct {
  int (*module_has_body)(uintptr_t, FtmlStr);
  void (*render_module)(uintptr_t, FtmlStr, FtmlKeyValue *, size_t, FtmlStr, FtmlStringSink *);
  void (*render_user)(uintptr_t, FtmlStr, int, FtmlStringSink *);
  void (*get_i18n_message)(uintptr_t, FtmlStr, FtmlStringSink *);
  void (*get_html_injected_code)(uintptr_t, FtmlStr, FtmlStringSink *);
  void (*get_page_info)(uintptr_t, FtmlStr *, size_t, FtmlPageInfoSink *);
  void (*evaluate_expression)(uintptr_t, FtmlStr, FtmlExpressionResult *, FtmlStringSink *);
  void (*normalize_page_name)(uintptr_t, FtmlStr, FtmlStringSink *);
  void (*include_pages)(uintptr_t, FtmlIncludeRef *, size_t, FtmlFetchedPageSink *);
  void (*no_such_include)(uintptr_t, FtmlStr, FtmlStringSink *);
  int (*next_include_level)(uintptr_t);
} FtmlCallbacks;

typedef struct {
  FtmlStr page;
  FtmlStr category;
  FtmlStr site;
  FtmlStr title;
  FtmlStr domain;
  FtmlStr media_domain;
  double rating;
  const FtmlStr *tags;
  size_t tag_count;
  FtmlStr language;
} FtmlPageInfo;

FtmlResult *ftml_render_html(FtmlStr source, const FtmlCallbacks *callbacks, uintptr_t host,
                             const FtmlPageInfo *page_info, FtmlStr mode);
FtmlResult *ftml_render_text(FtmlStr source, const FtmlCallbacks *callbacks, uintptr_t host,
                             const FtmlPageInfo *page_info, FtmlStr mode);
FtmlResult *ftml_collect_backlinks(FtmlStr source, const FtmlCallbacks *callbacks, uintptr_t host,
                                   const FtmlPageInfo *page_info, FtmlStr mode);
FtmlResult *ftml_collect_code_and_html(FtmlStr source, const FtmlCallbacks *callbacks, uintptr_t host,
                                       const FtmlPageInfo *page_info, FtmlStr mode);

void ftml_result_free(FtmlResult *result);
FtmlStr ftml_result_body(const FtmlResult *result);
size_t ftml_result_included_len(const FtmlResult *result);
FtmlStr ftml_result_included_at(const FtmlResult *result, size_t index);
size_t ftml_result_linked_len(const FtmlResult *result);
FtmlStr ftml_result_linked_at(const FtmlResult *result, size_t index);
size_t ftml_result_html_len(const FtmlResult *result);
FtmlStr ftml_result_html_at(const FtmlResult *result, size_t index);
size_t ftml_result_code_len(const FtmlResult *result);
FtmlStr ftml_result_code_language_at(const FtmlResult *result, size_t index);
FtmlStr ftml_result_code_contents_at(const FtmlResult *result, size_t index);
FtmlStr ftml_version(void);

/* The vtable is filled on the C side because cgo cannot take the address of an
   exported Go function from Go code. */
void pwikit_fill_callbacks(FtmlCallbacks *out);

#endif
