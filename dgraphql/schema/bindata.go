// Code generated by go-bindata. DO NOT EDIT.
// sources:
// query.graphql (219B)
// query_alpha.graphql (14B)
// registered_token.graphql (208B)
// schema.graphql (59B)
// serum.graphql (2.792kB)
// subscription.graphql (94B)
// subscription_alpha.graphql (21B)
// token.graphql (129B)

package schema

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func bindataRead(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	clErr := gz.Close()

	if err != nil {
		return nil, fmt.Errorf("read %q: %v", name, err)
	}
	if clErr != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type asset struct {
	bytes  []byte
	info   os.FileInfo
	digest [sha256.Size]byte
}

type bindataFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}

func (fi bindataFileInfo) Name() string {
	return fi.name
}
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}
func (fi bindataFileInfo) IsDir() bool {
	return false
}
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var _queryGraphql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x2a\xa9\x2c\x48\x55\x08\x2c\x4d\x2d\xaa\x54\xa8\xe6\x52\x50\x28\xc9\xcf\x4e\xcd\x2b\xb6\x52\x88\x0e\x01\x31\x82\x52\x8b\x0b\xf2\xf3\x8a\x53\x15\x63\x15\x61\x72\x1a\x89\x29\x29\x45\xa9\xc5\xc5\x56\x0a\xc1\x25\x45\x99\x79\xe9\x8a\x9a\x56\x0a\x28\x6a\xb9\x14\x14\x8a\x52\xd3\x33\x8b\x4b\x52\x8b\x52\x53\x42\x60\xc6\x05\xa1\x0a\xa1\x1a\x8c\xa6\x1e\x9b\x15\x38\xf4\x73\xd5\x72\x71\x15\x27\x27\xe6\x24\x16\x29\x84\x66\xe6\x95\x98\x99\xc0\x78\x5e\xc1\xfe\x7e\x80\x00\x00\x00\xff\xff\x45\xf5\x4f\x3a\xdb\x00\x00\x00")

func queryGraphqlBytes() ([]byte, error) {
	return bindataRead(
		_queryGraphql,
		"query.graphql",
	)
}

func queryGraphql() (*asset, error) {
	bytes, err := queryGraphqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "query.graphql", size: 219, mode: os.FileMode(0644), modTime: time.Unix(1608156034, 0)}
	a := &asset{bytes: bytes, info: info, digest: [32]uint8{0xb3, 0xc, 0x8, 0x60, 0x16, 0xef, 0x49, 0x1b, 0xe7, 0xf6, 0x47, 0x12, 0x35, 0x54, 0x67, 0x76, 0xa6, 0x65, 0x87, 0xa8, 0xe8, 0x21, 0xee, 0x43, 0x43, 0xb5, 0xb9, 0x4c, 0xab, 0xe8, 0xd3, 0xb9}}
	return a, nil
}

var _query_alphaGraphql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x2a\xa9\x2c\x48\x55\x08\x2c\x4d\x2d\xaa\x54\xa8\xae\xe5\x02\x04\x00\x00\xff\xff\x76\xca\x60\x3d\x0e\x00\x00\x00")

func query_alphaGraphqlBytes() ([]byte, error) {
	return bindataRead(
		_query_alphaGraphql,
		"query_alpha.graphql",
	)
}

func query_alphaGraphql() (*asset, error) {
	bytes, err := query_alphaGraphqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "query_alpha.graphql", size: 14, mode: os.FileMode(0644), modTime: time.Unix(1608153826, 0)}
	a := &asset{bytes: bytes, info: info, digest: [32]uint8{0xf6, 0x58, 0xf, 0x86, 0xb0, 0x43, 0x44, 0x23, 0x5f, 0x97, 0xd3, 0xde, 0x25, 0xbd, 0x4b, 0x29, 0x22, 0xad, 0x9b, 0x95, 0xef, 0x8, 0x81, 0x45, 0x11, 0x3a, 0x12, 0x62, 0xab, 0x4c, 0x93, 0xf8}}
	return a, nil
}

