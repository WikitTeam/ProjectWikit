//go:build cgo && !nocgo

#include "ftml.h"
#include "_cgo_export.h"

void pwikit_fill_callbacks(FtmlCallbacks *out) {
  out->module_has_body = pwikitModuleHasBody;
  out->render_module = pwikitRenderModule;
  out->render_user = pwikitRenderUser;
  out->get_i18n_message = pwikitGetI18nMessage;
  out->get_html_injected_code = pwikitGetHTMLInjectedCode;
  out->get_page_info = pwikitGetPageInfo;
  out->evaluate_expression = pwikitEvaluateExpression;
  out->normalize_page_name = pwikitNormalizePageName;
  out->include_pages = pwikitIncludePages;
  out->no_such_include = pwikitNoSuchInclude;
  out->next_include_level = pwikitNextIncludeLevel;
}
