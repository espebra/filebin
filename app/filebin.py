#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Copyright (C) 2013 Espen Braastad
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU Affero General Public License as
# published by the Free Software Foundation, either version 3 of the
# License, or (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU Affero General Public License for more details.
#
# You should have received a copy of the GNU Affero General Public License
# along with this program.  If not, see <http://www.gnu.org/licenses/>.

import os
import re
import sys
import math
import time
import json
import magic
import fcntl
import select
import shutil
import random
import httplib
import hashlib
import pymongo
import datetime
import tempfile
import mimetypes
import string

import PythonMagick
import pyexiv2

import subprocess

import flask
import werkzeug

# https://code.google.com/p/python-geoip/
import pygeoip

# Import smtplib for the actual sending function
import smtplib
from email.mime.text import MIMEText


#app.config.from_envvar('FILEBIN_SETTINGS')
app = flask.Flask(__name__)

# Load app defaults
app.config.from_pyfile('../conf/application.cfg')

# Load the local.cfg if it exists (silent=True)
app.config.from_pyfile('/etc/filebin/local.cfg', silent=True)

# Load the instance spesific local.cfg if it exists (silent=True)
app.config.from_pyfile('../conf/local.cfg', silent=True)

app.debug = True

# Generate tag
def generate_tag():
    chars = string.ascii_lowercase + string.digits
    length = 10
    return ''.join(random.choice(chars) for _ in xrange(length))

# Generate passphrase
def generate_key():
    chars = string.ascii_letters + string.digits
    length = 30
    return ''.join(random.choice(chars) for _ in xrange(length))

def get_pages_for_tag(tag):
    per_page = int(app.config['FILES_PER_PAGE'])
    num_files = len(get_files_in_tag(tag))
    pages = int(math.ceil(num_files / round(per_page)))
    return pages

    if files:
        files = get_files_in_tag(tag)
        for f in files:
            filename = f['filename']

# Generate path to save the file
def get_path(tag = False, filename = False, thumbnail = False):

    # Use two levels of directories, just for, eh, scalability
    #m = re.match('^(.)(.)',tag)
    #a = m.group(1)
    #b = m.group(2)

    # Make sure the filename is safe
    if filename:
        filename = werkzeug.utils.secure_filename(filename)

    if thumbnail == True:
        path = '%s/%s' % (app.config['THUMBNAIL_DIRECTORY'],tag)

        if filename:
            #path = '%s/%s-thumb.jpg' % (path,filename)
            path = '%s/%s' % (path,filename)

    else:
        path = '%s/%s' % (app.config['FILE_DIRECTORY'],tag)

        if filename:
            path = '%s/%s' % (path,filename)

    return str(path)

# Function to calculate the md5 checksum for a file on the local file system
def md5_for_file(target):
    md5 = hashlib.md5()
    with open(target,'rb') as f:
        for chunk in iter(lambda: f.read(128*md5.block_size), b''):
            md5.update(chunk)

    f.close()
    return md5.hexdigest()

# A simple log function. Might want to inject to database and/or syslog instead
def log(priority,text):
    try:
        f = open(app.config['LOGFILE'], 'a')

    except:
        pass

    else:
        time = datetime.datetime.utcnow().strftime("%Y-%m-%d %H:%M:%S")
        if f:
            f.write("%s %s : %s\n" % (time, priority, text))
            f.close()

# Input validation
# Verify the flask.request. Return True if the flask.request is OK, False if it isn't.
def verify(tag = False, filename = False):

    illegal_tags = ['thumbnails','files','static','temp','app','conf', \
                    'lost+found']
    if tag:
        # We want to have a long tag
        if len(tag) < 10:
            return False

        # Only known chars are allowed in the tag
        if len(tag) >= 10 and len(tag) < 100:
            m = re.match('^[a-zA-Z0-9]+$',tag)
            if not m:
                return False

        # Verify that the given tag is not in the blacklist
        if tag in illegal_tags:
            return False

    if filename:
        # We want to have a valid length
        if len(filename) < 1:
            return False

        if len(filename) > 200:
            return False

    return True

def get_tags():
    tags = []

    col = dbopen('tags')
    try:
        cursor = col.find()

    except:
        cursor = False

    if cursor:
        for t in cursor:
            tags.append(t['_id'])

    return tags

def get_public_tags():
    tags = []

    col = dbopen('tags')
    try:
        cursor = col.find({'expose' : 'public'})

    except:
        cursor = False

    if cursor:
        for t in cursor:
            tag = t['_id']
            files = get_files_in_tag(tag)
            if len(files) > 0:
                tags.append(tag)

    return tags

def get_files_in_tag(tag, page = False, per_page = app.config['FILES_PER_PAGE']):
    files = []

    if not verify(tag):
        return files

    conf = get_tag_configuration(tag)

    col = dbopen('files')
    try:
        if page == False:
            cursor = col.find({'tag' : tag}, sort = [('captured',1),('filename',1)])
        else:
            skip = (int(page)-1) * per_page
            cursor = col.find({'tag' : tag},skip = skip, limit = per_page, sort = [('captured',1),('filename',1)])

    except:
        cursor = False

    if cursor:
        for f in cursor:
            filename = f['filename']
            i = {}
            i['filename'] = f['filename']
            #i['downloads'] = int(f['downloads'])
            i['mimetype'] = f['mimetype']

            if 'md5sum' in f:
                i['md5'] = f['md5sum']
            if 'checksum' in f:
                i['md5'] = f['checksum']

            i['filepath'] = get_path(tag,filename)
            i['size_bytes'] = f['size']
            i['size'] = "%.2f" % (f['size'] / 1024 / round(1024))
            #i['bandwidth'] = "%.2f" % ((f['downloads'] * f['size']) / 1024 / round(1024))
            i['uploaded'] = f['uploaded']
            i['client'] = f['client']
            i['uploaded_iso'] = datetime.datetime.strptime(str(f['uploaded']), \
                                    "%Y%m%d%H%M%S")

            if 'captured' in f:
                try:
                    i['captured_iso'] = str(datetime.datetime.strptime( \
                                          str(f['captured']), "%Y%m%d%H%M%S"))
                except:
                    pass

            # Add thumbnail path if the tag should show thumbnails and the
            # thumbnail for this filename exists.
            if conf['preview'] == 'on':
                thumbfile = get_path(tag,filename,True)
                if os.path.exists(thumbfile):
                    i['thumbnail'] = True

            #files[filename] = i 
            files.append(i)

    return files

