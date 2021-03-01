// Code generated by go-bindata. DO NOT EDIT.
// sources:
// analytics.graphql (56B)
// query.graphql (1.061kB)
// query_alpha.graphql (14B)
// schema.graphql (28B)
// serum_fill.graphql (1.929kB)
// serum_market.graphql (1.287kB)
// serum_order.graphql (1.247kB)
// serum_order_tracker.graphql (1.949kB)
// token.graphql (772B)

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
		return nil, fmt.Errorf("read %q: %w", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	clErr := gz.Close()

	if err != nil {
		return nil, fmt.Errorf("read %q: %w", name, err)
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

var _analyticsGraphql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x2a\xa9\x2c\x48\x55\x70\x49\xcc\xcc\xa9\x0c\xcb\xcf\x29\xcd\x4d\x55\xa8\xe6\x52\x50\x50\x50\x48\x49\x2c\x49\xb5\x52\x08\xc9\xcc\x4d\x55\x04\xf3\xcb\x12\x73\x4a\x53\xad\x14\xdc\x72\xf2\x13\x4b\xcc\x4c\x14\xb9\x6a\x01\x01\x00\x00\xff\xff\x32\x54\xee\x0b\x38\x00\x00\x00")

func analyticsGraphqlBytes() ([]byte, error) {
	return bindataRead(
		_analyticsGraphql,
		"analytics.graphql",
	)
}

func analyticsGraphql() (*asset, error) {
	bytes, err := analyticsGraphqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "analytics.graphql", size: 56, mode: os.FileMode(0644), modTime: time.Unix(1614612938, 0)}
	a := &asset{bytes: bytes, info: info, digest: [32]uint8{0x13, 0xfe, 0x8b, 0x7a, 0x9e, 0x96, 0xf, 0xc8, 0x34, 0xc6, 0xf8, 0x4a, 0x35, 0x91, 0xfd, 0x44, 0x66, 0xb1, 0xc4, 0xb9, 0x67, 0x67, 0xae, 0x31, 0x5, 0xca, 0x38, 0x48, 0x29, 0xfe, 0x2c, 0xf5}}
	return a, nil
}

var _queryGraphql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x84\x93\x4f\x6f\xdb\x48\x0c\xc5\xef\xfe\x14\x2f\xba\x6c\x16\x30\x82\x2c\xb0\xd8\x83\x8f\x9b\x34\x68\x0b\x34\xfd\xe3\xf4\x03\xd0\x12\x65\x11\x19\x0f\x5d\x92\x72\x6a\x04\xfd\xee\xc5\x8c\xa2\x34\x0d\x52\xe4\xa6\xf1\xbc\x47\x72\x7e\x8f\x8e\xe3\x9e\xf1\x79\x64\x3b\xe2\x7e\xb1\x00\x42\x6f\x39\xfb\x69\xab\x63\x8e\x15\xbe\x4a\x8e\xff\xfe\x5d\xa2\x1d\xcd\xd5\x56\x58\x87\x49\xde\xfe\xbd\xc2\x4d\x91\x5d\x68\xce\xdc\x86\x68\x3e\x79\xb4\x9e\x52\xd7\x19\xbb\xcf\xda\x93\x59\xfc\xa6\xdb\x72\x95\x35\x4d\xf3\x85\x63\xb4\x8c\x44\x1e\xf8\xe7\xfc\x1c\xbd\xa4\xe4\xe8\xd5\x10\x03\x63\x2b\x07\xce\x08\xa3\x8e\x6d\x89\x1d\xd9\x2d\xc7\x12\x6a\xd8\x68\x0c\x67\x98\xcc\x0e\x31\xe3\x03\x9b\xcb\x26\x31\x34\xa7\x23\x3a\x0a\x82\x2b\x92\x1c\x78\x3a\x64\x0d\x1c\x39\xb0\x27\x0b\x68\x0f\x82\x27\x0d\xc4\x40\x01\x71\x48\xae\x0d\x4d\x35\xe0\x1c\xb8\x93\x94\xaa\x67\xc3\xb0\xda\x86\x3b\xdc\x0d\x9c\xd1\x52\x4a\xdc\x9d\x35\x4d\xb3\x00\x9c\x6d\xdc\x5d\x49\x4a\x6f\xc5\x43\xed\x78\x3a\xcd\x3a\x3f\x79\x9e\xf9\x09\xae\xf5\xec\x78\x86\xac\x56\xfa\x50\xd5\xaf\x33\x5f\xff\x12\x3f\x2b\xf3\x88\xd4\x27\xd1\x5f\x8e\x49\x87\x8e\xe4\x01\xcc\x93\xd1\xa7\xbb\xcb\x72\x75\x49\x41\x2f\x45\xb6\x7e\x41\x77\xb2\xf8\xb1\x58\x34\x4d\x73\x51\x07\x73\x18\x7f\x1b\xc5\xb8\x43\x28\x5a\xcd\x21\x79\x64\xb0\xc4\xc0\x56\xb2\xbc\x23\xeb\x6a\x6a\xd4\xde\x96\x6f\x47\x6f\xba\x03\x21\x89\xd7\x30\xf6\xb4\x95\x4c\xc1\x1d\x38\xf1\x8e\x73\x78\x99\xb1\x2e\xe4\x27\xda\xf2\xbb\xdc\x2b\xee\x17\x40\x7d\xdf\xd4\xb4\xd8\x4a\x62\xbd\x98\xc7\x6c\x9b\x7f\x2c\x75\x97\x18\x9d\x21\x51\x66\x72\x26\x6b\x87\x39\x64\xdd\xef\xd5\x25\x18\x9d\xd8\x84\x6e\x22\x02\x78\x90\xc5\xc5\x6f\xb4\x2b\xd5\x17\xfa\xd6\x8d\x7d\xa5\xed\x8c\x62\x2e\xcf\xb9\xfb\x53\xf1\x4b\x0e\xb6\x9d\x64\x76\x48\x2d\x66\x5c\xd6\x92\x0a\x1a\x06\x1d\x48\x12\x95\xed\xa6\x3e\x78\xfa\x6f\x68\x2e\x9b\xb9\x37\x76\xce\x85\xdc\xe6\x88\x18\xc4\x27\x83\xe4\x5e\xe7\xae\x03\xf9\x35\x7f\x8f\x02\x72\x85\xff\x55\x13\x53\xae\x01\x7a\x4b\x89\xec\x61\xcb\xe6\xd3\x8d\xec\x78\xfe\xbe\x4a\x4a\x4f\xae\xde\xaf\x3f\x5e\xff\x0c\x00\x00\xff\xff\xb6\x56\x7f\x76\x25\x04\x00\x00")

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

	info := bindataFileInfo{name: "query.graphql", size: 1061, mode: os.FileMode(0644), modTime: time.Unix(1614181656, 0)}
	a := &asset{bytes: bytes, info: info, digest: [32]uint8{0x55, 0xdd, 0xbf, 0xc9, 0x70, 0x33, 0xec, 0xa7, 0x7d, 0xe2, 0xd1, 0x58, 0x6f, 0xc2, 0xec, 0xcd, 0xef, 0x63, 0xc8, 0x64, 0x67, 0x97, 0xa, 0x6d, 0x39, 0x10, 0x4b, 0x57, 0x6a, 0x9a, 0xc4, 0x6d}}
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

var _schemaGraphql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x2a\x4e\xce\x48\xcd\x4d\x54\xa8\xe6\x52\x50\x50\x50\x28\x2c\x4d\x2d\xaa\xb4\x52\x08\x04\x51\x5c\xb5\x5c\x80\x00\x00\x00\xff\xff\x9e\xeb\xeb\x5e\x1c\x00\x00\x00")

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

	info := bindataFileInfo{name: "schema.graphql", size: 28, mode: os.FileMode(0644), modTime: time.Unix(1611874630, 0)}
	a := &asset{bytes: bytes, info: info, digest: [32]uint8{0xbb, 0x52, 0xf, 0x8c, 0x59, 0xcb, 0xec, 0xb1, 0xfd, 0x14, 0xda, 0xdd, 0x80, 0x73, 0x37, 0xba, 0xee, 0x84, 0xf, 0x5e, 0x4d, 0x73, 0x64, 0xe3, 0xb4, 0x18, 0xb7, 0x86, 0x6e, 0xc3, 0xc5, 0x70}}
	return a, nil
}

