# Templates

Site and Admin Templates should be derived form sites.json each root folder in the list is a potential template

In the admin we should support listing and editing the files within. These files will be html and should be managed as raw files. 

We should support versioning via timestamp in the filename of the file so if the template is called default.html 

Prior to save we should right a version-$date-defaul.html, we should only keep a history of 3 copies to keep the system from getting too crowded.  They should be stored in a mirrored versions directory.

Template files should override the in built templates ie if we have a file admin/dashboard.html in our templates folder it would override the one in templateengine/admin/dashboard.html  