def get_header(header):
    value = False

    if os.environ:
        m = re.compile('%s$' % header, re.IGNORECASE)
        header = string.replace(header,'-','_')
        for h in os.environ:
            if m.search(h):
                 value = os.environ[h]

    if not value:
        try:
            value = flask.request.headers.get(header)

        except:
            pass
    if value:
        log("DEBUG","Header %s = %s" % (header,value))
    else:
        log("DEBUG","Header %s was NOT FOUND" % (header))

    return value

# Detect the client address here
def get_client():
    client = False

    try:
        client = flask.request.headers.get('x-forwarded-for')

    except:
        pass

    if client == False or client == None:
        try:
            client = flask.request.headers.get('x-client')

        except:
            pass

    if client == False or client == None:
        try:
            client = flask.request.environ['REMOTE_ADDR']

        except:
            pass

    return client

def dbopen(collection):
    dbhost = app.config['DBHOST']
    dbport = app.config['DBPORT']
    db = app.config['DBNAME']
    # Connect to mongodb
    try:
        connection = pymongo.Connection(dbhost,dbport)

    except:
        log("ERROR","Unable to connect to database server " \
            "at %s:%s" % (dbhost,dbport))
        return False

    # Select database
    try:
        database = connection[db]

    except:
        log("ERROR","Unable to select to database %s " \
            "at %s:%s" % (db,dbhost,dbport))
        return False

    # Select collection
    try:
        col = database[collection]

    except:
        log("ERROR","Unable to select to collection %s " \
            "in database at %s:%s" % (collection,db,dbhost,dbport))
        return False

    # Return collection handler
    return col

def authenticate_key(tag,key):
    col = dbopen('tags')
    try:
        configuration = col.find_one({'_id' : tag, 'key' : key})

    except:
        return False

    if configuration:
        return True

    return False

def read_tag_creation_time(tag):
    col = dbopen('tags')
    try:
        t = col.find_one({'_id' : tag})

    except:
        return False

    try:
        t['registered']

    except:
        return False

    else:
        return t['registered']

    return False

def generate_thumbnails(tag):
    conf = get_tag_configuration(tag)

    if not conf['preview'] == 'on':
        return True

    thumbnail_dir = get_path(tag, thumbnail = True)
    if not os.path.exists(thumbnail_dir):
        os.makedirs(thumbnail_dir)
        if not os.path.exists(thumbnail_dir):
            log("ERROR","Unable to create directory %s for tag %s" % \
                (thumbnail_dir,tag))
            return False

    files = get_files_in_tag(tag)
    m = re.compile('^image/(jpeg|jpg|png|gif)')
    for f in files:
        filename = f['filename']

        try:
            mimetype = f['mimetype']

        except:
            log("DEBUG","Unable to read mimetype for tag %s, filename %s" \
                % (tag, filename))

        else:
            if mimetype:
                if m.match(mimetype):
                    # Decide the name of the files here
                    thumbfile = get_path(tag,filename,True)
                    filepath = get_path(tag,filename)

                    # TODO: Should also check if filepath is newer than 
                    # thumbfile!
                    if not os.path.exists(thumbfile):
                        log("DEBUG","Create thumbnail (%s) of file (%s)" \
                            % (thumbfile,filepath))

                        try:
                            im = PythonMagick.Image(filepath)
                        except:
                            log("ERROR","Unable to load file %s in tag %s " \
                                "with mimetype %s to generate thumbnail image" \
                                % (filename,tag,mimetype))
                        else:
                            try:
                                im.scale('%dx%d' % ( \
                                    app.config['THUMBNAIL_WIDTH'], \
                                    app.config['THUMBNAIL_HEIGHT']))
                            except:
                                log("ERROR","Unable to scale image %s in " \
                                    "tag %s with mimetype %s to generate " \
                                    "thumbnail image" \
                                    % (filename,tag,mimetype))
                            else:
                                try:
                                    im.write(str(thumbfile))
                                except:
                                    log("ERROR","Unable to write thumbnail " \
                                        "image of %s in " \
                                        "tag %s with mimetype %s to %s " \
                                        % (filename,tag,mimetype, \
                                        str(thumbfile)))
                                else:
                                    log("INFO","Generated thumbnail for " \
                                        "file %s in tag %s with mimetype %s" \
                                        % (filename,tag,mimetype))

def get_tag_lifetime(tag):
    days = False
    conf = get_tag_configuration(tag)

    registered = datetime.datetime.strptime(str(conf['registered']), \
                                            "%Y%m%d%H%M%S")
    now = datetime.datetime.utcnow()
    ttl = int(conf['ttl'])
    if ttl == 0:
        # Expire immediately
        to = now
    elif ttl == 1:
        # One week from registered
        to = registered + datetime.timedelta(weeks = 1)
    elif ttl == 2:
        # One month from registered
        to = registered + datetime.timedelta(weeks = 4)
    elif ttl == 3:
        # Six months from registered
        to = registered + datetime.timedelta(weeks = 26)
    elif ttl == 4:
        # One year from registered
        to = registered + datetime.timedelta(weeks = 52)
    elif ttl == 5:
        # Forever
        to = now + datetime.timedelta(weeks = 52)
    if int(to.strftime("%Y%m%d%H%M%S")) > int(now.strftime("%Y%m%d%H%M%S")):
        # TTL not reached
        if ttl == 5:
            # Forever, will never expire
            days = -1
        else:
            # Will expire some day
            diff = to - now
            days = int(diff.days)

    else:
        # Tag should be removed
        # TTL reached
        days = False

    return days

def get_log_days(tag = False):
    d = []
    col = dbopen('log')
    try:
        f = {}
        if tag:
            f['tag'] = tag

        entries = col.find(f).sort('time',-1)

    except:
        return d

    try:
        entries

    except:
        return d

    else:
        for entry in entries:
           if 'year' in entry and 'month' in entry and 'day' in entry:
               year = '%04d' % int(entry['year'])
               month = '%02d' % int(entry['month'])
               day = '%02d' % int(entry['day'])
               date = '%s-%s-%s' % (year,month,day)

               if not date in d:
                   d.append(date)

    return d

def get_country(ip):
    geoip = False

    if not ip:
        return False

    if ip.find(".") != -1:
        # ipv4
        geoip = pygeoip.GeoIP('GeoIP.dat', pygeoip.MEMORY_CACHE)

    elif ip.find(":") != -1:
        # ipv6
        geoip = pygeoip.GeoIP('GeoIPv6.dat', pygeoip.MEMORY_CACHE)

    if geoip:
        try:
            country = geoip.country_code_by_addr(ip)
            return country

        except:
            pass

    return False

