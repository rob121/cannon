This application is going to be a content management CMS called cannon

Your goal is to generate a system that is simple/elegant. Prefer reuasability featuring DRY principles.

When in doubt ask for feedback about the best way to accomplish things

follow the instructions in the "specs" folder .info/specs

# Overview

The content management system is written in golang

# Features

* Dyanmic Routing
* Extensions and Plugins
* Custom Templating
* User System
* Multisite
* Administrative Panel
* Langauge/Locale/i18n Pack for both main site and admin (editable via admin)

Configuration and Bootstrapping

On initial startup the application should look for a sites.json file. Using https://github.com/rob121/vhelp

This file is the bootstrap that holds the sites and the individual configuration e.g

extension:{
 "dir": "/opt/sites/extensions"
}
"sites" [
{
    "id": "site1",
    "name": "My Site"
    "database: "mysql://username:password@tcp(127.0.0.1:3306)/my_database?charset=utf8mb4&parseTime=True&loc=Local",
    "template_dir":"/opt/site1/templates",
    "assets_dir: "/opt/site1/assets",
    "host": "https://example2.com",
    "tmp_dir": "/tmp"
    "language_dir": "/poath/to/language"
},
{
    "id": "site2",
    "name": "My Site 2"
    "host": "https://example.com",
    "database: "sqlite:/path/to/database/mydb.sqlite",
    "template_dir":"/opt/site2/templates",
    "assets_dir: "/opt/site2/assets",
    "tmp_dir": "/tmp"
}
]

Template dir is where the templates are stored for the site, assets dir are public files (css, images,etc), Tmp Dir is where any tmp files are written.  The templates for the default site are going be part of the core and so if the template dir is empty then the default is used. There are two templates

"default" the frontend site 
"admin"  the admin management console template


We should have a sites package that holds the configuration for each individual site and can resolve from the request the specific site being operated against. Additionally in bootstrapping it should make a db connection that is referenceable against the current active request so in context serving a request we pull the right db connection etc.

The database should be using Gorm and should support postgres,mysql,sqlite with sqlite being the default.  

# Installation

When the application starts up if sites.json is empty or lists no sites then regardless of the http route requested we should redirec the visitor to /install 

install is multistep 1st to provide enough information to fill sites.json and then if sites.json is used but no users exist in the system, provide a form to boostrap an administrator user.  The 3rd step is to redirect the user to /admin where the administrative site will live.  Once having sites.json written the system should bootstrap the database.

# Extensions

Extensions are pre compiled binaries that the application will start at startup as child processes,extensions must be saved to the directory in the sites.json which is independent of the sites themselves. the extensions manager will have the capability to mark them enabled/disabled and also re-start a process if necessary.  Each process will open a socket that the cms can connect to.  When a process is added a hash of the process name and site name is used to create a socket address. The socket address is saved in the db and used to pass to the start of the process so that each end knows about the connection ie $processname --socket=/opt/sites/sockets/34652543623462365.sock  --site=$id the db should have a sort column to define which order extensions are executed in. 

On startup the configuration from the db will start the  extension processses and make a inital /capabilities request.  More about specification on extensions is found in specs/extensions.md

# Tables 

This is a list of tables I expect we'll need

Authenticators
  auth_id
  name
  status (active/inactive)
  configuration (json)
Users
  user_id
  given_name
  family_name
  email
  username
  locked (bool)
  validated (bool)
  hash
  status (active/inactive)
  auth_id (See Authenticators)
UserProfile
  user_profile_id
  user_id
  profile_id 
Profiles
  profile_id
  name
ProfileFields
  profile_field_id
  profile_id
  name
  type
  sort
  status (active/inactive)
ProfileUserFieldValue
  user_id
  field_id
  value  
Extensions
  extension_id
  name
  socket
  status (active/inactive)
Menu
  menu_id
  menu_name
  status (active/inactive)
MenuItem
  menu_item_id
  name
  route_id
  class
  target
Routes - list of routes with support for wildcards and also regex ie /contact or /contact/* and /conta[a-z0-9].  
   route_id
   name
   path e.g /contact
   type (Url, Extension, Local File,Controller)
   status (active/inactive)

# Routing

HTTP Request routing must be dynamic by it's nature to serve the system. Before a request is processed we expect the state of extensions to be loaded, the capabilities should be exposed and so a "extensions" package should be called in middleware and pass the state of the request to the appropriate active extensions for any possible changes/alterations

# Custom Templating

The application once started up and serving requests should reference the specific site configuration for templates.   The system should operate using go standard templates and so that when a template path is requested or a sub template path is requested it should first look in the defined template and then in default.

# User System

Users utilities and current user state should be accessed through a user package, especially since other processes may be pulling in the package in extensions from the current request being passed in, for this we should have a shared session location in the tmp file directory.

Using https://github.com/markbates/goth I want to optionally support login via these other methods.  We shoud enable/disable them in Authenticators the configuration fields and names of supported providers should be loaded in at bootstrapping/install

For built in authentication we should support robust bcrypt based passwords and usernames in the Users table

# Multisite

As already outlined we support multisite we should have multiple sites be able to be built and when routing to that domain load the context for the correct site.

# Administrative Panel

The admin panel can be refrenced more in specs/admin_design.md

# Language Support

Language can be supported via a langague directory where files are stored in the format ISO 639-1 code ie en-US.ini, any file with en-US.ini or en-US-*.ini will be loaded and parse for language information. A language package should expose a lang.Fmt("user.registrered","Username","Dave")

in the en-US.ini file we expect to see

[user]
registered = "You have sucessfully registred as {{.Username}}

When lang.Fmt("user.registrered","Username","Dave") is processed we'd see

"You have sucessfully registred as Dave"