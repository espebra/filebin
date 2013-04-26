#!/usr/bin/env python

'''
Pure Python reader for GeoIP Country Edition databases.
'''

__author__ = 'David Wilson <dw@botanicus.net>'


import os
import struct

from cStringIO import StringIO


#
# Constants.
#

# From GeoIP.h.
SEGMENT_RECORD_LENGTH = 3
STANDARD_RECORD_LENGTH = 3
ORG_RECORD_LENGTH = 4
MAX_RECORD_LENGTH = 4
FULL_RECORD_LENGTH = 50
NUM_DB_TYPES = 20

GEOIP_COUNTRY_EDITION     = 1
GEOIP_REGION_EDITION_REV0 = 7
GEOIP_CITY_EDITION_REV0   = 6
GEOIP_ORG_EDITION         = 5
GEOIP_ISP_EDITION         = 4
GEOIP_CITY_EDITION_REV1   = 2
GEOIP_REGION_EDITION_REV1 = 3
GEOIP_PROXY_EDITION       = 8
GEOIP_ASNUM_EDITION       = 9
GEOIP_NETSPEED_EDITION    = 10
GEOIP_DOMAIN_EDITION      = 11
GEOIP_COUNTRY_EDITION_V6  = 12

COUNTRY_BEGIN = 16776960
STATE_BEGIN_REV0 = 16700000
STATE_BEGIN_REV1 = 16000000
STRUCTURE_INFO_MAX_SIZE = 20
DATABASE_INFO_MAX_SIZE = 100

GeoIP_country_code = '''
    AP EU AD AE AF AG AI AL AM AN AO AQ AR AS AT AU AW AZ BA BB BD BE BF BG BH
    BI BJ BM BN BO BR BS BT BV BW BY BZ CA CC CD CF CG CH CI CK CL CM CN CO CR
    CU CV CX CY CZ DE DJ DK DM DO DZ EC EE EG EH ER ES ET FI FJ FK FM FO FR FX
    GA GB GD GE GF GH GI GL GM GN GP GQ GR GS GT GU GW GY HK HM HN HR HT HU ID
    IE IL IN IO IQ IR IS IT JM JO JP KE KG KH KI KM KN KP KR KW KY KZ LA LB LC
    LI LK LR LS LT LU LV LY MA MC MD MG MH MK ML MM MN MO MP MQ MR MS MT MU MV
    MW MX MY MZ NA NC NE NF NG NI NL NO NP NR NU NZ OM PA PE PF PG PH PK PL PM
    PN PR PS PT PW PY QA RE RO RU RW SA SB SC SD SE SG SH SI SJ SK SL SM SN SO
    SR ST SV SY SZ TC TD TF TG TH TJ TK TM TN TO TL TR TT TV TW TZ UA UG UM US
    UY UZ VA VC VE VG VI VN VU WF WS YE YT RS ZA ZM ME ZW A1 A2 O1 AX GG IM JE
    BL MF
'''.split()

GeoIP_country_continent = '''
    AS EU EU AS AS SA SA EU AS SA AF AN SA OC EU OC SA AS EU SA AS EU AF EU AS
    AF AF SA AS SA SA SA AS AF AF EU SA NA AS AF AF AF EU AF OC SA AF AS SA SA
    SA AF AS AS EU EU AF EU SA SA AF SA EU AF AF AF EU AF EU OC SA OC EU EU EU
    AF EU SA AS SA AF EU SA AF AF SA AF EU SA SA OC AF SA AS AF SA EU SA EU AS
    EU AS AS AS AS AS EU EU SA AS AS AF AS AS OC AF SA AS AS AS SA AS AS AS SA
    EU AS AF AF EU EU EU AF AF EU EU AF OC EU AF AS AS AS OC SA AF SA EU AF AS
    AF NA AS AF AF OC AF OC AF SA EU EU AS OC OC OC AS SA SA OC OC AS AS EU SA
    OC SA AS EU OC SA AS AF EU AS AF AS OC AF AF EU AS AF EU EU EU AF EU AF AF
    SA AF SA AS AF SA AF AF AF AS AS OC AS AF OC AS AS SA OC AS AF EU AF OC NA
    SA AS EU SA SA SA SA AS OC OC OC AS AF EU AF AF EU AF -- -- -- EU EU EU EU
    SA SA
'''.split()