def get_log(year = False,month = False,day = False,tag = False):
    ret = []
    col = dbopen('log')
    try:
        f = {}
        if year:
            f['year'] = int(year)

        if month:
            f['month'] = int(month)

        if day:
            f['day'] = int(day)

        if tag:
            f['tag'] = tag

        entries = col.find(f).sort('time',-1)

    except:
        return ret

    try:
        entries

    except:
        return ret

    else:
        for entry in entries:
           l = {}
           l['time']      = datetime.datetime.strptime(str(entry['time']),"%Y%m%d%H%M%S")

           if 'description' in entry:
               l['description'] = entry['description']

           if 'client' in entry:
               l['client'] = entry['client']
               try:
                   l['country'] = get_country(l['client'])
               except:
                   pass

           if 'tag' in entry:
               l['tag'] = entry['tag']

           if 'referer' in entry:
               l['referer'] = entry['referer']

           if 'useragent' in entry:
               l['useragent'] = entry['useragent']

           if 'filename' in entry:
               l['filename']  = entry['filename']

           ret.append(l)

    return ret

def get_tag_configuration(tag):
    col = dbopen('tags')
    try:
        configuration = col.find_one({'_id' : tag})

    except:
        return False

    try:
        configuration

    except:
        return False

    else:
        return configuration

    return False

def get_hostname():
    try:
        hostname = os.environ['HTTP_HOST']

    except:
        hostname = False

    return hostname

def get_last(count = False, files = False, tags = False, reports = False):

    if count:
        count = int(count)

    ret = []
    if files == True:
        col = dbopen('files')
        if count:
            cursor = col.find().sort('uploaded',-1).limit(count)

        else:
            cursor = col.find().sort('uploaded',-1)

        for entry in cursor:
           l = {}
           l['time'] = datetime.datetime.strptime(str(entry['uploaded']),"%Y%m%d%H%M%S")

           l['tag'] = entry['tag']
           l['filename'] = entry['filename']
           l['mimetype'] = entry['mimetype']
           l['downloads'] = entry['downloads']
           l['client'] = entry['client']

           try:
               l['country'] = get_country(l['client'])
           except:
               pass

           ret.append(l)

    if reports == True:
        col = dbopen('reports')

        if count:
            cursor = col.find().sort('time',-1).limit(count)

        else:
            cursor = col.find().sort('time',-1)

        for entry in cursor:
           l = {}
           tag = entry['tag']
           l['time'] = datetime.datetime.strptime(str(entry['time']),"%Y%m%d%H%M%S")
           l['tag'] = tag
           l['client'] = entry['client']
           l['reason'] = entry['reason']

           try:
               l['country'] = get_country(l['client'])
           except:
               pass

           ret.append(l)

    if tags == True:
        col = dbopen('tags')

        if count:
            cursor = col.find().sort('registered',-1).limit(count)

        else:
            cursor = col.find().sort('registered',-1)

        for entry in cursor:
           l = {}
           tag = entry['_id'] 
           l['time'] = datetime.datetime.strptime(str(entry['registered']),"%Y%m%d%H%M%S")

           l['tag'] = tag
           l['ttl'] = entry['ttl']

           if 'client' in entry:
               l['client'] = entry['client']

               try:
                   l['country'] = get_country(l['client'])
               except:
                   pass

           try:
               l['days_left'] = get_tag_lifetime(tag)

           except:
               l['days_left'] = False

           files = get_files_in_tag(tag)
           l['files'] = len(files)

           ret.append(l)
    return ret

def hash_key(key):
    # Let's hash the admin key
    m = hashlib.sha512()
    m.update(key)
    return m.hexdigest()

def add_file_to_database(i):
    status = False

    now = datetime.datetime.utcnow()
    i['downloads'] = 0
    i['uploaded']  = now.strftime("%Y%m%d%H%M%S")

    col = dbopen('files')
    try:
        col.update({
                     'tag'         : i['tag'],
                     'filename'    : i['filename']
                   },
                   i,
                   True)

    except:
        log("ERROR","Unable to add file %s in tag %s to database" \
            % (i['filename'],i['tag']))

    else:
        status = True
    return status

def create_default_tag_configuration(tag,key):
    now = datetime.datetime.utcnow()
    status = False

    hashed_key = hash_key(key)

    col = dbopen('tags')
    try:
        col.update({'_id'          : tag},
                   {
                     '_id'         : tag,
                     'key'         : hashed_key,
                     'ttl'         : 2,
                     'expose'      : 'private',
                     'client'      : get_client(),
                     'permission'  : 'rw',
                     'preview'     : 'on',
                     'registered'  : now.strftime("%Y%m%d%H%M%S")
                   },
                   True)

    except:
        log("ERROR","Unable to create default configuration for " \
            "tag %s." % (tag))

    else:
        status = True

    return status

def verify_admin_request(req):
    try:
        ttl = int(req.form['ttl'])
        expose = req.form['expose']
        preview = req.form['preview']
        permission = req.form['permission']

    except:
        return False

    if ttl < 0 or ttl > 5:
        return False

    if expose != 'private' and expose != 'public':
        return False

    if preview != 'on' and preview != 'off':
        return False

    if permission != 'ro' and permission != 'rw':
        return False

    return True

# Increment download counter
def increment_download_counter(tag,filename):
    col = dbopen('files')
    try:
        col.update({
                     'tag'         : tag,
                     'filename'    : filename
                   },
                   {
                     '$inc' : {
                       'downloads' : 1
                     }
                   },
                   True)

    except:
        log("ERROR","Unable to increment download counter for " \
            "%s in %s" % (filename,tag))


def send_email(subject,body,to = app.config['EMAIL']):
    try:
        me = app.config['FROM_EMAIL']
        you = to
        msg = MIMEText(body)

        msg['Subject'] = subject
        msg['From'] = me
        msg['To'] = you

        s = smtplib.SMTP(app.config['SMTPHOST'])
        s.set_debuglevel(1)
        s.sendmail(me, [you], msg.as_string())
        s.quit()

    except:
        pass

def clean_log():
    col = dbopen('log')
    days = int(app.config['NUMBER_OF_DAYS_TO_KEEP_LOGS'])
    dt = datetime.datetime.now() - datetime.timedelta(days = days)

    d =  {
             'time' : {
                 '$lt' : int(dt.strftime("%Y%m%d%H%M%S"))
             }
         }

    col.remove(d)

