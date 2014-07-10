go-blame
========

A simple wrapper for calling git-blame from Go. Wraps command-line
git, and thus requires git to be installed. Supports blame queries to
show the most-recent-authorship percentages of a portion of code.

Requirements
------------

* Python `hglib` package (for Mercurial blaming)


Known issues
------------

Tests fail because 90f26648c7d4b2dd4d0067591ae247f374e24c64 introduced a temporary workaround to ignore failures on `git blame`. Check the commit log for more info. TODO: fix the actual underlying feature and the tests should pass again. (Don't remove the tests because we don't want to ignore empty files in blame output.)
