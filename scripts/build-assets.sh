#!/bin/sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
STATIC_DIR="$ROOT_DIR/web/static"
ASSET_DIR="$ROOT_DIR/web/assets"

hash_file() {
	sha256sum "$1" | awk '{ print substr($1, 1, 7) }'
}

publish_static_file() {
	source_path=$1
	target_path=$2
	chmod 0644 "$source_path"
	mv "$source_path" "$target_path"
}

find_one() {
	pattern=$1
	result=$(find "$STATIC_DIR" -maxdepth 1 -type f -name "$pattern" | sort | head -n 1)
	if [ -z "$result" ]; then
		echo "missing static asset matching $pattern" >&2
		exit 1
	fi
	echo "$result"
}

rehash_static_asset() {
	pattern=$1
	target_prefix=$2
	target_suffix=$3
	source_path=$(find_one "$pattern")
	hash=$(hash_file "$source_path")
	target_name="${target_prefix}.${hash}${target_suffix}"
	target_path="$STATIC_DIR/$target_name"

	if [ "$(basename "$source_path")" = "$target_name" ]; then
		echo "$target_name"
		return
	fi

	tmp_path=$(mktemp "${TMPDIR:-/tmp}/caldo-asset.XXXXXX")
	cp "$source_path" "$tmp_path"
	rm -f "$STATIC_DIR"/$pattern
	publish_static_file "$tmp_path" "$target_path"
	echo "$target_name"
}

build_css() {
	tmp_css=$(mktemp "${TMPDIR:-/tmp}/caldo-css.XXXXXX")
	if command -v tailwindcss >/dev/null 2>&1; then
		BROWSERSLIST_IGNORE_OLD_DATA=1 tailwindcss -c "$STATIC_DIR/tailwind.config.js" -i "$STATIC_DIR/tailwind.input.css" -o "$tmp_css" --minify
	else
		existing_css=$(find_one "app.*.css")
		echo "tailwindcss not found; rehashing existing CSS bundle" >&2
		cp "$existing_css" "$tmp_css"
	fi

	hash=$(hash_file "$tmp_css")
	target_name="app.${hash}.css"
	rm -f "$STATIC_DIR"/app.*.css
	publish_static_file "$tmp_css" "$STATIC_DIR/$target_name"
	echo "$target_name"
}

build_app_js() {
	source_path="$ASSET_DIR/app.js"
	if [ ! -f "$source_path" ]; then
		echo "missing app JS source: $source_path" >&2
		exit 1
	fi

	hash=$(hash_file "$source_path")
	target_name="app.${hash}.js"
	rm -f "$STATIC_DIR"/app.*.js
	cp "$source_path" "$STATIC_DIR/$target_name"
	chmod 0644 "$STATIC_DIR/$target_name"
	echo "$target_name"
}

build_source_asset() {
	source_name=$1
	target_pattern=$2
	target_prefix=$3
	target_suffix=$4
	source_path="$ASSET_DIR/$source_name"
	if [ ! -f "$source_path" ]; then
		echo "missing source asset: $source_path" >&2
		exit 1
	fi

	hash=$(hash_file "$source_path")
	target_name="${target_prefix}.${hash}${target_suffix}"
	rm -f "$STATIC_DIR"/$target_pattern
	cp "$source_path" "$STATIC_DIR/$target_name"
	chmod 0644 "$STATIC_DIR/$target_name"
	echo "$target_name"
}

write_manifest() {
	css_name=$1
	app_js_name=$2
	htmx_name=$3
	htmx_sse_name=$4
	alpine_name=$5
	favicon_svg_name=$6
	favicon_png_name=$7
	apple_touch_icon_name=$8
	tmp_manifest=$(mktemp "${TMPDIR:-/tmp}/caldo-manifest.XXXXXX")

	{
		printf '{\n'
		printf '  "app.css": "%s",\n' "$css_name"
		printf '  "app.js": "%s",\n' "$app_js_name"
		printf '  "htmx.min.js": "%s",\n' "$htmx_name"
		printf '  "htmx-sse.js": "%s",\n' "$htmx_sse_name"
		printf '  "alpine.min.js": "%s",\n' "$alpine_name"
		printf '  "favicon.svg": "%s",\n' "$favicon_svg_name"
		printf '  "favicon.png": "%s",\n' "$favicon_png_name"
		printf '  "apple-touch-icon.png": "%s"\n' "$apple_touch_icon_name"
		printf '}\n'
	} > "$tmp_manifest"

	publish_static_file "$tmp_manifest" "$STATIC_DIR/manifest.json"
}

css_name=$(build_css)
app_js_name=$(build_app_js)
htmx_name=$(rehash_static_asset "htmx.*.min.js" "htmx" ".min.js")
htmx_sse_name=$(rehash_static_asset "htmx-sse.*.js" "htmx-sse" ".js")
alpine_name=$(rehash_static_asset "alpine.*.min.js" "alpine" ".min.js")
favicon_svg_name=$(build_source_asset "favicon.svg" "favicon.*.svg" "favicon" ".svg")
favicon_png_name=$(build_source_asset "favicon-32.png" "favicon.*.png" "favicon" ".png")
apple_touch_icon_name=$(build_source_asset "apple-touch-icon.png" "apple-touch-icon.*.png" "apple-touch-icon" ".png")

write_manifest "$css_name" "$app_js_name" "$htmx_name" "$htmx_sse_name" "$alpine_name" "$favicon_svg_name" "$favicon_png_name" "$apple_touch_icon_name"