def dblog(description,client = False,tag = False,filename = False):
    referer = get_header('referer')
    useragent = get_header('user-agent')

    now = datetime.datetime.utcnow()
    col = dbopen('log')
    try:
        i = {}
        i['time'] = int(now.strftime("%Y%m%d%H%M%S"))
        i['year'] = int(now.strftime("%Y"))
        i['month'] = int(now.strftime("%m"))
        i['day'] = int(now.strftime("%d"))

        if client:
            i['client'] = client

        if tag:
            i['tag'] = tag

        if filename:
            i['filename'] = filename

        if referer:
            i['referer'] = referer

        if useragent:
            i['useragent'] = useragent

        i['description'] = description
        col.insert(i)


    except:
        log("ERROR","Unable to log %s of file %s to tag %s and client %s" \
            % (direction,filename,tag,client))

def unblock_tag(tag):
    client = get_client()
    dblog("Unblocking tag %s" % (tag), client, tag)
    col = dbopen('tags')
    try:
        col.update({'_id' : tag},
                   {
                     '$set' : {
                       'blocked' : False
                     }
                   },
                   False)

    except:
        log("ERROR","Unable to update configuration for " \
            "tag %s." % (tag))
        return False

    else:
        return True

def block_tag(tag):
    client = get_client()
    dblog("Blocking tag %s" % (tag), client, tag)
    col = dbopen('tags')
    try:
        col.update({'_id' : tag},
                   {
                     '$set' : {
                       'blocked' : True
                     }
                   },
                   False)

    except:
        log("ERROR","Unable to update configuration for " \
            "tag %s." % (tag))
        return False

    else:
        return True

def get_mimetype(path):
    m = magic.open(magic.MAGIC_MIME_TYPE)
    m.load()
    mimetype = m.file(path)
    return mimetype

def delete_file_from_db(tag,filename):
    col = dbopen('files')
    try:
        col.remove({
                    'filename' : filename,
                    'tag' : tag
                   },
                   False)

    except:
        log("ERROR","%s: Unable to remove file %s" % (tag,filename))
        return False

    else:
        log("INFO","%s: File %s removed from the database" % (tag,filename))
        return True

def delete_file(path):
    # Remove the file from the file system
    if os.path.exists(path):
        try:
            os.remove(path)

        except:
            log("ERROR","Unable to remove %s from the filesystem" % (path))
            return False

        else:
            log("INFO","The file %s was successfully removed." % (path))
            return True
    else:
        log("INFO","The file %s does not exist. No need to remove." % (path))
        return True

def get_time_of_capture(path):
    version = False
    time = False
    try:
        v = pyexiv2.version_info

    except:
        v = False

    if v:
        version = "%d%d%d" % (v[0],v[1],v[2])

    if int(version) >= 032:
        time = get_datetime2(path)
    else:
        time = get_datetime(path)

    return time

def get_datetime(path):
    time = False
    ret = False

    try:
        image = pyexiv2.Image(path)
    except:
        log("ERROR","EXIF: Unable to load image %s" % (path))
    else:
        try:
            image.readMetadata()
        except:
            log("ERROR","EXIF: Unable to load metadata from %s" % (path))
        else:
            try:
                time = str(image['Exif.Photo.DateTimeOriginal'])
            except:
                try:
                    time = str(image['Exif.Photo.DateTimeOriginal'])
                except:
                    log("ERROR", "EXIF: Unable to find DateTime and " \
                         "DateTimeOriginal in %s" % (path))

    if time:
        log("DEBUG","EXIF: DateTime = %s for %s" % (time,path))
        try:
            time_dt = pyexiv2.StringToDateTime(time)

        except:
            log("ERROR","EXIF: Unable to convert DateTime from string to " \
                "datetime")
            ret = time

        else:
            ret = time_dt
    return ret

def get_datetime2(path):
    time = False

    try:
        image = pyexiv2.ImageMetadata(path)
    except:
        log("ERROR","EXIF: Unable to load image %s" % (path))
    else:
        try:
            image.read()
        except:
            log("ERROR","EXIF: Unable to load metadata from %s" % (path))
        else:
            try:
                time = image['Exif.Photo.DateTimeOriginal'].value
            except:
                try:
                    time = image['Exif.Photo.DateTimeOriginal'].value
                except:
                    log("ERROR", "EXIF: Unable to find DateTime and " \
                         "DateTimeOriginal in %s" % (path))

    return time

def remove_tag(tag):
    status = True

    # Remove from the database
    col = dbopen('tags')
    try:
        col.remove({'_id' : tag})

    except:
        log("ERROR","%s: Unable to remove tag from mongodb/tags" % (tag))
        status = False

    else:
        log("INFO","%s: Removed tag from mongodb/tags" % (tag))

    col = dbopen('files')
    try:
        col.remove({'tag' : tag})

    except:
        log("ERROR","%s: Unable to remove tag from mongodb/files" % (tag))
        status = False

    else:
        log("INFO","%s: Removed tag from mongodb/files" % (tag))

    thumbdir = get_path(tag,thumbnail = True)
    if os.path.exists(thumbdir):
        try:
            shutil.rmtree(thumbdir)

        except:
            log("ERROR","%s: Unable to remove thumbnail files (%s)" % \
                (tag,thumbdir))
            status = False

        else:
            log("INFO","%s: Removed thumbnail files (%s) for tag" % \
                (tag,thumbdir))

    else:
        log("INFO","%s: Thumbnail directory (%s) does not exist" % \
            (tag,thumbdir))

    filedir = get_path(tag)
    if os.path.exists(filedir):
        try:
            shutil.rmtree(filedir)

        except:
            log("ERROR","%s: Unable to remove files (%s)" % (tag,filedir))
            status = False

        else:
            log("INFO","%s: Removed files (%s) for tag" % (tag,filedir))

    return status

@app.route("/overview/dashboard")
@app.route("/overview/dashboard/")
def dashboard():
    data = {}
    data['totals'] = {}
    size = 0
    downloads = 0
    bandwidth = 0

    data['uploads'] = get_last(10,files = True)
    data['tags'] = get_last(10,tags = True)
    data['reports'] = get_last(10,reports = True)

    tags = get_tags()
    data['totals']['tags'] = len(tags)
    data['totals']['files'] = 0
    for tag in tags:
        files = get_files_in_tag(tag)
        for f in files:
            data['totals']['files'] += 1
            size += int(f['size_bytes']) / 1024 / 1024 / round(1024)

    data['totals']['size'] = '%.2f' % (size)

    response = flask.make_response( \
        flask.render_template("overview_dashboard.html", \
        data = data, active = 'dashboard', title = "Dashboard"))
    response.headers['cache-control'] = 'max-age=0, must-revalidate'
    return response