var _serum_fillGraphql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x8c\x55\xcd\x6e\xdc\x36\x10\xbe\xeb\x29\x26\xba\xc4\x06\xdc\x5d\xa0\x4d\x73\xd0\x6d\xd3\xa4\x85\xe1\xda\x4d\xbc\x1b\xe4\x10\xe4\xc0\x25\x47\xd2\xd4\x14\x29\xf3\xc7\x1b\xa1\xe8\xbb\x17\x43\x4a\x96\xd6\xcd\x1a\x3e\x71\x25\xcd\xf7\x33\x1f\xc9\xd9\xa2\x2c\xcb\x5d\x8b\xf0\x9b\x35\x06\x65\x20\x6b\x20\x0c\x3d\x42\x6d\x1d\x08\xd8\xa2\x8b\xdd\xef\xa4\xf5\x0a\xb6\x88\x10\x5a\x84\xaf\x7f\x38\xd1\xb7\x9f\xfe\x04\x69\x1d\x82\xb4\x46\x62\x1f\xfc\xb7\xb3\x36\x84\xde\x57\xeb\xb5\xb2\xd2\xaf\x54\x1d\x3d\xae\xc8\xae\x9b\x48\x0a\xfd\x9a\x6b\x7f\x9a\x6a\xd7\x0d\x33\xdc\xeb\xf5\x79\x59\x96\x45\x52\x7b\xd4\x59\xd8\xf8\xa7\x00\x28\x37\xa0\xc9\x07\xb0\x35\xa0\x6a\xd0\x43\xb0\x73\x6d\x59\x40\x7e\x5b\xc1\xd7\xc7\x97\x1f\x54\x83\xaf\xbe\xbd\x2a\x18\x7c\x69\x6a\xeb\x3a\x91\x9b\xb2\x20\x48\x41\x2f\x1a\x32\xe9\x0d\xa3\x7b\xd1\x20\x17\x55\xf0\x71\xfc\xf5\xaa\xf8\xb7\xe0\x48\x8a\x0d\x78\x32\x8d\x5e\x58\x03\xd4\xd8\xa1\x49\x66\x38\x08\x39\x5b\x65\x8f\xab\xe2\xff\xdd\xb0\x99\xdc\x47\x59\x7e\x8a\xe8\x06\x90\xd1\x79\xeb\x2e\x40\x68\x6d\x0f\x64\x1a\x18\x6c\x64\x6f\xd2\x9a\x40\x26\x22\xd4\x18\x64\xcb\x1f\x1c\xfa\xa8\x83\x07\x7c\x40\x03\xa2\x0e\xe8\x80\x4c\x40\xe7\x62\x9f\x34\x6d\xcd\x58\xb7\xb0\x71\x01\x07\x0a\xad\x8d\x01\x3a\xf2\xec\x1e\x04\xec\x51\x84\x15\x1b\x83\x51\xba\x82\x6d\x70\x64\x9a\x9c\x50\xde\xfb\xb9\x45\xbb\xff\x1b\xe5\x04\x30\x56\x61\x35\x7f\x4c\xd1\x1c\xf7\x37\xf5\xc6\x24\x5e\xdb\x00\x26\x76\xfb\x64\x14\x0e\x2d\xc9\x36\xe5\xb4\x60\x97\x32\x3a\x87\x6a\xe4\x67\xc4\x4d\xec\x2a\xf8\x4c\x26\xbc\x7d\xb3\x74\x14\x9c\x30\x5e\xa4\xae\x5e\x7b\x20\xa3\xf0\x7b\x6a\x8e\x4c\xa2\x4c\x5a\x87\x16\x1d\x3e\xaf\xb0\xa0\xb9\x64\x8e\x1f\x49\x91\xf1\xc1\xc5\x93\x52\x0b\x8a\x97\x28\x2e\xd8\x4e\x2a\x5a\xa7\xd0\x4d\x51\x45\x43\xf7\x11\x81\x14\x9a\x40\x35\xa1\x9b\xc4\x05\x34\xc4\x5b\xdf\x09\x77\x87\xd3\x96\x24\xe8\xa9\xcc\xa8\x43\x1f\x44\xd7\x4f\x07\x74\xaf\xad\xbc\x5b\xee\x05\xf9\x67\xa2\x9a\xd0\x15\xec\xa8\xc3\x27\x9b\xa1\xd0\xbd\xf6\xd0\xc7\xbd\x26\x09\x77\x38\xa4\xf1\x70\x4c\x78\x01\xb4\xc2\x55\x12\xb6\x86\x53\x12\x01\xf6\x71\x00\xeb\xc0\xa3\xd6\x33\xe2\xa8\xa1\xcc\xfd\xa3\x43\x99\xcb\x12\xec\x85\x2d\x64\xc4\x78\x62\xaf\xd3\xc3\x44\xf8\x25\xc1\x3d\x29\x7c\xca\x41\x9e\x25\x2e\x00\x29\xb4\xe8\xe0\xdd\xe5\x7b\x38\x63\xbf\xe8\xce\xd9\xfa\x66\x7b\x05\x67\xfb\x38\xa0\x3b\x9f\x4e\x2d\xa5\x5b\x41\x0a\x77\x43\x7f\x94\xd3\x7d\x14\x26\x50\x18\x52\xfe\xf6\x0e\x4d\x0e\xe1\x20\x3c\x38\x94\x48\x0f\xa8\x60\x3c\x55\xf3\x0c\x4b\x9c\x13\xf2\x76\x2c\xab\x60\xc7\xf8\x4d\x67\xa3\x09\x2f\x94\xe8\x79\xbc\xf1\xe5\x7f\x5e\xe2\xa3\xa0\xd3\xf4\xbd\x23\x89\x99\xe9\x71\xbb\xf0\xbb\x90\x61\xf9\x47\xc0\x85\x0a\x8d\xed\x48\xe6\xd1\x4a\x1e\x84\x3e\x88\x81\x6f\x0f\x9c\xed\x85\xc7\xd1\xdb\x1a\xee\xa3\x0d\xe3\xd3\x79\xf6\x91\x24\x4e\x0f\x21\xc4\x1d\xdf\x01\x8d\x0f\xa8\x9f\x18\xa9\x59\x3d\x93\xd4\xb9\xac\x3a\x02\xcd\xf3\x69\xd1\x1d\xa4\x11\xf5\x20\x74\xc4\xf9\xce\x00\x28\xf2\xbd\x16\xc3\xe4\x83\x0f\x22\x63\xc6\x60\x98\x08\x4d\xec\x8e\x3d\x31\xd1\xbb\xcd\xf6\x43\x01\xb0\xbd\xbd\xfe\x39\x2f\xbf\xe4\xe5\x4d\x5e\x7e\xcd\xcb\xdb\x02\xe0\x7a\x7b\x7b\x3d\xf3\x8c\xa7\x25\x71\x6c\xb6\x57\xcc\x74\xf9\xbe\x00\xf8\x7c\x73\x75\xf3\xd7\x97\x1b\x2e\xfc\x2f\x00\x00\xff\xff\x58\x12\x5b\x8b\x89\x07\x00\x00")

