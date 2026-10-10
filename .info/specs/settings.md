We need under system->configuration page, this page should list a global configuraiton and then configurations for each extension - each extension is responsible for it's own settings page creation this ui should just present the form and post to /configuraiton for update In a side menu should be a list of "GLobal" and Then "Extensions->*" when clicking on an extension we should load the form.  

Global Configuration may have multiple sections so we should suppor tmultiple configuration definitions (mroe below) for each section ie mail,etc.  

Regardless all configurations both global and extension should be based off https://jsonforms.io/ json definitons

This go implementation will render the forms from jsonforms.io: https://github.com/TobiEiss/go-jsonforms

Add to the extension package and extensions.md information and support for the jsonforms.io format to be rendered on /configuration and to be written with POST /configuration.

We need a interface (package) for pulling data for our configurations as well.





