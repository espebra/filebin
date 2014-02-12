file { "sources.list":
  path => "/etc/apt/sources.list",
  source => "/vagrant/apt/sources.list",
}

exec { "apt-get update":
  command => "/usr/bin/apt-get update",
  require => [File["sources.list"]],
}

exec { "install requirements":
  command => "/usr/bin/apt-get -y install nginx mongodb-server python-pymongo python-jinja2 python-flask python-werkzeug python-magic python-pythonmagick python-pyexiv2 uwsgi uwsgi-plugin-python zip",
  require => Exec["apt-get update"],
}

file { "nginx_conf":
  path => "/etc/nginx/sites-available/filebin.conf",
  source => "/vagrant/nginx/nginx.conf",
  require => Exec["install requirements"],
}

file { "/etc/nginx/sites-enabled/filebin.conf":
  ensure => "link",
  target => "/etc/nginx/sites-available/filebin.conf",
  require => File["nginx_conf"],
}

file { "uwsgi_conf":
  path => "/etc/uwsgi/apps-available/filebin.yaml",
  source => "/vagrant/uwsgi/filebin.yaml",
  require => Exec["install requirements"],
}

file { "/etc/uwsgi/apps-enabled/filebin.yaml":
  ensure => "link",
  target => "/etc/uwsgi/apps-available/filebin.yaml",
  require => File["uwsgi_conf"],
}

exec { "restart_services":
  command => "/usr/sbin/service nginx restart && /usr/sbin/service uwsgi restart",
  require => [File["/etc/nginx/sites-enabled/filebin.conf"],
             File["/etc/uwsgi/apps-enabled/filebin.yaml"]
  ]
}