func serum_fillGraphqlBytes() ([]byte, error) {
	return bindataRead(
		_serum_fillGraphql,
		"serum_fill.graphql",
	)
}

func serum_fillGraphql() (*asset, error) {
	bytes, err := serum_fillGraphqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "serum_fill.graphql", size: 1929, mode: os.FileMode(0644), modTime: time.Unix(1614612948, 0)}
	a := &asset{bytes: bytes, info: info, digest: [32]uint8{0xdd, 0x66, 0x54, 0x9e, 0x42, 0x2, 0x8e, 0x4b, 0x93, 0xaf, 0x50, 0xd8, 0xbe, 0x22, 0xc8, 0x93, 0x5c, 0x81, 0x17, 0x82, 0xd1, 0xe1, 0xa0, 0xb3, 0x5b, 0x58, 0xa4, 0x51, 0xb3, 0x94, 0x33, 0x50}}
	return a, nil
}

var _serum_marketGraphql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xdc\x54\x4f\x4f\xdb\x4e\x10\xbd\xfb\x53\x3c\x7c\x01\x24\x48\x24\x14\xfd\x0e\xbe\x21\xf2\xa3\xaa\xd4\x4a\x45\x81\x5e\x10\x87\xc1\x1e\xdb\x5b\xd6\x3b\x66\x77\x96\x28\xaa\xf8\xee\xd5\xda\x0e\x49\x4a\x3e\x41\x4f\x89\x67\xdf\x9f\x79\x4f\xf6\xe6\x79\x7e\xdf\x32\x6e\xc4\x39\x2e\xd5\x88\x83\x6e\x7a\x46\x2d\x1e\x84\x15\xfb\xd8\x7d\x27\xff\xc2\x3a\xc3\x8a\x19\xda\x32\x1e\xbf\x78\xea\xdb\xbb\x6f\x28\xc5\x33\x4a\x71\x25\xf7\x1a\x9e\xce\x5a\xd5\x3e\x14\xf3\x79\x25\x65\x98\x55\x75\x0c\x3c\x33\x32\x6f\xa2\xa9\x38\xcc\x13\xf6\x72\x8b\x9d\x37\x49\xe1\xd5\xce\xcf\xf3\x3c\xcf\x06\xbf\x3d\xa7\xbd\x55\x7e\x67\x40\x9e\xe7\xd7\xb0\x26\x28\xa4\x06\x57\x0d\x07\xa8\x8c\xf8\x5b\x63\x6d\x52\xc0\x38\x2f\xf0\xb8\x27\xf3\x7f\xd5\xf0\xc9\xd3\x49\x36\x4a\x7c\x75\xb5\xf8\x8e\xc6\x80\x02\x32\x15\x7a\x6a\x8c\x1b\x26\xa3\x46\x4f\x0d\x27\x58\x81\x1f\xd3\xbf\x2d\xf9\x5e\x94\x2c\x4a\x89\x6e\x58\xa2\x1b\xf4\xa1\x2d\x29\xd6\xa9\x81\x68\x2b\x78\xd6\xe8\x27\x25\x4d\xf8\x9b\x04\x2f\xf0\x60\x9c\xfe\xb7\x38\xc9\xde\xb3\x2c\x1d\x5e\x23\x18\xd7\xd8\x83\xc0\x69\x53\xb0\xe5\x8e\x47\xfd\x54\x72\xb9\x2b\x21\x65\x9f\x65\xc7\x9a\x1a\x88\x53\x47\x77\x91\xfd\x06\x65\xf4\x41\xfc\x05\xc8\x5a\x59\x1b\xd7\x60\x23\x31\xe5\x2d\xc5\xa9\x71\x91\x51\xb3\x96\x6d\x3a\xf0\x1c\xa2\xd5\x00\x7e\x63\x07\xaa\x95\x3d\x8c\x53\xf6\x3e\xf6\x83\xab\xd4\x89\xeb\xf7\x16\xb9\xc0\xda\x68\x2b\x51\xd1\x99\x90\x52\x80\xf0\xcc\xa4\xb3\x31\xf4\x68\x5d\x60\xa5\xde\xb8\xe6\xa3\xba\xf6\x60\x63\xc8\xf3\x2f\x2e\xb7\x14\x27\x15\x17\xfb\xc7\x43\x4d\x7f\xa7\xc4\x36\x62\xd2\x1a\xbb\x3f\x0d\xa0\xaa\xf2\x1c\xc2\xa4\x34\x3d\x1d\x73\xff\x60\x54\x26\xf4\x96\x36\x70\xd4\xf1\xe4\x4f\x1d\x6f\x29\x9f\x18\x01\xcf\x14\x18\x2a\x2f\xec\x26\x97\x34\xb8\x4f\xcf\x05\x86\x9f\xcf\x36\x01\xaf\x51\xf4\x90\x35\x4c\x0e\x69\x47\x52\x2e\xc9\xd8\xcd\x92\x94\xfe\xbd\xb4\x3b\xda\xc3\x6a\x89\x37\xb1\xb1\x63\xbc\x91\x8d\xe3\x35\x93\x5e\x77\x4b\x41\x71\xb5\x40\x2b\xd1\x07\x9c\x9d\x3a\x59\x9f\xe2\x12\x57\x8b\x36\xbd\xbd\x4e\xd6\xe7\x93\x7c\x02\x5e\x2d\xda\x9f\x83\xc8\xc3\x6a\x59\xe0\xd6\x0a\x0d\x5f\xd8\xce\xa6\x4a\x5d\xa2\x22\xa5\x89\x35\x0c\xf6\x38\x8f\xcb\xdd\x20\xdd\x11\xef\xd9\x9f\x00\x00\x00\xff\xff\xe2\xfa\xbb\xb4\x07\x05\x00\x00")