@app.route("/overview/")
@app.route("/overview")
def overview():
    return flask.redirect('/overview/dashboard')

@app.route("/overview/log/")
@app.route("/overview/log")
def overview_log():
    now = datetime.datetime.utcnow()
    year = '%04d' % int(now.strftime("%Y"))
    month = '%02d' % int(now.strftime("%m"))
    day = '%02d' % int(now.strftime("%d"))
    return flask.redirect('/overview/log/%s-%s-%s' % (year,month,day))

@app.route("/overview/log/<date>/")
@app.route("/overview/log/<date>")
def overview_log_day(date):
    try:
       year = date[0:4]
       month = date[5:7]
       day = date[8:10]

    except:
       flask.abort(400)

    client = get_client()
    dblog("Show log overview", client = client)

    year = '%04d' % int(year)
    month = '%02d' % int(month)
    day = '%02d' % int(day)
    date = '%s-%s-%s' % (year,month,day)

    log = get_log(year,month,day)
    days = get_log_days()

    response = flask.make_response(flask.render_template("overview_log.html", \
        log = log, days = days, year = year, month = month, day = day, \
        active = 'logs', date = date, title = "Logs"))
    response.headers['cache-control'] = 'max-age=0, must-revalidate'
    return response

@app.route("/overview/tags/", methods = ['POST', 'GET'])
@app.route("/overview/tags", methods = ['POST', 'GET'])
def overview_tags():
    client = get_client()
    #dblog("Show tag overview", client = client)

    if flask.request.method == 'POST':
        try:
            tag = flask.request.form['tag']
            action = flask.request.form['action']

        except:
            pass

        else:
           if verify(tag):
                if action == 'block':
                    block_tag(tag)

                elif action == 'unblock':
                    unblock_tag(tag)

    reports = get_last(reports = True)

    tags = {}
    for t in get_tags():
        n = {}
        for d in reports:
            if d['tag'] == t:
                n['reported'] = True

        n['files'] = 0
        n['size'] = 0

        conf = get_tag_configuration(t)
        n['conf'] = conf

        files = get_files_in_tag(t)
        if files:
            n['files'] = len(files)
            for f in files:
                n['size'] += f['size_bytes'] / 1024 / float(1024)

        # Show only two decimals
        n['size'] = '%.2f' % n['size']

        # Only show the tags with files
        if n['files'] > 0:
            if not t in tags:
                tags[t] = {}

            tags[t] = n

    response = flask.make_response(flask.render_template("overview_tags.html", \
        tags = tags, active = 'tags', title = "Tags"))

    response.headers['cache-control'] = 'max-age=0, must-revalidate'
    return response

@app.route("/overview/map/")
@app.route("/overview/map")
def overview_map():
    client = get_client()
    dblog("Show map overview", client = client)

    files = {}
    countries = {}
    tags = get_tags()
    for tag in tags:
       f = get_files_in_tag(tag)
       for l in f:
           try:
               country = get_country(l['client'])
               if country:
                   if len(country) == 2:
                       if not country in countries:
                           countries[country] = 0

                       countries[country] += 1
           except:
               pass

    response = flask.make_response(flask.render_template("overview_map.html", countries = countries, active = 'map', title = "Map"))
    response.headers['cache-control'] = 'max-age=0, must-revalidate'
    return response

@app.route("/overview/files/")
@app.route("/overview/files")
def overview_files():
    client = get_client()
    dblog("Show files overview", client = client)

    files = {}
    tags = get_tags()
    for tag in tags:
       f = get_files_in_tag(tag)
       for l in f:
           try:
               l['country'] = get_country(l['client'])
           except:
               pass
       files[tag] = f

    response = flask.make_response(flask.render_template("overview_files.html", files = files, active = 'files', title = "Files"))
    response.headers['cache-control'] = 'max-age=0, must-revalidate'
    return response

@app.route("/monitor/")
@app.route("/monitor")
def monitor():
    tags = get_tags()
    num_files = 0
    size = 0
    for tag in tags:
        files = get_files_in_tag(tag)
        for f in files:
             num_files += 1
             size += f['size_bytes']

    response = flask.make_response(flask.render_template('monitor.txt', tags = tags, num_files = num_files, size = size))

    response.headers['status'] = '200'
    response.headers['content-type'] = 'text/plain'
    response.headers['cache-control'] = 'max-age=60, must-revalidate'

    return response

@app.route("/about/")
@app.route("/about")
def about():
    client = get_client()
    host = get_header('host')
    response = flask.make_response(flask.render_template("about.html", \
        title = "About", host = host))
    response.headers['cache-control'] = 'max-age=3600 must-revalidate'
    return response

@app.route("/")
def index():
    client = get_client()
    response = flask.make_response(flask.render_template("index.html", title = "Online storage at your fingertips"))
    response.headers['cache-control'] = 'max-age=3600, must-revalidate'
    return response

def report(tag):

    submitted = 0

    if not verify(tag):
        time.sleep(app.config['FAILURE_SLEEP'])
        flask.abort(400)

    if flask.request.method == 'POST':
        try:
            reason = flask.request.form['reason']

        except:
            flask.abort(400)

        else:
            client = get_client()
            subject = 'Filebin: Request to delete tag %s' % (tag)
            body = 'Good day,\n%s has just sent a request to delete tag %s. ' \
                   'The reason to why the tag should be deleted is:\n\n' \
                   '"%s"' % (client,tag,reason)

            send_email(subject,body)

            now = datetime.datetime.utcnow()
            col = dbopen('reports')
            try:
                i = {}
                i['time'] = now.strftime("%Y%m%d%H%M%S")
                i['year'] = int(now.strftime("%Y"))
                i['month'] = int(now.strftime("%m"))
                i['day'] = int(now.strftime("%d"))
                i['tag'] = tag
                i['reason'] = reason
                i['client'] = client
                col.insert(i)

            except:
                dblog("Failed to submit report", \
                    client = client, tag = tag)
                log("ERROR","Unable to add report, tag %s, client " \
                    "%s, reason %s" % (tag,client,reason))
                submitted = -1

            else:
                dblog("Tag %s reported" % tag, client = client, tag = tag)
                submitted = 1

    response = flask.make_response(flask.render_template("report.html", \
        tag = tag, submitted = submitted, \
        title = "Report tag %s" % (tag)))

    response.headers['cache-control'] = 'max-age=3600, must-revalidate'
    return response

