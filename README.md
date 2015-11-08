[![Build Status](https://travis-ci.org/espebra/filebin.svg)](https://travis-ci.org/espebra/filebin)

# Requirements

To build ``filebin``, a Golang build environment and some Golang packages are needed.

When ``filebin`` has been built, it doesn't have any specific requirements to run. It even comes with its own web server bundled.

It is recommended but not required to run it behind a TLS/SSL proxy such as [Hitch](http://hitch-tls.org/) and web cache such as [Varnish Cache](https://www.varnish-cache.org/).

# Installation

Install ``golang``:

```
$ sudo yum/apt-get/brew install golang
```

Create the Go workspace and set the ``GOPATH`` environment variable:

```
$ mkdir ~/go
$ cd ~/go
$ mkdir src bin pkg
$ export GOPATH="~/go"
```

Download and install ``filebin``. The binary will be created as ``~/go/bin/filebin``.

```
$ go get github.com/espebra/filebin
$ cd src/github.com/espebra/filebin
$ go install
```

Create the directories to use for storing files, logs and temporary files:

```
$ mkdir ~/filebin ~/filebin/files ~/filebin/logs ~/filebin/temp
```

# Usage

The built in help text will show the various command line arguments available:

```
~/go/bin/filebin --help
```

Some arguments commonly used to start ``filebin`` are:

```
~/go/bin/filebin --verbose \
  --host 0.0.0.0 --port 31337
  --baseurl http://api.example.com:31337
  --filedir ~/filebin/files \
  --logdir ~/filebin/logs \
  --tempdir ~/filebin/temp \
  --expiration 604800
```

By default, ``filebin`` will listen on ``127.0.0.1:31337``.

## Baseurl

The ``baseurl`` parameter is used when building [HATEOAS](https://en.wikipedia.org/wiki/HATEOAS) links.

An example when having a TLS/SSL proxy in front on port 443 would be ``--baseurl https://filebin.example.com/``.

## Expiration

Tags expire after some time of inactivity. By default, tags will expire 3 months after the most recent file was uploaded. It is not possible to download files or upload more files to tags that are expired.

## Triggers

Triggers enable external scripts to be executed at certain events.

### Uploaded file

The parameter ``--trigger-uploaded-file /usr/local/bin/uploaded-file`` will make ``filebin`` execute ``/usr/local/bin/uploaded-file``, with the ``tag`` and ``filename`` as arguments for every file uploaded. The execution is non-blocking.

## Upload file

| **Method**		| POST			|
| **URL**		| /			|
| **URL parameters**	| *None*		|
| **Success response**	| ``201``		|
| **Error response**	| ``400``		|

### Examples

In all examples, the file ``/path/to/some file`` will be uploaded.

Using the following command, the ``tag`` will be automatically generated and the ``filename`` will be set to the SHA256 checksum of the content. The checksum of the content will not be verified.

```
$ curl --data-binary "@/path/to/some file" http://localhost:31337/
```

Using the following command, ``tag`` will be set to ``customtag`` and ``filename`` will be set to ``myfile``.

```
$ curl --data-binary "@/path/to/some file" http://localhost:31337/ \
  -H "tag: customtag" -H "filename: myfile"
```

Using the following command, ``filebin`` will verify the checksum of the uploaded file and discard the upload if the checksum does not match the specified checksum:

```
$ curl --data-binary "@/path/to/some file" http://localhost:31337/ \
  -H "tag: customtag" -H "filename: myfile" \
  -H "content-sha256: 82b5f1d5d38641752d6cbb4b80f3ccae502973f8b77f1c712bd68d5324e67e33"
```

## Show tag

| **Method**		| GET			|
| **URL**		| /:tag			|
| **URL parameters**	| *None*		|
| **Success response**	| ``200``		|
| **Error response**	| ``404``		|

### Examples

The following command will print a JSON structure showing which files that available in the tag ``customtag``.

```
$ curl http://localhost:31337/customtag
```

## Download file

| --------------------- | ----------------------|
| **Method**		| GET			|
| **URL**		| /:tag/:filename	|
| **URL parameters**	| *None*		|
| **Success response**	| ``200``		|
| **Error response**	| ``404``		|

### Examples

Downloading a file is as easy as specifying the ``tag`` and the ``filename`` in the request URI:

```
$ curl http://localhost:31337/customtag/myfile
```

# Roadmap

* Automatically clean up expired tags.
* Support for reporting/moderating/blocking tags.
* Support for deleting single files from tags.
* Web interface.
* Administrator dashboard.