var _registered_tokenGraphql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x6c\x8b\x31\x0e\xc2\x30\x0c\x45\xf7\x9e\xc2\xdc\x01\x31\x74\x63\x64\x2d\x70\x80\x96\x7c\x82\x45\x62\x47\xb1\x2b\x14\x10\x77\x47\x30\x65\x60\x7c\xff\xbd\xef\xad\x80\x26\x44\x36\x47\x45\x38\xe9\x1d\x32\xc1\x8a\x8a\x81\x5e\x03\xd1\x1c\x42\x85\xd9\x48\x47\xaf\x2c\x71\x33\x10\x65\x16\xdf\xaf\x7e\xd3\xca\xde\x7a\x71\xad\xc0\x13\x7f\x95\xad\xa5\xa4\x36\xd2\x99\xc5\x77\xdb\xef\x12\x70\xe1\x3c\x27\x1b\xe9\x20\xfe\x4b\x5a\x5e\x34\xf5\x27\x99\x33\x7a\x4e\x1a\xb5\xe7\x07\x16\x63\xef\x92\xf7\xf0\x09\x00\x00\xff\xff\x65\x08\x1e\x7a\xd0\x00\x00\x00")

func registered_tokenGraphqlBytes() ([]byte, error) {
	return bindataRead(
		_registered_tokenGraphql,
		"registered_token.graphql",
	)
}

func registered_tokenGraphql() (*asset, error) {
	bytes, err := registered_tokenGraphqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "registered_token.graphql", size: 208, mode: os.FileMode(0644), modTime: time.Unix(1608153758, 0)}
	a := &asset{bytes: bytes, info: info, digest: [32]uint8{0xa7, 0xad, 0xf8, 0x27, 0xfd, 0xab, 0x5a, 0x68, 0x60, 0x15, 0x89, 0x72, 0xbe, 0xcc, 0xcf, 0x4f, 0xcf, 0x8c, 0x4c, 0x3b, 0xe6, 0xcb, 0x89, 0x83, 0xe7, 0x3c, 0x5d, 0x1a, 0xf6, 0xe1, 0xec, 0xb5}}
	return a, nil
}

var _schemaGraphql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x2a\x4e\xce\x48\xcd\x4d\x54\xa8\xe6\x52\x50\x50\x50\x28\x2c\x4d\x2d\xaa\xb4\x52\x08\x04\x51\x60\x81\xe2\xd2\xa4\xe2\xe4\xa2\xcc\x82\x92\xcc\xfc\x3c\x2b\x85\x60\x24\x1e\x57\x2d\x17\x20\x00\x00\xff\xff\x52\xd9\x58\xe5\x3b\x00\x00\x00")

func schemaGraphqlBytes() ([]byte, error) {
	return bindataRead(
		_schemaGraphql,
		"schema.graphql",
	)
}

func schemaGraphql() (*asset, error) {
	bytes, err := schemaGraphqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "schema.graphql", size: 59, mode: os.FileMode(0644), modTime: time.Unix(1608156028, 0)}
	a := &asset{bytes: bytes, info: info, digest: [32]uint8{0x5b, 0xba, 0x8f, 0x74, 0x29, 0x4e, 0xaf, 0x41, 0x66, 0x2b, 0x4e, 0x31, 0x85, 0x84, 0x19, 0x59, 0x50, 0x81, 0xda, 0x72, 0x50, 0x56, 0xaf, 0xe3, 0xd8, 0xb7, 0x35, 0xbc, 0xd1, 0x85, 0xf3, 0xc6}}
	return a, nil
}