@app.route("/<tag>/", methods = ['POST', 'GET'])
@app.route("/<tag>", methods = ['POST', 'GET'])
@app.route("/<tag>/page/<page>/", methods = ['POST', 'GET'])
@app.route("/<tag>/page/<page>", methods = ['POST', 'GET'])
def tag_page(tag,page = 1):

    if not verify(tag):
        time.sleep(app.config['FAILURE_SLEEP'])
        flask.abort(400)

    try:
        view = flask.request.args.get('v','')
    except:
        view = False

    try:
        key = flask.request.args.get('key','')
    except:
        key = False

    if view == 'playlist':
        return tag_playlist(tag)

    elif view == 'json':
        return tag_json(tag)

    elif view == 'report':
        return report(tag)

    elif view == 'configuration':
        return admin_configuration(tag,key)

    elif view == 'files':
        return admin_files(tag,key)

    elif view == 'map':
        return admin_map(tag,key)

    else:
        return tag_html(tag, page)

def tag_html(tag,page):
    files = {}

    conf = get_tag_configuration(tag)

    num_files = len(get_files_in_tag(tag))

    pages = get_pages_for_tag(tag)

    # Input validation
    try:
        int(page)

    except:
       flask.abort(400)

    page = int(page)
    if page < 1:
        page = 1

    if page > pages:
        page = pages

    #log("DEBUG","PAGES: Tag %s has %d files, which will be presented in %d pages with %d files per page" % (tag, num_files, pages, per_page))
    files = get_files_in_tag(tag,page)

    # By default, do not show captured at in the file listing
    datetime_found = False
    for f in files:
        if 'captured_iso' in f:
            datetime_found = True
            continue

    client = get_client()

    try:
        valid_days = get_tag_lifetime(tag)

    except:
        valid_days = False

    response = flask.make_response(flask.render_template("tag.html", \
        tag = tag, files = files, conf = conf, num_files = num_files, \
        pages = pages, page = page, valid_days = valid_days, \
        datetime_found = datetime_found, \
        title = "Tag %s" % (tag)))

    response.headers['cache-control'] = 'max-age=3600, must-revalidate'
    return response

def tag_playlist(tag):
    out = ""

    if not verify(tag):
        time.sleep(app.config['FAILURE_SLEEP'])
        flask.abort(400)

    conf = get_tag_configuration(tag)
    files = get_files_in_tag(tag)

    protocol = "http"
    host = get_header('host')

    for f in files:
        out += "%s://%s/%s/%s\n" % (protocol, host, tag, f['filename'])

    h = werkzeug.Headers()
    h.add('cache-control', 'max-age=7200, must-revalidate')
    return flask.Response(out, mimetype='text/plain', headers = h)

def tag_json(tag):
    files = {}

    if not verify(tag):
        time.sleep(app.config['FAILURE_SLEEP'])
        flask.abort(400)

    conf = get_tag_configuration(tag)
    files = get_files_in_tag(tag)

    for f in files:
        # Remove some unecessary stuff
        del(f['filepath'])
        del(f['uploaded_iso'])
        del(f['size'])
        f['tag'] = tag

    # Verify json format
    try:
        ret = json.dumps(files, indent=2)

    except:
        flask.abort(501)

    h = werkzeug.Headers()
    #h.add('Content-Disposition', 'inline' % (tag))
    h.add('cache-control', 'max-age=7200, must-revalidate')
    return flask.Response(ret, mimetype='text/json', headers = h)

def admin_files(tag,key):
    client = get_client()
    filename = False

    if not verify(tag):
        flask.abort(400)

    # Let's hash the admin key
    hashed_key = hash_key(key)

    if not authenticate_key(tag,hashed_key):
        time.sleep(app.config['FAILURE_SLEEP'])
        flask.abort(401)

    conf = get_tag_configuration(tag)

    status = 0
    if flask.request.method == 'POST':
        try:
            filename = flask.request.form['filename']
            action   = flask.request.form['action']

        except:
            status = -1

        else:
            if action == 'delete':
                filename = werkzeug.utils.secure_filename(filename)
                if verify(filename = filename):
                    # Remove the file from the file system
                    file_path = get_path(tag,filename)
                    log("INFO","%s: Will remove %s" % (tag,filename))
                    if delete_file(file_path):
                        status = 1
                        log("INFO","%s: The file %s was deleted by %s" % \
                            (tag,file_path,client))
                    else:
                        status = -1
                        log("ERROR","%s: Failed to remove the file %s from " \
                            "the file system." % (tag,file_path,client))

                    if status == 1:
                        # Make sure that the thumbnail is removed also.
                        thumb_path = get_path(tag,filename,thumbnail = True)
                        delete_file(thumb_path)

                        if delete_file_from_db(tag,filename):
                            dblog("File %s was deleted" % \
                                (filename), client, tag)

    files = get_files_in_tag(tag)

    response = flask.make_response(flask.render_template("admin_files.html", \
        tag = tag, key = key, files = files, conf = conf, active = 'files', \
        status = status, filename = filename, \
        title = "Administration for %s" % (tag)))
    response.headers['cache-control'] = 'max-age=0, must-revalidate'
    return response

def admin_map(tag,key):
    client = get_client()

    if not verify(tag):
        flask.abort(400)

    # Let's hash the admin key
    hashed_key = hash_key(key)

    if not authenticate_key(tag,hashed_key):
        time.sleep(app.config['FAILURE_SLEEP'])
        flask.abort(401)

    conf = get_tag_configuration(tag)

    files = get_files_in_tag(tag)
    countries = {}
    for l in files:
        try:
            country = get_country(l['client'])
            if country:
                if len(country) == 2:
                    if not country in countries:
                        countries[country] = 0

                    countries[country] += 1
        except:
            pass

    response = flask.make_response(flask.render_template("admin_map.html", \
        tag = tag, key = key, countries = countries, conf = conf, \
        active = 'map', title = "Administration for %s" % (tag)))
    response.headers['cache-control'] = 'max-age=0, must-revalidate'
    return response

