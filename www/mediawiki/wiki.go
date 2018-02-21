// Copyright (c) 2018 Aram Hăvărneanu <aram@mgk.ro>
//
// Permission to use, copy, modify, and distribute this software for any
// purpose with or without fee is hereby granted, provided that the above
// copyright notice and this permission notice appear in all copies.
//
// THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
// WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
// MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
// ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
// WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
// ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
// OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

// Package mediawiki provides access to MediaWiki installations to Go
// web servers via CGI.
package mediawiki // import "mgk.ro/www/mediawiki"

import (
	"net/http"
	"net/http/cgi"
	"os"
	"path/filepath"
	"strings"
)

const (
	// Default PHP script to execute, if unspecified by the HTTP request.
	DirIndex = "index.php"

	// Default PHP CGI binary
	PHPExe = "php-cgi"

	// only execute PHP code from these directories (relative to
	// the mediawiki installation directory), but not from their
	// children.
	PHPWhitelistDirs = "/"

	// only serve static assets from these directories (relative
	// to the mediawiki installation directory) and from their
	// children, recursively.
	AssetWhitelistDirsRecursive = "/resources:/skins"
)

type httpError int

func (e httpError) Error() string {
	return http.StatusText(int(e))
}

const (
	errBadRequest httpError = http.StatusBadRequest
	errForbidden  httpError = http.StatusForbidden
	errNotFound   httpError = http.StatusNotFound
)

func httpReturnError(w http.ResponseWriter, err error) {
	switch v := err.(type) {
	case httpError:
		http.Error(w, v.Error(), int(v))
	default:
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

// A *MediaWiki is an http.Handler that serves a mediawiki instance.
type MediaWiki struct {
	Root      string // directory containing Mediawiki files
	URLPrefix string // the URL prefix used for accessing the wiki
	PHPExe    string // php-cgi executable

	// see the constants with the same name for a description of
	// these fields.
	PHPWhitelistDirs            []string
	AssetWhitelistDirsRecursive []string
}

// New returns a new HTTP handler that serves the mediawiki
// installed at root using the PHP CGI executable specified by
// php. The handler should be installed at urlprefix.
func New(root, urlprefix, php string) *MediaWiki {
	mw := &MediaWiki{
		Root:             root,
		URLPrefix:        urlprefix,
		PHPExe:           php,
		PHPWhitelistDirs: []string{filepath.Join(root, PHPWhitelistDirs)},
	}
	for _, dir := range strings.Split(AssetWhitelistDirsRecursive, ":") {
		dir := filepath.Join(root, dir)
		mw.AssetWhitelistDirsRecursive = append(mw.AssetWhitelistDirsRecursive, dir)
	}

	// If LocalSettings.php is not present, the wiki is not
	// installed, in which case we need to allow access to the
	// installer.
	// BUG(aram): After install, the server still allows access to the installer until restart.
	fi, err := os.Stat(filepath.Join(root, "LocalSettings.php"))
	if err != nil || !fi.Mode().IsRegular() {
		mw.PHPWhitelistDirs = append(mw.PHPWhitelistDirs, filepath.Join(root, "/mw-config"))
		mw.AssetWhitelistDirsRecursive = append(mw.AssetWhitelistDirsRecursive, filepath.Join(root, "/mw-config"))
	}

	if mw.PHPExe == "" {
		mw.PHPExe = PHPExe
	}
	return mw
}

// getFileName validates the HTTP request, rejecting access to
// non-whitelisted scripts or static assets. It returns the canonicalized
// path to the script or resource, or an error.
func (mw *MediaWiki) getFileName(w http.ResponseWriter, r *http.Request) (string, error) {
	file, err := filepath.Rel(mw.URLPrefix, r.URL.Path)
	if err != nil {
		return "", errBadRequest
	}
	file = filepath.Join(mw.Root, file)
noindex:
	fi, err := os.Stat(file)
	if err != nil {
		return "", errNotFound
	}
	if fi.IsDir() {
		file = filepath.Join(file, DirIndex)
		goto noindex
	}

	ext := filepath.Ext(file)
	if ext == ".php" {
		for _, dir := range mw.PHPWhitelistDirs {
			if filepath.Dir(file) == dir {
				return file, nil
			}
		}
	} else {
		for _, dir := range mw.AssetWhitelistDirsRecursive {
			if strings.HasPrefix(file, dir) {
				return file, nil
			}
		}
	}
	return "", errForbidden
}

func (mw *MediaWiki) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	file, err := mw.getFileName(w, r)
	if err != nil {
		httpReturnError(w, err)
		return
	}
	ext := filepath.Ext(file)
	switch ext {
	case ".php":
		h := cgi.Handler{
			Path: mw.PHPExe,
			Dir:  mw.Root,
			Args: []string{file},
			Env: []string{
				// We need to explicitely set this, because we are ran by
				// an interpretor.
				"SCRIPT_FILENAME=" + file,

				// Unsure what this variable should to be set to. Leaving it
				// unset causes PHP to become confused about its own URL.
				// Making it set, but empty, seems to work.
				"SCRIPT_NAME=",

				// without this variable, php-cgi refuses to run.
				"REDIRECT_STATUS=1",
			},
		}
		h.ServeHTTP(w, r)
	default:
		http.ServeFile(w, r, file)
	}
}
