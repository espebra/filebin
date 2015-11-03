[![Build Status](https://travis-ci.org/espebra/filebin.svg)](https://travis-ci.org/espebra/filebin)

# Installation

Install ``golang``:

```
yum/apt-get/brew install golang
```

Create the Go workspace and set the ``GOPATH`` environment variable:

```
mkdir ~/go
cd ~/go
mkdir src bin pkg
export GOPATH="~/go"
```

Download and install filebin:

```
go get github.com/espebra/filebin
cd src/github.com/espebra/filebin
go install
```

The binary will be created as ``~/go/bin/filebin``.

Create the filebin directories:

```
mkdir /srv/filebin/files /srv/filebin/logs /srv/filebin/temp
```

# Usage

```
~/go/bin/filebin --help
```

Quick start:

```
~/go/bin/filebin --filedir /srv/filebin/files --logdir /srv/filebin/logs --tempdir /srv/filebin/temp
```

By default, filebin will listen on ``127.0.0.1:31337``.


# Expiration

Tags expire after some time of inactivity. By default, tags will expire 3 months after the most recent file was uploaded. It is not possible to download files or upload more files to tags that are expired.

# API

## Upload file

In all examples, I will upload the file ``/path/to/file``.

Using the following command, the ``tag`` will be automatically generated and the ``filename`` will be set to the SHA256 checksum of the content. The checksum of the content will not be verified.

```
$ curl --data-binary @/path/to/file http://localhost:31337/
```

Using the following command, ``tag`` will be set to ``customtag`` and ``filename`` will be set to ``myfile``.

```
$ curl --data-binary @/path/to/file http://localhost:31337/ -H "tag: customtag" -H "filename: myfile"
```

## Show tag

The following command will print a JSON structure showing which files that available in the tag ``customtag``.

```
$ curl http://localhost:31337/customtag
```

## Download file

Downloading a file is as easy as specifying the ``tag`` and the ``filename`` in the request URI:

```
$ curl http://localhost:31337/customtag/myfile
```