def admin_configuration(tag,key):
    client = get_client()
    saved = 0

    if not verify(tag):
        flask.abort(400)

    # Let's hash the admin key
    hashed_key = hash_key(key)

    if not authenticate_key(tag,hashed_key):
        time.sleep(app.config['FAILURE_SLEEP'])
        flask.abort(401)

    ttl_iso = {}
    # When the tag was created (YYYYMMDDhhmmss UTC)
    registered = read_tag_creation_time(tag)

    try:
        registered_iso = datetime.datetime.strptime(str(registered),"%Y%m%d%H%M%S")

    except:
        registered_iso = "N/A"

    else:
        ttl_iso['oneweek']  = (registered_iso + datetime.timedelta(7)).strftime("%Y-%m-%d")
        ttl_iso['onemonth'] = (registered_iso + datetime.timedelta(30)).strftime("%Y-%m-%d")
        ttl_iso['sixmonths'] = (registered_iso + datetime.timedelta(182)).strftime("%Y-%m-%d")
        ttl_iso['oneyear']  = (registered_iso + datetime.timedelta(365)).strftime("%Y-%m-%d")

    if flask.request.method == 'POST':
        if not verify_admin_request(flask.request):
            time.sleep(failure_sleep)
            flask.abort(400)

        ttl        = int(flask.request.form['ttl'])
        expose     = flask.request.form['expose']
        permission = flask.request.form['permission']
        preview    = flask.request.form['preview']

        dblog("Save administration settings for tag %s" % (tag), client, tag)
        col = dbopen('tags')
        try:
            col.update({'_id' : tag},
                       {
                         '$set' : {
                           'ttl' : ttl,
                           'expose' : expose,
                           'client' : get_client(),
                           'permission' : permission,
                           'preview' : preview
                         }
                       },
                       False)

        except:
            log("ERROR","Unable to update configuration for " \
                "tag %s." % (tag))

        else:
            saved = 1

    else:
        dblog('Show administration settings for tag %s' % (tag), client, tag)

    conf = get_tag_configuration(tag)

    response = flask.make_response(flask.render_template("admin_configuration.html", \
        tag = tag, conf = conf, key = key, registered_iso = registered_iso, \
        ttl_iso = ttl_iso, saved = saved, active = 'configuration', \
        title = "Administration for %s" % (tag)))
    response.headers['cache-control'] = 'max-age=0, must-revalidate'
    return response

#def nonblocking(pipe, size):
#    f = fcntl.fcntl(pipe, fcntl.F_GETFL)
# 
#    if not pipe.closed:
#        fcntl.fcntl(pipe, fcntl.F_SETFL, f | os.O_NONBLOCK)
# 
#    if not select.select([pipe], [], [])[0]:
#        yield ""
# 
#    while True:
#        data = pipe.read(size)
#
#        ## Stopper på StopIteration, så på break
#        if len(data) == 0:
#            break

@app.route("/archive/<tag>/")
@app.route("/archive/<tag>")
def archive(tag):
    client = get_client()
    def stream_archive(files_to_archive):
        command = "/usr/bin/zip -j - %s" % (" ".join(files_to_archive))
        p = subprocess.Popen(command, stdout=subprocess.PIPE, shell = True, close_fds = True)
        f = fcntl.fcntl(p.stdout, fcntl.F_GETFL)

        while True:
            if not p.stdout.closed:
                fcntl.fcntl(p.stdout, fcntl.F_SETFL, f | os.O_NONBLOCK)

            if not select.select([p.stdout], [], [])[0]:
                yield ""

            data = p.stdout.read(4096)
            yield data
            if len(data) == 0:
                break

    if not verify(tag):
        time.sleep(app.config['FAILURE_SLEEP'])
        flask.abort(400)

    tag_path = get_path(tag)
    if not os.path.exists(tag_path):
        time.sleep(app.config['FAILURE_SLEEP'])
        flask.abort(404)

    log_prefix = '%s archive -> %s' % (tag,client)
    log("INFO","%s: Archive download request received" % (log_prefix))

    files = get_files_in_tag(tag)
    files_to_archive = []
    for f in files:
        filepath = get_path(tag,f['filename'])
        files_to_archive.append(filepath)
        #log("INFO","Zip tag %s, file path %s" % (tag,filepath))

    h = werkzeug.Headers()
    #h.add('Content-Length', '314572800')
    h.add('Content-Disposition', 'attachment; filename=%s.zip' % (tag))
    h.add('cache-control', 'max-age=7200, must-revalidate')
    return flask.Response(stream_archive(files_to_archive), mimetype='application/zip', headers = h, direct_passthrough = True)

@app.route("/upload/<tag>/")
@app.route("/upload/<tag>")
def upload_to_tag(tag):
    if not verify(tag):
        flask.abort(400)

    # Generate the administration only if the tag does not exist.
    key = False
    conf = get_tag_configuration(tag)

    host = get_header('host')

    if conf:
        # The tag is read only
        if conf['permission'] != 'rw':
            flask.abort(401)

    else:
        key = generate_key()
        create_default_tag_configuration(tag,key)

    response = flask.make_response(flask.render_template("upload.html", \
        tag = tag, key = key, host = host, \
        title = "Upload to tag %s" % (tag)))

    # Cannot have to long TTL here as it will show the link to the
    # administration interface.
    response.headers['cache-control'] = 'max-age=5, must-revalidate'
    return response

@app.route("/upload/")
@app.route("/upload")
def upload():
    tag = generate_tag()
    response = flask.redirect('/upload/%s' % (tag))
    response.headers['cache-control'] = 'max-age=0, must-revalidate'
    return response

@app.route("/download/", methods = ['POST', 'GET'])
@app.route("/download", methods = ['POST', 'GET'])
def download():
    if flask.request.method == 'POST':
        try:
            tag = flask.request.form['tag']
            if not verify(tag):
                tag = False

        except:
            tag = False

        if tag:
            return flask.redirect('/%s' % (tag))
        else:
            flask.abort(400)
    else:
        tags = get_public_tags()
        response = flask.make_response(flask.render_template("download.html" , tags = tags, title = "Download"))
        response.headers['cache-control'] = 'max-age=3600, must-revalidate'
        return response

