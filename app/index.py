#!/usr/bin/python
from wsgiref.handlers import CGIHandler
from filebin import app

CGIHandler().run(app)