var _serumGraphql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xac\x56\xdd\x4e\xe3\x3c\x10\xbd\xcf\x53\x98\x77\xf8\xf4\x5d\x44\xda\x8b\xfe\x21\x45\xf4\x8f\xa6\x2c\x5a\x21\x84\xdc\x64\x68\x2d\x1c\x3b\xd8\x63\xa0\xb0\x7d\xf7\x95\x93\x26\xb1\x93\xb4\xcb\xee\x92\x8b\x4a\x39\x33\x9e\xce\xcc\x39\x33\x4e\x80\xfb\x1c\x48\x0c\xca\x64\x91\xd0\xa8\x4c\x82\x4c\x8a\x15\xe8\x5c\x0a\x0d\xe4\x23\x20\x04\xd5\x5b\xcc\xb6\x82\xa2\x51\x10\x92\x18\x15\x13\xdb\x8b\x12\x9f\x28\x25\x55\x48\x86\x52\x72\xa0\xc2\x82\xac\x09\x12\x76\xc2\x06\x87\x20\x30\x82\x49\xd1\xb1\x90\x6f\xe4\x46\xa4\x90\xc8\x14\x52\x17\xfe\x59\x79\x32\x64\x94\xb3\x77\x98\x51\xf5\x04\x58\xe1\x73\x78\x5d\xa8\x14\x54\xf5\x3e\xa3\x98\xec\x3c\x64\x24\x85\x36\x19\x4c\x5e\x40\xa0\xae\x41\x2a\x12\xe0\x9e\x5f\x0c\x88\x1c\x2e\x8d\x48\xfb\xbc\x86\xfb\x11\x67\x20\x30\x4a\x83\xb2\x61\xbd\xc9\xda\x66\xe5\x4a\x6e\x15\xcd\xa2\x71\x24\x52\x78\x0b\xc9\x0d\x13\xf8\xff\x7f\xb6\x33\x34\x49\xa4\x11\x38\xb2\x3f\x21\x21\x5d\x8b\xb6\xa8\x7d\xee\x8e\xa6\x7b\x6b\x4c\x29\xd2\x29\x88\x2d\xee\x4a\x73\x73\xcc\x5a\xaa\x23\xf6\x69\x98\x81\x92\x96\x8e\xe5\x10\x78\x74\xfb\x3d\x1d\x1c\xb3\x28\xca\xc8\x0a\xc8\x65\x5b\xe7\x7c\x24\x99\x58\xcb\x27\x10\x2d\x7c\xa9\x58\x02\x1d\x43\x22\x99\x98\x31\xe1\x05\xc9\xad\xa7\x0f\x9e\xcd\xa9\xc8\x65\x43\x35\x4c\x25\xc6\xec\x1d\xdc\x7e\x3e\x1b\x89\x7d\xf8\x23\xc0\x8a\x22\x0c\x73\xed\xa2\x2f\xd4\x70\xb4\x32\x06\x35\x97\x22\xe9\x46\x1a\x1b\x8d\xeb\x9d\x02\xbd\x93\x3c\x6d\xac\x0e\x3d\x24\x3c\xdf\xb8\xa6\x96\x23\x52\x0a\xc2\x6c\x38\x4b\xae\x60\x1f\xba\x0c\x31\x5d\xe6\x62\x49\x72\xa7\x47\xdf\x2a\x86\x74\xc3\xc1\x19\xaa\x83\xd3\xa0\x4a\xf0\xbd\x64\x1d\x41\x1b\x48\xe6\x20\x0a\x47\xed\xc1\x0a\x9e\x0d\x68\xbc\x36\x60\xc0\x33\xe4\x74\x6f\x93\x71\x23\xbc\x8a\x16\x62\x09\xfd\x6e\xdb\xe8\x9f\x4c\xba\x98\xce\x79\x21\x87\x65\x39\x0c\xad\x14\x44\xcb\x59\x65\x63\xa6\x93\x72\x2c\x8e\xb8\xaf\x8a\x7a\xca\x6d\xb1\x9a\xa5\x50\x4b\x3b\x66\x29\xac\xf7\x39\xd8\x38\x9c\x65\x0c\x0b\x29\x86\xee\x94\x64\xf4\xed\xda\x50\x81\x0c\xf7\x2e\xe7\xd2\x06\xb4\x47\x6d\xac\x45\xf5\x52\x94\x59\x4e\xfa\x38\x74\xa7\x2d\xf0\xc6\xb4\x97\x8a\x96\x94\x9b\x55\xf4\x3b\xae\x4e\x92\x02\x76\x69\x75\xe1\x0d\x4b\x7d\x56\xa9\x7e\xd2\x1d\xa6\x2e\x01\x56\x90\x00\x7b\x29\xd5\xe4\x31\x76\xca\x76\xe8\xcd\xbf\xc8\xbb\xe8\xee\xd9\xb1\xe8\x16\xdc\xea\x88\xb7\x8a\xbd\xa6\xb8\x6a\xbd\xab\xb2\xb9\xbf\xe8\x6f\xd7\x89\xae\x7c\x45\xcd\xfe\x65\x71\xbe\xec\xf0\x4c\x51\xed\xc2\x9b\x8b\xe4\xaf\xb5\xd0\x1e\xc7\x43\x7f\x7c\x67\x44\xdc\xd9\x28\xd4\x1e\xa5\xee\x2a\x76\x7b\xde\x83\xc6\x5c\x62\xcf\xed\x55\x97\xdd\x2d\xa9\x55\xb4\x73\xa1\x56\x1e\x7f\xb6\xab\xfe\x65\x01\x59\xcf\x5b\xca\x39\xb4\x5d\x7b\x40\x7d\x5c\xc4\x9f\xde\x5f\x8f\xa0\x14\xa8\xe5\xa8\x15\xcb\xa3\xc4\xfd\x9c\xf8\xe8\x19\x96\x9e\xee\x9c\x16\x4d\xf3\xf5\xf1\xd5\x6b\xff\x93\xaa\x6a\x12\x28\xfe\xb8\xd9\x90\x9f\xd1\x47\x37\xfb\xa2\x52\x10\x26\xab\x35\x5a\x84\x1d\xc4\x57\x01\x21\xc3\x68\x1c\x10\x72\x33\xbf\x9a\x2f\x6e\xe7\xb5\x63\xbd\xa1\x0b\xcf\x69\x34\x8b\xd6\x01\x21\xd1\x6c\x36\x19\x47\x83\xf5\xe4\x61\xb1\x7a\x18\x0d\xe6\xa3\xc9\x34\x20\x64\xb9\x88\xd7\x0f\x8b\xf9\xf4\x87\x1b\xe7\x57\x00\x00\x00\xff\xff\x23\xab\x2c\xec\xe8\x0a\x00\x00")

