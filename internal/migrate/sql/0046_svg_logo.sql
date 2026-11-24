-- compat: breaking
UPDATE web_theme
SET css = replace(css, '/-/static/images/wikitHana.png', '/-/static/images/wikitHana-white.svg'),
    updated_at = now()
WHERE css LIKE '%/-/static/images/wikitHana.png%';
