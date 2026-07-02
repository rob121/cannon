Review the admin content area, the tables with filters and search boxes etc. Basically all card headers need to be reviewed as the layouts are wonky.

Add more help docs about how to author extensions and for supporting different capabilites

Global Mail configuration needs to have username/password and from name

Add a backend page for Notifications 

Each notificaiton is a shouttrr (https://github.com/containrrr/shoutrrr) config ie slack,mail etc.  

Notifcation Model
 notification_id
 name
 shoutrr configuration
 status (active/inactive)

We should use event hooks for notificaitons so for each configured notification we should register which hooks apply

onUserSignup
onUserLogin
onUserVerified
onUserLocked


Additional backend user groups hierarchy

Administrator
 -> Manager
   -> Editor
    -> Writer



 We need these groups bootstrapped when the site start ups these are for the admin backend

Additionally frontend groups that always exist are 

Public->
  Registered

 The bakcned area should allow Administrator users to define what routes under /admin each group can crud to.

 Other groups may be used as well later .

 Other Site Global Config Fields (and then make sure the values are used in the various pages)

 * Site Offline  (bool) (show maintenance page)
 * Default List Limit (for per page table views)
 * Site Meta Description
 * Robots and Ai settings for /robots.txt etc.



Editing a file here http://127.0.0.1:8001/admin/templates does not work 

Add support for a template.json in the root of a template that has meta data ie frontend or backend temptale author etc. This metadata should be added in under /admin/templates to help with the display. Additonally we should be able to mark a template status 

Still to polish (follow-up)
Admin content list card headers — unified toolbar partial across items/categories/blocks (partial CSS fix only)
default_list_limit — stored in settings but admin lists still use hard-coded pageSize = 20 - also this should be a drop down in configuration 25,50,75,100,200,500
site_meta_description — not yet injected into public layout (field exists in settings)
OnUserLocked — hook exists; fire it from user lock toggle in admin users CRUD
Assign registered group on login (currently only on user create)
Then configure notifications under System → Notifications and group permissions under Users → Groups.
Add a "Log Level" in the config - Debug,Error,Warning, Info, None

Extensions is listed 2x here: http://127.0.0.1:8001/admin/help - There is no information on authoring extensions (from extensions.md)

http://127.0.0.1:8001/admin/media/upload - deeosn't work, it needs to be more polished and when uploading images we should see a preview. Additonally http://127.0.0.1:8001/admin/media should be formatted as a file explorer and not a table. 

See this page for layout guidance: https://apex-shadcn.dashboardpack.com/files


Implement the same table sort arrows on the blocks page as in the extensions page and then on Content/Categories 


http://localhost:8001/ has a route for controller category with a predefined category on the frontend I still get "No items in this category."

Log output for access log - with support for automatic rotation the log should be basedon the hostname of the site.

http://127.0.0.1:8001/admin/blocks/2 blocks need a configurate to show/hide dpeending on the route.  

http://127.0.0.1:8001/admin/items/new - still needs ui improvement, there are a lot of areas and it's very visually busy, save and cancel should be ain a toolbar at the top as it's too busy and you have to scroll to the bottom of the page.

http://127.0.0.1:8001/admin/field-groups/new implements requried, as a checkbox all admin checkboxes should render as a toggle 

_gothic_session cookie is set, can we rename that to _cannon_session?

http://127.0.0.1:8001/admin/field-groups - adding a field gorup works, adding a field within the group overwrites the ffield group or adds a new group.   Needs to be reviewed.

http://127.0.0.1:8001/admin/categories/1 group visibility should be on the sidebar and not th4e bottom, checkboxes should be toggles.


Add content settings for the frontend to global configuration
* Show Author
* Show published date
* Show Author Bio 
* Show title
* Show Comments

http://localhost:8001/admin/categories/1 wraps qutes around the access levels when "enabled"
http://localhost:8001/admin/categories is missing sorting capability like extensions
http://localhost:8001/admin/blocks is missing sorting capabiliyt like extensions
http://localhost:8001/admin/routes/4 group visibiliy/access should be layed out like http://localhost:8001/admin/categories/new is
http://localhost:8001/admin/items/new - the save/cancel buttons should be right justified in the toolbar like the rest of the admin ui

--


Add support for MFA and also passkeys

Estalbish a configuraiton setting to enable either of these for the user to setup

Add to the profile area interfaces to add mf and passkeys 

Update the login form to suppor thtese methods along with sso auth authenticators we previously setup.



Extensions

The contact extension html needs to have boostrap css classes added so it renders right

The calendar extension has an error: Last refresh failed: no such column: meeting_details also:
The issue is the google meet information is in the description, there are two things - here is sample text o fhte meeting information: Join with Google Meet: https://meet.google.com/yfa-bmuo-xdg\nOr 
  dial: (US) +1 402-735-0126 PIN: 332516045#\nMore phone numbers: https://te
 l.meet/yfa-bmuo-xdg?pin=9971233969167&hs=7\n\nLearn more about Meet at: htt
 ps://support.google.com/a/users/answer/9282720 There is also a X-GOOGLE-CONFERENCE key that can be used as a hint to hide the text, it appears that this text is appended so when X-GOOGLE-CONFERENCE has a value then we should splikt on "Join with Google Meet to hide if login is required.