func serumGraphqlBytes() ([]byte, error) {
	return bindataRead(
		_serumGraphql,
		"serum.graphql",
	)
}

func serumGraphql() (*asset, error) {
	bytes, err := serumGraphqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "serum.graphql", size: 2792, mode: os.FileMode(0644), modTime: time.Unix(1608153878, 0)}
	a := &asset{bytes: bytes, info: info, digest: [32]uint8{0xef, 0xbd, 0x4e, 0xa1, 0x3e, 0x6b, 0x12, 0xcf, 0xa8, 0x0, 0x5c, 0xe4, 0x99, 0x5f, 0x61, 0x15, 0xcb, 0xa1, 0xa3, 0x60, 0x67, 0x47, 0xfc, 0x26, 0x41, 0x8e, 0xa, 0xac, 0xe0, 0xaa, 0x5e, 0x3f}}
	return a, nil
}

var _subscriptionGraphql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x2a\xa9\x2c\x48\x55\x08\x2e\x4d\x2a\x4e\x2e\xca\x2c\x28\xc9\xcc\xcf\x53\xa8\xe6\x52\x50\x50\x50\x28\x4e\x2d\x2a\xcd\xf5\xcc\x2b\x2e\x29\x2a\x4d\x06\x09\x7b\x64\x16\x97\xe4\x17\x55\x6a\x24\x26\x27\xe7\x97\xe6\x95\x58\x29\x04\x97\x14\x65\xe6\xa5\x2b\x6a\x5a\x29\x04\xa3\x29\x0d\x4a\x2d\x2e\xc8\xcf\x2b\x4e\xe5\xaa\xe5\x02\x04\x00\x00\xff\xff\xba\xf2\xbd\x88\x5e\x00\x00\x00")

func subscriptionGraphqlBytes() ([]byte, error) {
	return bindataRead(
		_subscriptionGraphql,
		"subscription.graphql",
	)
}

