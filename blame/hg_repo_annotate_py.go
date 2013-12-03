package blame

var hgRepoAnnotatePy = `
import hglib, sys, re, json, subprocess, os
from email.utils import parseaddr
from datetime import datetime, tzinfo, timedelta
import time as _time

class LocalTimezone(tzinfo):

    def utcoffset(self, dt):
        if self._isdst(dt):
            return DSTOFFSET
        else:
            return STDOFFSET

    def dst(self, dt):
        if self._isdst(dt):
            return DSTDIFF
        else:
            return ZERO

    def tzname(self, dt):
        return _time.tzname[self._isdst(dt)]

    def _isdst(self, dt):
        tt = (dt.year, dt.month, dt.day,
              dt.hour, dt.minute, dt.second,
              dt.weekday(), 0, 0)
        stamp = _time.mktime(tt)
        tt = _time.localtime(stamp)
        return tt.tm_isdst > 0
Local = LocalTimezone()
STDOFFSET = timedelta(seconds = -_time.timezone)
if _time.daylight:
    DSTOFFSET = timedelta(seconds = -_time.altzone)
else:
    DSTOFFSET = STDOFFSET

DSTDIFF = DSTOFFSET - STDOFFSET

repodir = os.path.abspath(sys.argv[1])
v = sys.argv[2]

sys.stderr.write("Opening hg repository at %s, revision %s\n" % (repodir, v))
client = hglib.open(repodir)

commits = {}
hunksByFile = {}

for rev in client.log('%s:0' % v, followfirst=True):
    authorName, authorEmail = parseaddr(rev.author)
    dt = rev.date.replace(tzinfo=Local)
    commitID = rev.node[:12]
    commit = {
        'ID': commitID,
        'Message': rev.desc,
        'Author': {'Name': authorName, 'Email': authorEmail},
        'AuthorDate': dt.isoformat('T'),
    }
    commits[commitID] = commit

sys.stderr.write("Read %d commits in hg repository at %s, revision %s\n" % (len(commits), repodir, v))

filesNulSep = subprocess.check_output(["hg", "locate", "--print0", "-r", v], cwd=repodir)
files = [f for f in filesNulSep.split("\x00") if f]
sys.stderr.write("Found %d files in hg repository at %s, revision %s\n" % (len(files), repodir, v))

totalHunks = 0
hunkFile = None
hunk = None
charOffsetInFile = 0
def addHunk(file, hunk):
    if file not in hunksByFile:
        hunksByFile[file] = []
    hunksByFile[file].append(hunk)
totalHunks += 1
i = 0
for file in files:
    filepath = os.path.join(repodir, file)
    sys.stderr.write("[% 2d/%d %.1f%%] Annotating file %s in hg repository at %s\n" % (i, len(files), float(i)/float(len(files))*100, file, repodir))
    i += 1
    lineno = 0
    for (info, contents) in client.annotate(files=[filepath], rev=v, changeset=True):
        changeset = info.strip()
    
        startNewHunk = False
        advanceCurrentHunk = False
        if file != hunkFile:
            if hunk is not None:
                addHunk(hunkFile, hunk)
            startNewHunk = True
            charOffsetInFile = 0
        elif changeset != hunk['CommitID']:
            if hunk is not None:
                addHunk(hunkFile, hunk)
            startNewHunk = True
        else:
            advanceCurrentHunk = True
            
        if startNewHunk:
            hunk = {
                'CommitID': changeset[:12],
                'LineStart': lineno,
                'LineEnd': lineno,
                'CharStart': charOffsetInFile,
                'CharEnd': charOffsetInFile+len(contents)+1,
            }
            if hunk['CharStart'] != 0:
                hunk['CharStart'] += 1
            hunkFile = file
        
        charOffsetInFile += len(contents) + 1 # +1 for newline
        if advanceCurrentHunk:
            hunk['LineEnd'] += 1
            hunk['CharEnd'] = charOffsetInFile
        lineno += 1
    if hunk is not None:
        hunk['CharEnd'] = charOffsetInFile + 1
        addHunk(hunkFile, hunk)

sys.stderr.write("Read %d hunks from %d files in hg repository at %s, revision %s\n" % (totalHunks, len(hunksByFile), repodir, v))

json.dump({'Commits': commits, 'Hunks': hunksByFile}, sys.stdout)
`