func serum_marketGraphqlBytes() ([]byte, error) {
	return bindataRead(
		_serum_marketGraphql,
		"serum_market.graphql",
	)
}

func serum_marketGraphql() (*asset, error) {
	bytes, err := serum_marketGraphqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "serum_market.graphql", size: 1287, mode: os.FileMode(0644), modTime: time.Unix(1614636021, 0)}
	a := &asset{bytes: bytes, info: info, digest: [32]uint8{0x6f, 0x62, 0x30, 0xe2, 0x74, 0x5b, 0x1b, 0x5, 0x85, 0x6a, 0x9a, 0xef, 0x10, 0x97, 0x32, 0x55, 0x97, 0xf8, 0xf8, 0x58, 0x45, 0x33, 0x89, 0xdb, 0xe2, 0x84, 0x3b, 0xf0, 0x37, 0x81, 0x87, 0xb3}}
	return a, nil
}

var _serum_orderGraphql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xa4\x94\xcb\x6a\xdb\x40\x14\x86\xf7\x7a\x8a\x3f\xda\x24\x81\xe0\x6e\x4a\x17\xda\xb9\x89\x4b\x45\xe3\x38\xc4\x2a\x25\x94\x62\x46\x9a\xe3\xea\x60\x69\x46\x99\x0b\xae\x28\x79\xf7\x32\xa3\xb8\x56\x1a\x85\x16\xba\x94\x35\xff\xf7\xcd\xb9\x58\x89\xeb\x3b\xc2\x9a\x8c\x6f\x57\x46\x92\xc1\xcf\x04\x48\xd3\xb4\xa8\x09\x3a\xfe\xa0\x7c\x5b\x92\x81\x57\xfc\xe0\x09\x2c\x49\x39\xde\x32\x99\x34\x4d\x13\x84\xb7\x19\x3e\xb3\x72\xef\xde\x9e\x24\xc7\xa8\x6d\xb4\x3b\x24\x59\x61\x5f\x73\x55\xc3\xd5\xcf\x4c\xba\xaa\xbc\x31\x24\x67\x03\x29\x44\x6e\xa6\x69\xce\x08\x65\x45\xe5\x58\xab\x53\x8b\x5a\xd8\x1a\x7b\x76\x35\xab\x88\x8c\xae\x7d\x4d\x86\xfe\x62\x18\x61\x3e\x0a\x5b\x67\x58\x3b\xc3\xea\xfb\xeb\x26\x56\x92\x7e\xfc\xaf\x2a\x0f\x90\xa9\xaa\x58\x59\x67\xfc\xab\xae\x11\xe2\x9f\x94\x23\xdc\xab\x4a\xc7\x2d\x59\x27\xda\x0e\x7a\x1b\x71\x65\xa3\xab\xdd\x78\x42\x6c\x07\xc5\x07\x6e\x9a\x17\x45\x1d\xd2\x19\x0a\x6e\xe9\x8f\xbe\x49\x32\xa7\x16\x9d\x2f\x1b\xae\xb0\xa3\x1e\x5b\x6d\x46\xc0\x78\xe7\x0b\xf0\x8c\x66\xd1\xac\x55\x28\x48\x38\x94\xbe\x87\x36\xb0\xd4\x34\xc7\x48\x2b\xcc\x8e\xdc\xb1\x99\x92\xcc\xd4\xb8\x86\x63\x31\x36\x51\xc3\x64\x9b\x86\x48\x36\x9c\x58\xc6\x87\x03\xf1\x4b\xcc\x5b\x96\xf4\x02\xc2\x36\x48\x2e\x40\xec\x6a\x32\x78\x9f\x5f\xe1\x2c\xdc\x98\xcc\x79\xb8\xfc\x7c\xfd\x09\x67\xa5\xef\xc9\x9c\x1f\xb6\x99\x25\x65\x58\xb3\xa4\xa2\xef\x9e\xb5\xaa\x33\x5c\x11\x6c\x47\x55\xf8\x1b\x49\x3c\x8d\xfb\x28\x9b\x21\x1c\x93\xa4\x74\xcb\x95\x88\xf3\x67\x0b\xd1\xec\x45\x1f\xd6\x04\x67\xa5\xb0\x04\xa7\x77\xa4\xf0\x06\x0f\x5e\xbb\xa7\xa7\xf3\x41\x1d\x05\x53\xdd\x92\x64\xd9\x90\xc4\x83\x17\xca\xb1\xeb\x5f\xaa\x07\xc0\xe1\x7d\x86\x22\x60\xe7\xad\xf6\xca\x8d\x41\xa3\xc6\x84\xcf\xc7\xd3\x94\xfa\x8e\x32\xac\xee\xae\x16\x77\x9b\xe2\xfe\x76\x31\x0e\x6c\xb9\x69\x2c\x84\xb5\xba\x62\xe1\x48\xc2\xe9\x61\x09\x8e\xd2\x78\x24\xc3\xd7\xdf\xdb\x77\xf2\xed\x24\x79\x4c\x12\x52\xbe\x1d\x51\xe3\xf7\xe9\x3a\x5f\xe6\x45\x02\xe4\xcb\xe5\xe2\x2a\x9f\x17\x8b\xcd\xea\x6e\x73\x39\xbf\xb9\x5c\x5c\x27\xc0\xed\x6a\x5d\x6c\x56\x37\xd7\xf7\xc9\x63\xf2\x2b\x00\x00\xff\xff\x68\xe1\x2d\x12\xdf\x04\x00\x00")