#@app.route("/callback-upload", methods = ['GET', 'POST'])
@app.route("/callback-upload", methods = ['POST'])
def callback_upload():
    status = False
    start = time.time()

    i = {}
    i['client']            = get_client()
    i['tempfile']          = get_header('x-tempfile')
    i['tag']               = get_header('x-tag')
    i['useragent']         = get_header('x-useragent')
    i['reported_size']     = int(get_header('x-size'))
    i['reported_checksum'] = get_header('x-checksum')
    filename               = get_header('x-filename')

    if not verify(i['tag'],filename):
        flask.abort(400)

    # Use werkzeug to validate the filename
    try:
        i['filename'] = werkzeug.utils.secure_filename(filename)

    except:
        log("ERROR","%s: Unable to create a secure version of the filename" % \
            (filename))
        flask.abort(400)

    else:
        if filename != i['filename']:
            log("INFO","Filename '%s' was renamed to the secure version '%s'" \
                % (filename,i['filename']))

    log("INFO","Callback received [%s]." % (i))
 
    # The input values are to be trusted at this point
    # New flask.request from client
    log_prefix = '%s -> %s/%s' % (i['client'],i['tag'],i['filename'])

    conf = get_tag_configuration(i['tag'])
    if conf:
        # The tag is read only
        if conf['permission'] != 'rw':
            log("DEBUG","%s: The tag is read only, aborting." % log_prefix)
            flask.abort(401)

    log("INFO","%s: Upload request received, the file size is %d bytes. User " \
        "agent: %s" % (log_prefix,i['reported_size'],i['useragent']))

    # The final destination
    target_dir = get_path(i['tag'])

    if not os.path.exists(target_dir):
        try:
            os.makedirs(target_dir)

        except:
            log("ERROR","%s: Unable to create directory %s" % (\
                log_prefix,target_dir))
            flask.abort(501)

        else:
            log("DEBUG","%s: Directory %s created successfully." % \
                (log_prefix,target_dir))

    # Verify the md5 checksum here.
    i['checksum'] = md5_for_file(i['tempfile'])
    log("DEBUG","%s: Checksum on uploaded file: %s" % (log_prefix, i['checksum']))

    if i['reported_checksum'] == i['checksum']:
        log("DEBUG","%s: Checksum OK!" % (log_prefix))

    else:
        log("DEBUG","%s: Checksum mismatch! (%s != %s) (proper client side checksumming is not yet implemented)" % ( \
            log_prefix, i['reported_checksum'], i['checksum']))
        # TODO: Should abort here

    # Detect file type
    try:
        mimetype = get_mimetype(i['tempfile'])
    except:
        log("DEBUG","%s: Unable to detect mime type on %s" % \
            (log_prefix, i['tempfile']))
    else:
        if mimetype:
            i['mimetype'] = mimetype
            log("DEBUG","%s: Detected mime type %s on %s" % \
                (log_prefix, mimetype, i['tempfile']))
        else:
            log("DEBUG","%s: No mime type found in %s" % \
                (log_prefix, i['tempfile']))

    captured = False
    if mimetype:
         m = re.match('^image',mimetype)
         if m:
             captured_dt = get_time_of_capture(i['tempfile'])

             if captured_dt:
                 try:
                     captured = int(captured_dt.strftime("%Y%m%d%H%M%S"))
                 except:
                     captured = captured_dt

    if captured:
        i['captured'] = captured
        log("DEBUG","%s: Captured at %s" % (log_prefix, captured))

    if os.path.exists(i['tempfile']):
        log("DEBUG","%s: The temp file %s exists locally" % \
            (log_prefix,i['tempfile']))
    else:
        log("ERROR","%s: The temp file %s does not exist locally" % \
            (log_prefix,i['tempfile']))

    try:
        stat = os.stat(i['tempfile'])
    except:
        log("ERROR","%s: Unable to read size of temp file" % ( \
            log_prefix,i['tempfile']))
    else:
        i['size'] = int(stat.st_size)

        # Verify that the file size is more than 0 bytes
        if i['size'] == 0:
            log("ERROR","%s: The file %s was 0 bytes. Let's abort here." % \
                (log_prefix,i['tempfile']))
            flask.abort(400)

        # Verify that the file size is equal to the one reported by the client
        # before uploading.
        if i['size'] == i['reported_size']:
            log("DEBUG","%s: The uploaded file (%s) size is correct (%d bytes)." % \
                (log_prefix,i['tempfile'],i['size']))
        else:
            log("ERROR","%s: The uploaded file %s was %d bytes, but should " \
                "have been %d bytes. Aborting." % \
                (log_prefix,i['tempfile'],i['size'],i['reported_size']))
            flask.abort(400)

    i['filepath'] = get_path(i['tag'],i['filename'])
    log("DEBUG","%s: Will save the content to %s" % (log_prefix,i['filepath']))

    # Uploading to temporary file is complete. Will now copy the contents 
    # to the final target destination.
    try:
        shutil.move(i['tempfile'],i['filepath'])

    except:
        log("ERROR","%s: Unable to move tempfile (%s) to target " \
            "(%s)" % (log_prefix,i['tempfile'],i['filepath']))

    else:
        log("DEBUG","%s: Content moved from tempfile (%s) to " \
            "final destination (%s)" % (log_prefix,i['tempfile'], \
            i['filepath']))

        if not add_file_to_database(i):
            log("ERROR","%s: Unable to add file to database." % (log_prefix))

        else:
            # Log the activity
            text = 'Client %s uploaded %s/%s successfully, %s bytes, checksum %s' % (i['client'],i['tag'],i['filename'],i['size'],i['checksum'])
            dblog('%s' % (text), tag = i['tag'], client = i['client'], filename = i['filename'])
            log("INFO","%s" % (text))
            status = True

    elapsed = time.time() - start
    log("INFO","%s: Callback duration was %d seconds" % (log_prefix,elapsed))

    response = flask.make_response(flask.render_template('uploader.html'))

    if status:

        response.headers['status'] = '200'
    else:
        response.headers['status'] = '501'

    response.headers['content-type'] = 'text/plain'
    response.headers['cache-control'] = 'max-age=0, must-revalidate'

    return response

@app.route("/maintenance/")
@app.route("/maintenance")
def maintenance():
    # Fix indexes

    try:
        col = dbopen('log')
        col.create_index('time')

        col = dbopen('files')
        col.create_index('tag')
        col.create_index('uploaded')

    except:
        pass

    tags = get_tags()
    for tag in tags:
        valid_days = get_tag_lifetime(tag)
        if valid_days:
            #log("DEBUG","%s: TTL not reached (%d days left)" % (tag,valid_days))
            generate_thumbnails(tag)

        else:
            log("INFO","%s: TTL reached. Should be deleted." % (tag))

            # Remove from tags and files
            # Remove from filesystem
            if remove_tag(tag):
                log("INFO","%s: Removed." % (tag))
                dblog("Tag %s has been removed due to expiry." % (tag), \
                    tag = tag)

            else:
                log("ERROR","%s: Unable to remove." % (tag))
                dblog("Failed to remove tag %s. It has expired." % (tag), \
                    tag = tag)

    clean_log()

    response = flask.make_response(flask.render_template('maintenance.html', title = "Maintenance"))
    response.headers['cache-control'] = 'max-age=0, must-revalidate'
    return response

@app.route("/robots.txt")
def robots():
    response = flask.make_response(flask.render_template('robots.txt'))
    response.headers['content-type'] = 'text/plain'
    response.headers['cache-control'] = 'max-age=3600, must-revalidate'
    return response

if __name__ == '__main__':
    app.debug = True
    app.run(host='0.0.0.0')

