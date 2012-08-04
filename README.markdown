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
      daily
      missingok
      rotate 90
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
    </Directory>

Varnish
-------
Protect /overview and /monitor using ACL. Otherwise, the web application will provide OK cache-control headers.

    acl admins {
      "1.2.3.4";
      "2.3.4.5";
    }
    
    acl monitor {
      "3.4.5.6";
    }
    
    sub vcl_recv {
      # [...]
      if (req.url ~ "^/overview"){
        if (!client.ip ~ admins) {
          error 403 "Forbidden";
        }
      }
      if (req.url ~ "^/monitor"){
        if (!client.ip ~ monitor) {
          error 403 "Forbidden";
        }
      }
    }

TODO
----
* Improved archive downloading (streaming).
* API documentation.
* Statistics in the admin interface.
* Proper MD5 checksumming during upload.
* Client side error handling.
* PURGE requests to Varnish.