func subscriptionGraphql() (*asset, error) {
	bytes, err := subscriptionGraphqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "subscription.graphql", size: 94, mode: os.FileMode(0644), modTime: time.Unix(1608153851, 0)}
	a := &asset{bytes: bytes, info: info, digest: [32]uint8{0x7d, 0x54, 0x96, 0xf5, 0x98, 0x89, 0xb2, 0xd0, 0xf9, 0x8a, 0x9b, 0x7f, 0xcc, 0xb5, 0x9e, 0xee, 0xf6, 0x5f, 0xf7, 0x11, 0xe4, 0xcc, 0x82, 0x8b, 0x4a, 0x4f, 0x67, 0x9a, 0x59, 0xee, 0x8f, 0x2b}}
	return a, nil
}

var _subscription_alphaGraphql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x2a\xa9\x2c\x48\x55\x08\x2e\x4d\x2a\x4e\x2e\xca\x2c\x28\xc9\xcc\xcf\x53\xa8\xae\xe5\x02\x04\x00\x00\xff\xff\x4d\xe9\x40\xe8\x15\x00\x00\x00")

func subscription_alphaGraphqlBytes() ([]byte, error) {
	return bindataRead(
		_subscription_alphaGraphql,
		"subscription_alpha.graphql",
	)
}

func subscription_alphaGraphql() (*asset, error) {
	bytes, err := subscription_alphaGraphqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "subscription_alpha.graphql", size: 21, mode: os.FileMode(0644), modTime: time.Unix(1608153609, 0)}
	a := &asset{bytes: bytes, info: info, digest: [32]uint8{0x3d, 0x20, 0x13, 0xe7, 0x39, 0xf6, 0x4e, 0x44, 0x50, 0x61, 0x33, 0x62, 0x95, 0x35, 0x1d, 0x5c, 0x43, 0x43, 0xb0, 0xe4, 0xe2, 0xf9, 0xa2, 0x94, 0xdd, 0xcc, 0xad, 0x3b, 0x5d, 0x4d, 0xce, 0x16}}
	return a, nil
}

var _tokenGraphql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x6c\xc9\x3b\x0e\x02\x31\x0c\x04\xd0\x3e\xa7\x98\xbd\x03\xa2\x48\x47\x49\xcb\xe7\x00\x2b\x32\x80\xc5\xae\x13\xd9\x4e\x11\x10\x77\x47\x74\x14\xb4\xef\xc5\x68\xc4\xa9\x3e\xa8\x07\x7a\xab\xea\xc4\x2b\x01\x73\x29\x46\xf7\x8c\x63\x98\xe8\x6d\x4a\xc0\x2a\x1a\xbb\x1e\xf7\x6a\x12\xe3\x37\xae\x46\x3e\xf9\xb7\xbc\xb7\xb6\x8c\x8c\xb3\x68\x6c\x37\x5f\x29\xbc\xc8\x3a\x2f\x9e\xb1\xd7\x98\xd2\x3b\x7d\x02\x00\x00\xff\xff\xea\xfd\xcb\xa8\x81\x00\x00\x00")

func tokenGraphqlBytes() ([]byte, error) {
	return bindataRead(
		_tokenGraphql,
		"token.graphql",
	)
}

