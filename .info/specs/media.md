# Media

Media manager under `/admin/media` for uploading, browsing, and deleting files stored under the site assets directory (`/assets/media/...`).

## Library

- Model: `MediaAsset` (`path`, `mime`, `folder`, `alt`, dimensions).
- Configuration → Media: max upload size, approved extensions, **image styles**.
- Featured images and galleries on content store the **original** public URL string (not a media ID).

## Image styles

Named derivatives generated on raster image upload (JPEG/PNG/GIF; WebP decode supported; SVG skipped).

### Defaults

| Key | Width | Mode | File suffix |
|-----|------:|------|-------------|
| `thumbnail` | 320 | fit | `_thumb` (legacy-compatible) |
| `medium` | 640 | fit | `_medium` |
| `large` | 1200 | fit | `_large` |

Configured in Configuration → Media as `image_styles` (array of `{key, width, height, mode}`). Keys: `[a-z0-9_]`. Mode v1: `fit` only (max width, optional max height, never upscale). Height `0` = proportional from width.

### Path convention

Sibling of the original:

`/assets/media/folder/photo.jpg` → `/assets/media/folder/photo_medium.jpg`

Style key `thumbnail` uses `_thumb` so existing thumbnails keep working.

### Template helper

```
{{imageStyle .Item.Image "medium"}}
```

Resolves to the derivative URL when the file exists on disk; otherwise returns the original path.

Default theme usage:

- Listing / search cards: `medium`
- Item detail featured image: `large`
- Admin media picker: `thumbnail` via `ResolveWebThumbnail`

### Generation and cleanup

- Upload (media library, avatars, profile field images) runs `GenerateStyles` for all configured styles.
- Delete removes known style siblings plus the original.
- Changing styles in settings does **not** regenerate the library (no admin regenerate-all in v1).

## Out of scope (later)

- Per-category style maps
- Crop / focal point
- WebP re-encode / CDN URL rewrite
- On-the-fly HTTP transform endpoint
- Content fields storing media IDs instead of paths
