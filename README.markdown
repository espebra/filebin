About
=====
Filebin is a very simple web application that supports multifile upload and large file uploads. The upload feature is based on XmlHttpRequest. Clients are required to use browsers with File API support to be able to upload files.

Requirements
============
Some Python modules, MongoDB and Apache or Nginx+uwsgi.

For ubuntu:
    sudo apt-get install mongodb-server python-pymongo python-jinja2 python-flask python-werkzeug python-magic python-pythonmagick python-pyexiv2 uwsgi uwsgi-plugin-python

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

Nginx 
-----
    server {
        listen   80;
        listen   [::]:80;
    
        access_log /var/log/nginx/filebin.net-access.log;
        server_name www.filebin.net;
        rewrite ^(.*) http://filebin.net$1 permanent;
    }

    server {
        listen   80;
        listen   [::]:80;
    
        access_log /var/log/nginx/filebin.net-access.log;

        # Please note that the root path should point to the directory where
        # the tags and files are being stored.
        root /srv/www/filebin/files/;

        index index.html index.htm;
        server_name filebin.net;
    
        # http://aspyct.org/blog/2012/08/20/setting-up-http-cache-and-gzip-with-nginx/
        gzip on;
        gzip_http_version 1.1;
        gzip_vary on;
        gzip_comp_level 6;
        gzip_proxied any;
        gzip_types text/plain text/css application/json application/x-javascript text/xml application/xml application/xml+rss text/javascript application/javascript text/x-js;
        gzip_buffers 16 8k;
        gzip_disable "MSIE [1-6]\.(?!.*SV1)";
        server_tokens off;
    
        # For historical compability
        rewrite ^/([^/]+)/file/(.*) http://filebin.net/$1/$2 permanent;
        rewrite ^/admin/([^/]+)/([^/]+)/(.+) http://filebin.net/$1?v=configuration&key=$2 permanent;
    
        location /static {
            root /srv/www/filebin/;
        }
    
        location /thumbnails {
            root /srv/www/filebin/;
        }
    
        location /uploader {
            limit_except POST          { deny all; }
    
            client_body_temp_path      /srv/www/filebin/temp;
            client_body_in_file_only   clean;
            client_body_buffer_size    128K;
            client_max_body_size       0;
    
            proxy_pass_request_headers on;
            proxy_set_header           host        $host;
            proxy_set_header           x-tempfile  $request_body_file;
            proxy_set_header           x-filename  $http_x_filename;
            proxy_set_header           x-tag       $http_x_tag;
            proxy_set_header           x-size      $http_x_size;
            proxy_set_header           x-useragent $http_user_agent;
            proxy_set_header           x-checksum  $http_x_checksum;
            proxy_set_header           x-client    $remote_addr;
            proxy_set_body             off;
            proxy_redirect             off;
    
            # A high timeout is necessary to let the backend calculate the
            # checksum and move the (potentially large) file into place.
            proxy_read_timeout         3600;
            proxy_pass                 http://filebin.net/callback-upload;
        }
    
        location /overview {
            try_files $uri @filebin;
            allow   1.2.3.4;
            deny    all;
        }
    
        location /monitor {
            try_files $uri @filebin;
            allow   1.2.3.4;
            deny    all;
        }

        # Try to serve the file if the file exists, or let the web application
        # handle the request.
        location / { try_files $uri @filebin; }
        location @filebin {
            include uwsgi_params;
            uwsgi_param                REMOTE_ADDR $remote_addr;
            uwsgi_param                REMOTE_PORT $remote_port;
            uwsgi_read_timeout         6000;
            uwsgi_send_timeout         6000;
            client_max_body_size       1024M;
            client_body_buffer_size    128k;
            uwsgi_pass unix:/run/shm/filebin.sock;
        }
    }

Uwsgi (filebin.yaml)
--------------------
    uwsgi:
        uid: nginx
        gid: nginx
        socket: /run/shm/filebin.sock
        post-buffering: 0
        plugins: http,python
        processes: 15
        module: filebin
        callable: app
        chdir: /srv/www/filebin/app/
        no-orphans: true
        log-date: true
        master: true

Periodic tasks
--------------
http://filebin.net/maintenance will generate thumbnails and remove expired tags for you. Configure a cron job to GET this URL periodically to ensure that these tasks are being executed. It will return 200 if the tasks are executed successfully.

TODO
====
* Statistics in the admin interface.
* Proper MD5 checksumming during upload.
* Client side error handling.
* Fix the blocking feature which is broken now, or even add a remove by request feature to let users knowing tags remove them.