func tokenGraphql() (*asset, error) {
	bytes, err := tokenGraphqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "token.graphql", size: 129, mode: os.FileMode(0644), modTime: time.Unix(1608153734, 0)}
	a := &asset{bytes: bytes, info: info, digest: [32]uint8{0x6e, 0xfc, 0xe0, 0x34, 0x15, 0x16, 0xc2, 0xa8, 0xed, 0x78, 0xdf, 0x22, 0xa0, 0xb3, 0xf7, 0xfa, 0x49, 0xa0, 0x39, 0x79, 0x74, 0xe4, 0xf1, 0xed, 0x1b, 0xa1, 0xd2, 0x5a, 0x5c, 0x36, 0xdd, 0xdf}}
	return a, nil
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	canonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[canonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// AssetString returns the asset contents as a string (instead of a []byte).
func AssetString(name string) (string, error) {
	data, err := Asset(name)
	return string(data), err
}

// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
func MustAsset(name string) []byte {
	a, err := Asset(name)
	if err != nil {
		panic("asset: Asset(" + name + "): " + err.Error())
	}

	return a
}

// MustAssetString is like AssetString but panics when Asset would return an
// error. It simplifies safe initialization of global variables.
func MustAssetString(name string) string {
	return string(MustAsset(name))
}

// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func AssetInfo(name string) (os.FileInfo, error) {
	canonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[canonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, fmt.Errorf("AssetInfo %s not found", name)
}

// AssetDigest returns the digest of the file with the given name. It returns an
// error if the asset could not be found or the digest could not be loaded.
func AssetDigest(name string) ([sha256.Size]byte, error) {
	canonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[canonicalName]; ok {
		a, err := f()
		if err != nil {
			return [sha256.Size]byte{}, fmt.Errorf("AssetDigest %s can't read by error: %v", name, err)
		}
		return a.digest, nil
	}
	return [sha256.Size]byte{}, fmt.Errorf("AssetDigest %s not found", name)
}

// Digests returns a map of all known files and their checksums.
func Digests() (map[string][sha256.Size]byte, error) {
	mp := make(map[string][sha256.Size]byte, len(_bindata))
	for name := range _bindata {
		a, err := _bindata[name]()
		if err != nil {
			return nil, err
		}
		mp[name] = a.digest
	}
	return mp, nil
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() (*asset, error){
	"query.graphql": queryGraphql,

	"query_alpha.graphql": query_alphaGraphql,

	"registered_token.graphql": registered_tokenGraphql,

	"schema.graphql": schemaGraphql,

	"serum.graphql": serumGraphql,

	"subscription.graphql": subscriptionGraphql,

	"subscription_alpha.graphql": subscription_alphaGraphql,

	"token.graphql": tokenGraphql,
}

// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"},
// AssetDir("data/img") would return []string{"a.png", "b.png"},
// AssetDir("foo.txt") and AssetDir("notexist") would return an error, and
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		canonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(canonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for childName := range node.Children {
		rv = append(rv, childName)
	}
	return rv, nil
}

type bintree struct {
	Func     func() (*asset, error)
	Children map[string]*bintree
}

var _bintree = &bintree{nil, map[string]*bintree{
	"query.graphql":              &bintree{queryGraphql, map[string]*bintree{}},
	"query_alpha.graphql":        &bintree{query_alphaGraphql, map[string]*bintree{}},
	"registered_token.graphql":   &bintree{registered_tokenGraphql, map[string]*bintree{}},
	"schema.graphql":             &bintree{schemaGraphql, map[string]*bintree{}},
	"serum.graphql":              &bintree{serumGraphql, map[string]*bintree{}},
	"subscription.graphql":       &bintree{subscriptionGraphql, map[string]*bintree{}},
	"subscription_alpha.graphql": &bintree{subscription_alphaGraphql, map[string]*bintree{}},
	"token.graphql":              &bintree{tokenGraphql, map[string]*bintree{}},
}}

// RestoreAsset restores an asset under the given directory.
func RestoreAsset(dir, name string) error {
	data, err := Asset(name)
	if err != nil {
		return err
	}
	info, err := AssetInfo(name)
	if err != nil {
		return err
	}
	err = os.MkdirAll(_filePath(dir, filepath.Dir(name)), os.FileMode(0755))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(_filePath(dir, name), data, info.Mode())
	if err != nil {
		return err
	}
	return os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
}

// RestoreAssets restores an asset under the given directory recursively.
func RestoreAssets(dir, name string) error {
	children, err := AssetDir(name)
	// File
	if err != nil {
		return RestoreAsset(dir, name)
	}
	// Dir
	for _, child := range children {
		err = RestoreAssets(dir, filepath.Join(name, child))
		if err != nil {
			return err
		}
	}
	return nil
}

func _filePath(dir, name string) string {
	canonicalName := strings.Replace(name, "\\", "/", -1)
	return filepath.Join(append([]string{dir}, strings.Split(canonicalName, "/")...)...)
}