#
# Helper functions.
#

def addr_to_num(ip):
    '''
    Convert an IPv4 address from a string to its integer representation.

    @param[in]  ip      IPv4 address as a string.
    @returns            Address as an integer.
    '''

    try:
        w, x, y, z = map(int, ip.split('.'))
        if w>255 or x>255 or y>255 or z>255:
            raise ValueError()
    except ValueError, TypeError:
        raise ValueError('%r is not an IPv4 address.' % (ip,))

    return (w << 24) | (x << 16) | (y << 8) | z


def num_to_addr(num):
    '''
    Convert an IPv4 address from its integer representation to a string.

    @param[in]  num     Address as an integer.
    @returns            IPv4 address as a string.
    '''

    return '%d.%d.%d.%d' % ((num >> 24) & 0xff,
                            (num >> 16) & 0xff,
                            (num >> 8) & 0xff,
                            (num & 0xff))

def latin1_to_utf8(string):
    return string.decode('latin-1').encode('utf-8')


def safe_lookup(lst, idx):
    if idx is None:
        return None
    return lst[idx]


#
# Classes.
#


class ReadBuffer(object):
    '''
    Utility to read data more easily.
    '''

    buffer = None

    def __init__(self, source, size, seek_offset=None, seek_whence=os.SEEK_SET):
        fp = StringIO(source)
        if seek_offset is not None:
            fp.seek(seek_offset, seek_whence)
        self.buffer = fp.read(size)

    def read_string(self):
        '''
        Read a null-terminated string.

        @returns            Result as a string.
        '''
        result, self.buffer = self.buffer.split('\0', 1)
        return result

    def read_int(self, size):
        '''
        Read a multibyte integer.

        @param[in]  size    Number of bytes to read as an integer.
        @returns            Result as an integer.
        '''
        result = sum(ord(self.buffer[i]) << (8*i) for i in range(size))
        self.buffer = self.buffer[size:]
        return result


class AddressInfo(object):
    '''
    Representation of a database lookup result.
    '''

    __slots__ = [ 'ip', 'ipnum', 'prefix', 'country', 'continent' ]

    def __init__(self, ip=None, ipnum=None, prefix=None, country_id=None):
        self.ip = ip
        self.ipnum = ipnum
        self.prefix = prefix
        self.country = safe_lookup(GeoIP_country_code, country_id)
        self.continent = safe_lookup(GeoIP_country_continent, country_id)

    network = property(lambda self:
        num_to_addr(self.ipnum & ~((32-self.prefix)**2-1)))

    def __str__(self):
        return '[%s of network %s/%d in country %s]' %\
               (self.ip, self.network, self.prefix, self.country)


class BigAddressInfo(AddressInfo):
    '''
    Representation of a database lookup result with more info in it.
    '''

    # __slots__ is inherited and appended to.
    __slots__ = [ 'city', 'region', 'postal_code', 'metro_code', 'area_code', 'longitude', 'latitude' ]

    def __init__(self, ip=None, ipnum=None, prefix=None, country_id=None,
                 city=None, region=None, postal_code=None, metro_code=None, area_code=None,
                 longitude=None, latitude=None):
        AddressInfo.__init__(self, ip, ipnum, prefix, country_id)
        self.city = city or None
        self.region = region or None
        self.postal_code = postal_code or None
        self.metro_code = metro_code
        self.area_code = area_code
        self.longitude = longitude
        self.latitude = latitude

    def __str__(self):
        return '[%s of network %s/%d in city %s, %s]' %\
               (self.ip, self.network, self.prefix, self.city, self.country)


