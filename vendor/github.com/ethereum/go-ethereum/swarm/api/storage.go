// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package api

import "path"

type Response struct {
	MimeType string
	Status   int
	Size     int64
	// Content  []byte
	Content string
}

// implements a service
//
// DEPRECATED: Use the HTTP API instead
type Storage struct {
	api *Api
}

func NewStorage(api *Api) *Storage {
	return &Storage{api}
}

// Put uploads the content to the swarm with a simple manifest speficying
// its content type
//
// DEPRECATED: Use the HTTP API instead
func (self *Storage) Put(content, contentType string) (string, error) {
	key, err := self.api.Put(content, contentType)
	if err != nil {
		return "", err
	}
	return key.String(), err
}

// Get retrieves the content from bzzpath and reads the response in full
// It returns the Response object, which serialises containing the
// response body as the value of the Content field
// NOTE: if error is non-nil, sResponse may still have partial content
// the actual size of which is given in len(resp.Content), while the expected
// size is resp.Size
//
// DEPRECATED: Use the HTTP API instead
func (self *Storage) Get(bzzpath string) (*Response, error) {
	uri, err := Parse(path.Join("bzz:/", bzzpath))
	if err != nil {
		return nil, err
	}
	key, err := self.api.Resolve(uri)
	if err != nil {
		return nil, err
	}
	reader, mimeType, status, err := self.api.Get(key, uri.Path)
	if err != nil {
		return nil, err
	}
	quitC := make(chan bool)
	expsize, err := reader.Size(quitC)
	if err != nil {
		return nil, err
	}
	body := make([]byte, expsize)
	size, err := reader.Read(body)
	if int64(size) == expsize {
		err = nil
	}
	return &Response{mimeType, status, expsize, string(body[:size])}, err
}

// Modify(rootHash, basePath, contentHash, contentType) takes th e manifest trie rooted in rootHash,
// and merge on  to it. creating an entry w conentType (mime)
//
// DEPRECATED: Use the HTTP API instead
func (self *Storage) Modify(rootHash, path, contentHash, contentType string) (newRootHash string, err error) {
	uri, err := Parse("bzz:/" + rootHash)
	if err != nil {
		return "", err
	}
	key, err := self.api.Resolve(uri)
	if err != nil {
		return "", err
	}
	key, err = self.api.Modify(key, path, contentHash, contentType)
	if err != nil {
		return "", err
	}
	return key.String(), nil
}
