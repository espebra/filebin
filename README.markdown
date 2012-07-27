About
=====
Filebin is a very simple web application that supports multifile upload and large file uploads. The upload feature is based on XmlHttpRequest. Clients are required to use browsers with File API support to be able to upload files.

Requirements
============
Some Python modules, MongoDB and Apache.

Additional configuration
========================

Logrotate
---------
    /var/log/app/filebin/*.log {
      weekly
      missingok
      rotate 52
      compress
      notifempty
      copytruncate
      create 640 www-data www-data
      sharedscripts
    }

Apache
------
    <Directory /srv/www/filebin>
      RewriteEngine On
      RewriteCond %{REQUEST_FILENAME} !-f
      RewriteRule ^(.*)$ index.py/$1 [QSA,L]
    
      Options +ExecCGI -MultiViews +SymLinksIfOwnerMatch
      Order allow,deny
      Allow from all
      AddHandler cgi-script .py
      DirectoryIndex index.py
      AddHandler default-handler .html .htm
    </Directory>

TODO
----
* Example Varnish configuration.
* Improved archive downloading (streaming).
* API documentation.
* Statistics in the admin interface.
* Proper MD5 checksumming during upload.
* Client side error handling.

