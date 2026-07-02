# Blocks 

Blocks are site areas that are rendered discretely.    Each block can belong to one or more spaces spaces are rendered in a template ie  a block type "content" could be assigned to space "footer" and then {{space "footer" }} could be called in the template to render the block.  

The admin view should allow filtration of blocks by the space, spaces are adhoc strings 

# Block Types 

There are two types of blocks native and extension

Native Blocks - HTML, Markdown
Extension Blocks - custom block if a block exposes /block capability it should list it's blocks at that endpoint 

# Block Definition

Each block definiton should specify block type, space, name and then depending on the block type selected metadata related to it ie a form id 

