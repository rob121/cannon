Build out the content controller for cannon cms, much of this feature list is taken from the k2 content manager here https://getk2.org/documentation.

Build out the content controler in the admin and frontend based on the items below.  Where it makes sense make a separate controller (ie media) 

Use best practice ui/ux especially in the administrator area.    When in doubt ask for feedback before making big decisions.

Core concept:
- The primary content unit is an "Item", not just an article.
- Items support structured content, rich media, custom fields, category-specific layouts, tags, comments, author profiles, and frontend/backend editing.
- The CMS should be usable for blogs, magazines, catalogs, portfolios, knowledge bases, directories, event listings, and download/document libraries.

Primary entities:
1. Items
   - title
   - slug/alias
   - intro text / excerpt
   - full body
   - status: draft, published, archived, trashed
   - featured flag
   - publish start/end dates
   - author
   - category
   - tags
   - image
   - image gallery
   - video/audio embed or upload
   - attachments/downloads
   - metadata: meta title, description, keywords, canonical URL
   - ordering / manual sort
   - created/modified timestamps

2. Categories
   - nested parent/child categories
   - category slug
   - description
   - image
   - category-specific display/template settings
   - category-specific field group assignment
   - ability to inherit settings from parent category
   - access control per category

3. Tags
   - reusable tag taxonomy
   - tag pages listing matching items
   - backend tag management

4. Custom Fields / Extra Fields
   - field groups
   - assign field groups to categories/content types
   - supported field types:
     - text
     - textarea
     - rich text
     - select
     - multi-select
     - radio
     - checkbox
     - date/time
     - number
     - URL/link
     - image
     - file
     - boolean
   - fields should be renderable in templates
   - fields should support validation rules
   - fields should support required/optional flags
   - fields should support ordering

5. Media Manager (under a separate media controller in admin, but able to be connected)
   - upload images/files
   - drag/drop upload
   - organize media by folders or collections
   - auto-generate image sizes/thumbnails
   - configurable image dimensions globally and per category
   - attach files to items
   - support galleries per item

6. Comments
   - comments per item
   - moderation queue
   - approve/unapprove
   - spam controls
   - optional anonymous comments
   - optional registered-user-only comments
   - comment counts
   - notification hooks

7. Users and Authors
   - extended user profile
   - author bio
   - avatar
   - author page/blog
   - list items by author
   - frontend author dashboard
   - user groups for editorial permissions

8. Access Control
   - role/user group system
   - permissions for:
     - create item
     - edit own item
     - edit any item
     - publish/unpublish
     - delete/trash
     - manage categories
     - manage tags
     - manage comments
     - upload media
     - manage custom fields
   - category-level permissions
   - frontend editing permissions

9. Frontend Editing Controller
   - logged-in users with permission can create/edit items from frontend
   - simplified item form
   - media upload support
   - custom fields support
   - save as draft / submit for review / publish depending on permissions

10. Templates and Layouts
   - item detail templates
   - category/list templates
   - tag listing templates
   - author listing templates
   - override-friendly template system
   - category-specific template selection
   - support sub-templates/partials
   - theme-safe rendering helpers for fields/media/tags/comments

11. Content Block Types
   - latest items
   - featured items
   - popular items
   - related items
   - items by category
   - items by tag
   - items by author
   - item carousel/list/grid
   - archive/month list
   - tag cloud
   - category tree/menu


13. Search and Filtering
   - full-text item search
   - filter by category
   - filter by tag
   - filter by author
   - filter by custom fields
   - sort by date, title, ordering, popularity, featured

14. URLs and SEO
   - clean SEO-friendly slugs
   - category/item URL structure
   - tag URLs
   - author URLs
   - canonical URLs
   - Open Graph/Twitter metadata
   - XML sitemap hooks
   - RSS/Atom feeds for category/tag/author/global item streams

15. Plugin/Extension API
   - hooks/events for item form rendering
   - hooks before/after item save
   - hooks before/after item render
   - hooks for category/user forms
   - hooks for comments
   - hooks for media processing
   - plugin-defined custom form fields
   - plugin-defined frontend render blocks

16. Admin Dashboard
   - stats for items, categories, tags, comments, custom fields, users, trashed items
   - quick links to create item/category/tag/field
   - filterable item list
   - bulk publish/unpublish/trash/delete
   - bulk category/tag assignment


