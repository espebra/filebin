About
=====
Filebin is a very simple web application that supports multifile upload and large file uploads. The upload feature is based on XmlHttpRequest. Clients are required to use browsers with File API support to be able to upload files.

Requirements
============
Some Python modules, MongoDB and Apache or Nginx+uwsgi.

For ubuntu:
    sudo apt-get install mongodb-server python-pymongo python-jinja2 python-flask python-werkzeug python-magic python-pythonmagick python-pyexiv2 

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

Nginx 
-------
    location / { try_files $uri @filebin; }
    location @filebin {
        include uwsgi_params;
        uwsgi_read_timeout 6000;
        uwsgi_send_timeout 6000;
        client_max_body_size 1024M;
        client_body_buffer_size 128k;
        uwsgi_pass unix:/run/shm/filebin.sock;
    }

Uwsgi (filebin.yaml)
-------
    uwsgi:
        uid: www-data
        gid: www-data
        socket: /run/shm/filebin.sock
        plugins: http,python
        chmod-socket: 666
        processes: 2
        module: filebin
        callable: app
        chdir: /srv/www/filebin

Varnish
-------
Protect /overview and /monitor using ACL. Otherwise, the web application will provide OK cache-control headers. This example works for Varnish 2.

    acl purge {
      "localhost";
    }
    acl admins {
      "1.2.3.4";
      "2.3.4.5";
    }
    acl monitor {
      "3.4.5.6";
    }
    
    sub vcl_recv {
      # [...]
      if (req.request == "PURGE") {
        if (!client.ip ~ purge) {
          error 405 "Not allowed.";
        } else {
          purge_url(req.url);
          error 200 "Purged";
        }
      }
      if (req.url ~ "^/archive"){
        # For streaming download
        return (pipe);
      }
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
* API documentation.
* Statistics in the admin interface.
* Proper MD5 checksumming during upload.
* Client side error handling.
