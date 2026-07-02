# Blocks 

Blocks are site areas that are rendered discretely.    Each block can belong to one or more spaces spaces are rendered in a template ie  a block type "content" could be assigned to space "footer" and then {{space "footer" }} could be called in the template to render the block.  

The admin view should allow filtration of blocks by the space, spaces are adhoc strings 

# Block Types 

There are two types of blocks native and extension

Native Blocks - HTML, Markdown
Extension Blocks - custom block if a block exposes /block capability it should list it's blocks at that endpoint 

# Block Templates

Block wrapper templates live under `partials/blocks/` in the active frontend theme, e.g. `default/partials/blocks/default.html`. Add new wrappers in that folder and reference them from the block **Template Wrapper** field in admin.

Login blocks use `default/partials/blocks/login.html` by default.

Legacy paths `default/partials/block.html` and `default/partials/login-block.html` are still resolved automatically.