func serum_orderGraphqlBytes() ([]byte, error) {
	return bindataRead(
		_serum_orderGraphql,
		"serum_order.graphql",
	)
}

func serum_orderGraphql() (*asset, error) {
	bytes, err := serum_orderGraphqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "serum_order.graphql", size: 1247, mode: os.FileMode(0644), modTime: time.Unix(1614613053, 0)}
	a := &asset{bytes: bytes, info: info, digest: [32]uint8{0x71, 0x27, 0xa0, 0xe2, 0x2a, 0xa4, 0x62, 0x76, 0x44, 0x6, 0xaa, 0x31, 0x85, 0x21, 0x74, 0x90, 0x56, 0x78, 0x33, 0xa6, 0x8d, 0x4f, 0x45, 0xe3, 0x6c, 0x31, 0x7e, 0x39, 0x61, 0x1e, 0xf7, 0xc3}}
	return a, nil
}

var _serum_order_trackerGraphql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xac\x55\xcb\x6e\xeb\x36\x10\xdd\xfb\x2b\x26\xea\xa2\x9b\xdb\xbb\x2a\xba\x30\x50\x14\x82\xa3\xa0\x42\x53\xd9\xb0\x95\xf6\xee\x1a\x5a\x1c\x5b\x83\x48\x43\x87\x1c\xc6\x31\xd2\xfe\x7b\x41\xbd\xed\xc4\xed\x26\x5e\x99\x33\xc3\x73\x0e\xe7\xa5\xd9\x0c\xd9\xd7\xb0\x5c\xdf\x26\xeb\xbf\x36\x79\x9c\x27\xf0\x36\x03\x00\x78\xc8\x7e\xcb\x96\x7f\x66\xb3\xe6\x10\x45\x11\xa4\xac\xa9\x50\x82\x0e\x8e\x25\x32\x48\x89\x60\xac\x46\x0b\xa5\x72\xb0\x45\x64\x88\x8b\x02\x0f\x82\x1a\xb6\xa7\xc6\x8d\xaf\x45\xa9\x78\x8f\x51\x14\x35\x28\xf1\x6a\xb5\x5e\xfe\x91\xdc\x8e\x98\x31\x77\x18\xe4\x80\x02\x26\x39\x70\xa2\xc4\x77\x24\x24\x23\x7a\x61\xea\x43\x85\x82\xd5\x09\x0a\xc5\x05\x56\x15\x6a\x50\xac\x27\x4a\xc8\x01\x1b\xa8\x0c\xef\xd1\x82\x39\x20\xf7\xcc\x8b\x38\x5b\x24\xf7\xf7\x13\xea\x18\x0e\xca\x0a\xa9\x0a\x9e\xbd\x62\x21\x39\x8d\x4c\xf8\x8a\x85\x17\xd4\x5f\x60\xeb\x25\xc0\x5b\x04\x65\x11\x9c\x50\x55\x81\x2b\x95\x45\xd7\xc0\x83\x94\x4a\x1a\xd7\x51\x91\x10\xef\x41\x0c\x6c\x11\x76\xd4\x88\x1b\xbc\xca\x39\x53\x90\x0a\xa9\x39\x92\x94\xa3\xe2\xaf\xbd\xc0\x55\xbc\xce\xd3\xf8\x7e\x90\x97\x87\xf4\xb1\x90\xc5\x51\x9f\xd9\x81\xe2\xcb\x9c\x4f\xb2\xd2\xb2\xf6\x88\xc9\xb7\x64\xf1\x90\x27\xb7\xb3\x7f\x66\x33\x39\x1d\x10\x36\x68\x7d\xbd\x0c\xb7\x37\xa2\x04\xbb\x32\x77\x5c\x2d\x2a\xfb\x7a\x8b\x16\x3c\xd3\xb3\x47\x20\x1d\x14\xec\x08\x6d\x8f\xc9\xbe\x9e\xc3\x03\xb1\xfc\xf4\xe3\xcd\x20\xf5\x60\xf1\x85\x8c\x77\x2d\x2a\xb9\xe6\x75\xbd\x31\x88\x0e\xe7\x91\x7c\x78\xf2\xd9\xbd\xf9\xb4\x03\x47\xec\xc2\x5b\x8b\x2c\x67\xd0\x9d\x0d\x9e\xd8\x1c\xb9\xe9\x16\xfc\x0f\x96\x29\xc2\x15\x12\xb1\x8a\x1d\x09\x19\xce\x54\xfd\x8e\x66\xf4\xf6\x2c\x8d\x45\x15\xc1\x34\xd0\x9c\x63\xf4\x44\xf9\x3a\xce\x36\x69\x9e\x2e\xb3\x8e\x6d\x0c\x9b\x43\xa3\x33\x1f\x0c\x37\xa1\x50\x9e\x03\xcf\x85\x07\x7e\xbe\xb4\xa4\x4c\x02\x7f\x5f\x5a\x87\xf1\x7b\xe7\x59\x0c\xf3\xf2\xce\xb5\x6a\xa7\xa0\x3a\xdd\xd1\xc7\x01\x49\x37\x0c\x5d\x17\x7d\xa4\xa4\xed\xa4\xef\xe0\x2e\xfd\xf6\x7b\x02\x9b\xd2\xf8\x6a\x3a\x95\x3b\xc2\x4a\x87\xb1\xb0\xf8\xec\xc9\xa2\xfe\x05\x92\x97\x30\xdc\x1c\xfa\x99\x02\x42\x8d\xce\xa9\x3d\x7e\x81\x14\xb4\xe1\xef\x05\x1c\x22\x94\xe6\x08\x47\x84\xa2\x81\x8b\xd8\x48\x04\xa5\x7a\xc1\xb6\x3a\xc6\x5a\x74\x07\xc3\x9a\x78\xdf\xd1\xb7\xbf\x76\xae\x7a\x15\x5b\x04\x6d\xfc\xb6\xc2\x1f\x8a\x12\x8b\xa7\x71\x63\x30\xb4\xbb\x49\x7f\xed\xdb\xe0\xb1\xb9\xf9\xd8\xd7\x7f\xec\x25\xd0\x28\x8a\x2a\x37\x14\xbb\x09\x9c\x4f\x22\x86\x19\xbb\x56\x91\xb7\x4f\x20\xb9\xb9\xc6\x32\x56\xb7\xa5\x19\xb6\xe3\xa6\x32\x92\x4d\x47\xf6\xcc\x9b\x8f\x6d\xfc\xab\x72\xe5\x1c\x36\x62\x89\xf7\xd7\xa3\x52\xd6\xf8\x7a\x05\x2c\x65\x27\xd6\xff\x6f\x58\x2c\x73\xc8\xa9\xc6\xab\x6f\xb9\x6c\xc7\x21\x71\x19\x1e\x9b\x05\xd7\x2e\xd5\x7e\xeb\x4d\xda\xac\xcf\x9b\xd2\x1a\x75\xb8\xdd\xe5\x2e\xfc\xbd\x4a\xd7\x37\xf7\xe7\x15\x68\xf2\x35\x1d\xa7\xbf\x83\x8f\x17\x8b\x64\x15\x56\xf2\xf9\x27\x29\x9c\xee\xd2\xe1\xef\x64\x71\xff\x1b\x00\x00\xff\xff\xbb\xe8\x84\x2c\x9d\x07\x00\x00")