class Database(object):
    '''
    GeoIP database reader implementation. Currently only supports country
    edition.
    '''

    def __init__(self, filename):
        '''
        Initialize a new GeoIP reader instance.

        @param[in]  filename    Path to GeoIP.dat as a string.
        '''

        self.filename = filename
        self.cache = open(filename, 'rb').read()
        self._setup_segments()

        if self.db_type not in (GEOIP_COUNTRY_EDITION,
                                GEOIP_CITY_EDITION_REV0,
                                GEOIP_CITY_EDITION_REV1):
            raise NotImplementedError('Database edition is not supported yet; '
                                      'Please use a Country or City database.')

    def _setup_segments(self):
        self.segments = None

        # default to GeoIP Country Edition
        self.db_type = GEOIP_COUNTRY_EDITION
        self.record_length = STANDARD_RECORD_LENGTH

        fp = StringIO(self.cache)
        fp.seek(-3, os.SEEK_END)

        for i in range(STRUCTURE_INFO_MAX_SIZE):
            delim = fp.read(3)

            if delim != '\xFF\xFF\xFF':
                fp.seek(-4, os.SEEK_CUR)
                continue

            self.db_type = ord(fp.read(1))

            # Region Edition, pre June 2003.
            if self.db_type == GEOIP_REGION_EDITION_REV0:
                self.segments = [STATE_BEGIN_REV0]

            # Region Edition, post June 2003.
            elif self.db_type == GEOIP_REGION_EDITION_REV1:
                self.segments = [STATE_BEGIN_REV1]

            # City/Org Editions have two segments, read offset of second segment
            elif self.db_type in (GEOIP_CITY_EDITION_REV0,
                                  GEOIP_CITY_EDITION_REV1,
                                  GEOIP_ORG_EDITION, GEOIP_ISP_EDITION,
                                  GEOIP_ASNUM_EDITION):
                self.segments = [0]

                for idx, ch in enumerate(fp.read(SEGMENT_RECORD_LENGTH)):
                    self.segments[0] += ord(ch) << (idx * 8)

                if self.db_type in (GEOIP_ORG_EDITION, GEOIP_ISP_EDITION):
                    self.record_length = ORG_RECORD_LENGTH

            break

        if self.db_type in (GEOIP_COUNTRY_EDITION, GEOIP_PROXY_EDITION,
                       GEOIP_NETSPEED_EDITION, GEOIP_COUNTRY_EDITION_V6):
            self.segments = [COUNTRY_BEGIN]

    def info(self):
        '''
        Return a string describing the loaded database version.

        @returns    English text string, or None if database is ancient.
        '''

        fp = StringIO(self.cache)
        fp.seek(-3, os.SEEK_END)

        hasStructureInfo = False

        # first get past the database structure information
        for i in range(STRUCTURE_INFO_MAX_SIZE):
            if fp.read(3) == '\xFF\xFF\xFF':
                hasStructureInfo = True
                break

            fp.seek(-4, os.SEEK_CUR)

        if hasStructureInfo:
            fp.seek(-6, os.SEEK_CUR)
        else:
            # no structure info, must be pre Sep 2002 database, go back to end.
            fp.seek(-3, os.SEEK_END)

        for i in range(DATABASE_INFO_MAX_SIZE):
            if fp.read(3) == '\0\0\0':
                return fp.read(i)

            fp.seek(-4, os.SEEK_CUR)

    def _decode(self, buf, branch):
        '''
        @param[in]  buf         Record buffer.
        @param[in]  branch      1 for left, 2 for right.
        @returns                X.
        '''

        offset = 3 * branch
        if self.record_length == 3:
            return buf[offset] | (buf[offset+1] << 8) | (buf[offset+2] << 16)

        # General case.
        end = branch * self.record_length
        x = 0

        for j in range(self.record_length):
            x = (x << 8) | buf[end - j]

        return x

    def _seek_record(self, ipnum):
        fp = StringIO(self.cache)
        offset = 0

        for depth in range(31, -1, -1):
            fp.seek(self.record_length * 2 * offset)
            buf = map(ord, fp.read(self.record_length * 2))

            x = self._decode(buf, int(bool(ipnum & (1 << depth))))
            if x >= self.segments[0]:
                return 32 - depth, x

            offset = x

        assert False, \
            "Error Traversing Database for ipnum = %lu: "\
            "Perhaps database is corrupt?" % ipnum


    def _lookup_country(self, ip):
        "Lookup a country db entry."

        ipnum = addr_to_num(ip)
        prefix, num = self._seek_record(ipnum)

        num -= COUNTRY_BEGIN
        if num:
            country_id = num - 1
        else:
            country_id = None

        return AddressInfo(country_id=country_id, ip=ip, ipnum=ipnum, prefix=prefix)

    def _lookup_city(self, ip):
        "Look up a city db entry."

        ipnum = addr_to_num(ip)
        prefix, num = self._seek_record(ipnum)
        record, next_record_ptr = self._extract_record(num, None)
        return BigAddressInfo(ip=ip, ipnum=ipnum, prefix=prefix, **record)

    def _extract_record(self, seek_record, next_record_ptr):
        if seek_record == self.segments[0]:
            return {'country_id': None}, next_record_ptr

        seek_offset = seek_record + (2 * self.record_length - 1) * self.segments[0]
        record_buf = ReadBuffer(self.cache, FULL_RECORD_LENGTH, seek_offset)
        record = {}

        # get country
        record['country_id'] = record_buf.read_int(1) - 1

        # get region
        record['region'] = record_buf.read_string()

        # get city
        record['city'] = latin1_to_utf8(record_buf.read_string())

        # get postal code
        record['postal_code'] = record_buf.read_string()

        # get latitude
        record['latitude'] = record_buf.read_int(3) / 10000.0 - 180

        # get longitude
        record['longitude'] = record_buf.read_int(3) / 10000.0 - 180

        # get area code and metro code for post April 2002 databases and for US locations
        if (self.db_type == GEOIP_CITY_EDITION_REV1) and (GeoIP_country_code[record['country_id']] == 'US'):
            metro_area_combo = record_buf.read_int(3)
            record['metro_code'] = metro_area_combo / 1000
            record['area_code'] = metro_area_combo % 1000

        # Used for GeoIP_next_record (which this code doesn't have.)
        if next_record_ptr is not None:
            next_record_ptr = seek_record - len(record_buf)

        return record, next_record_ptr

    def lookup(self, ip):
        '''
        Lookup an IP address returning an AddressInfo (or BigAddressInfo)
        instance describing its location.

        @param[in]  ip      IPv4 address as a string.
        @returns            AddressInfo (or BigAddressInfo) instance.
        '''

        if self.db_type in (GEOIP_COUNTRY_EDITION, GEOIP_PROXY_EDITION, GEOIP_NETSPEED_EDITION):
            return self._lookup_country(ip)
        elif self.db_type in (GEOIP_CITY_EDITION_REV0, GEOIP_CITY_EDITION_REV1):
            return self._lookup_city(ip)




if __name__ == '__main__':
    import time, sys

    dbfile = 'GeoIP.dat'
    if len(sys.argv) > 1:
        dbfile = sys.argv[1]

    t1 = time.time()
    db = Database(dbfile)
    t2 = time.time()

    print db.info()

    t3 = time.time()

    tests = '''
        127.0.0.1
        83.198.135.28
        83.126.35.59
        192.168.1.1
        194.168.1.255
        196.25.210.14
        64.22.109.113
    '''.split()

    for test in tests:
        addr_info = db.lookup(test)
        print addr_info
        if isinstance(addr_info, BigAddressInfo):
            print "   ", dict((key, getattr(addr_info, key)) for key in dir(addr_info) if not key.startswith('_'))

    t4 = time.time()

    print "Open: %dms" % ((t2-t1) * 1000,)
    print "Info: %dms" % ((t3-t2) * 1000,)
    print "Lookup: %dms" % ((t4-t3) * 1000,)