func serum_order_trackerGraphqlBytes() ([]byte, error) {
	return bindataRead(
		_serum_order_trackerGraphql,
		"serum_order_tracker.graphql",
	)
}

func serum_order_trackerGraphql() (*asset, error) {
	bytes, err := serum_order_trackerGraphqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "serum_order_tracker.graphql", size: 1949, mode: os.FileMode(0644), modTime: time.Unix(1614613044, 0)}
	a := &asset{bytes: bytes, info: info, digest: [32]uint8{0xda, 0x64, 0xa5, 0xec, 0xa4, 0x52, 0xcb, 0x11, 0x43, 0x21, 0xfe, 0xd1, 0xcc, 0x3e, 0x62, 0x89, 0xe6, 0xd, 0xa1, 0xb, 0xc, 0x25, 0x88, 0xfb, 0x91, 0xaf, 0x35, 0x89, 0x43, 0x42, 0x7, 0x19}}
	return a, nil
}

var _tokenGraphql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x6c\x91\x41\x6f\xd4\x3e\x10\xc5\xef\xfe\x14\x6f\x73\xfa\xff\xa5\xb2\xb9\x20\x0e\xb9\x2d\x08\xa1\x4a\x20\x51\xb5\x9c\xaa\x1e\xbc\xf6\x24\x31\x38\x9e\x60\x8f\xbb\x0a\xa8\xdf\x1d\xd9\xd9\x65\x53\x89\x5b\x32\xf3\x66\xde\xef\x8d\x95\x6a\x9a\xe6\x61\x24\x7c\xe0\x10\xc8\x88\xe3\x00\x59\x66\x42\xcf\x11\x1a\x0f\xfc\x83\xc2\x1e\xf7\x44\x90\x91\xf0\xf8\x29\xea\x79\xbc\xfb\x0c\xc3\x91\x60\x38\x18\x9a\x25\x3d\xfd\x37\x8a\xcc\xa9\x6b\x5b\xcb\x26\xed\x6d\x9f\x13\xed\x1d\xb7\x43\x76\x96\x52\x5b\xb4\x6f\x2e\xda\x76\x28\x1b\x7e\xfa\xf6\xff\xa6\x69\x54\x75\xaa\x1e\x1b\xfb\xdf\x0a\x68\x0e\xf0\x2e\x09\xb8\x07\xd9\x81\x12\x84\x57\x1d\x1a\x85\xb5\xd4\xe1\xb1\x56\x3e\xda\x81\x76\x4f\x3b\x55\xa6\x6e\x43\xcf\x71\xd2\x6b\x0a\x86\x76\x16\xb3\x1e\x5c\xa8\x95\x32\x39\xeb\x81\x8a\xa8\xc3\xd7\xf3\xd7\x4e\xbd\xd4\x1b\xa8\x03\x92\x0b\x83\x3f\xf3\x80\x3c\x4d\x14\x2a\x41\x49\x6e\xae\x7c\x05\x6c\xaf\x5e\xe3\x17\x88\x15\xbc\x69\xee\x32\xc5\x05\x26\xc7\xc4\xf1\x06\xda\x7b\x3e\xb9\x30\x60\xe1\x5c\x98\x0c\x07\x71\x21\x13\x7a\x12\x33\x96\x46\xa4\x94\xbd\x24\xd0\x33\x05\xe8\x5e\x28\xc2\x05\xa1\x18\xf3\x5c\xfd\xb8\x2f\xb3\x71\x83\x70\x83\x93\x93\x91\xb3\x60\x72\xa9\x50\x43\xe3\x48\x5a\xf6\x05\x0a\x67\xeb\x0e\xf7\x12\x5d\x18\xd6\xcb\xac\x8f\xbc\x46\xe3\xe3\x77\x32\x17\x71\x60\x4b\xdd\xda\xa8\xa7\xb8\x66\xaa\x79\xb4\xb5\x91\x52\xba\x2e\x03\x26\x17\xe4\x90\x65\xe4\xe8\x64\xb9\x34\x14\xd0\x47\xa2\x5f\xf4\xaf\x4e\xca\xf3\xec\x97\x0e\xdf\x5c\x90\x77\x6f\x15\x60\xc9\xb8\x49\xfb\xd4\xe1\x36\x88\x02\x9e\x29\xba\xde\x91\xed\xf0\x9e\xd9\x93\x0e\xd5\x87\x44\x9f\xc9\xbe\x90\xe8\xd7\x70\xa5\x52\x01\xd3\x32\x1d\xd9\x6f\xf9\x82\x9e\x68\xfb\xef\x79\xe0\x0d\xcc\x89\x8e\xc9\xc9\x5f\x85\x7a\x51\x7f\x02\x00\x00\xff\xff\xca\x3a\xed\xe1\x04\x03\x00\x00")

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

	info := bindataFileInfo{name: "token.graphql", size: 772, mode: os.FileMode(0644), modTime: time.Unix(1614616254, 0)}
	a := &asset{bytes: bytes, info: info, digest: [32]uint8{0x8d, 0xee, 0x88, 0x74, 0xfe, 0x72, 0xb2, 0x96, 0xe9, 0xa4, 0xb3, 0xe2, 0x3f, 0xa9, 0xb, 0x7, 0xcf, 0xf3, 0x26, 0x42, 0x97, 0x5d, 0xa1, 0x38, 0xec, 0x17, 0x9a, 0x5e, 0x50, 0xe3, 0x89, 0x85}}
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
	"analytics.graphql":           analyticsGraphql,
	"query.graphql":               queryGraphql,
	"query_alpha.graphql":         query_alphaGraphql,
	"schema.graphql":              schemaGraphql,
	"serum_fill.graphql":          serum_fillGraphql,
	"serum_market.graphql":        serum_marketGraphql,
	"serum_order.graphql":         serum_orderGraphql,
	"serum_order_tracker.graphql": serum_order_trackerGraphql,
	"token.graphql":               tokenGraphql,
}

// AssetDebug is true if the assets were built with the debug flag enabled.
const AssetDebug = false

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
	"analytics.graphql": {analyticsGraphql, map[string]*bintree{}},
	"query.graphql": {queryGraphql, map[string]*bintree{}},
	"query_alpha.graphql": {query_alphaGraphql, map[string]*bintree{}},
	"schema.graphql": {schemaGraphql, map[string]*bintree{}},
	"serum_fill.graphql": {serum_fillGraphql, map[string]*bintree{}},
	"serum_market.graphql": {serum_marketGraphql, map[string]*bintree{}},
	"serum_order.graphql": {serum_orderGraphql, map[string]*bintree{}},
	"serum_order_tracker.graphql": {serum_order_trackerGraphql, map[string]*bintree{}},
	"token.graphql": {tokenGraphql, map[string]*bintree{}},
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
