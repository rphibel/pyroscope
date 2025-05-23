package objstore

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/grafana/dskit/multierror"
	"github.com/thanos-io/objstore"

	"github.com/grafana/pyroscope/pkg/util/bufferpool"
)

var _ objstore.BucketReader = &ReadOnlyFile{}

type ReadOnlyFile struct {
	size    int64
	name    string
	path    string
	mu      sync.Mutex
	readers []*fileReader
}

func Download(ctx context.Context, name string, src BucketReader, dir string) (*ReadOnlyFile, error) {
	f, err := download(ctx, name, src, dir)
	if err != nil {
		return nil, fmt.Errorf("downloading %s: %w", name, err)
	}
	return f, nil
}

func download(ctx context.Context, name string, src BucketReader, dir string) (*ReadOnlyFile, error) {
	r, err := src.Get(ctx, name)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	path := filepath.Join(dir, filepath.Base(name))
	f := &ReadOnlyFile{
		name: name,
		path: path,
	}
	if err = os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	dst, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	defer dst.Close()

	buf := bufferpool.GetBuffer(32 << 10)
	defer bufferpool.Put(buf)
	buf.B = buf.B[:cap(buf.B)]
	n, err := io.CopyBuffer(dst, r, buf.B)
	if err != nil {
		_ = os.RemoveAll(path)
		return nil, err
	}

	f.size = n
	return f, nil
}

func (f *ReadOnlyFile) Close() error {
	var m multierror.MultiError
	for _, r := range f.readers {
		m.Add(r.Close())
	}
	m.Add(os.RemoveAll(f.path))
	f.readers = f.readers[:0]
	return m.Err()
}

func (f *ReadOnlyFile) Iter(context.Context, string, func(string) error, ...objstore.IterOption) error {
	return nil
}

func (f *ReadOnlyFile) IterWithAttributes(context.Context, string, func(attrs objstore.IterObjectAttributes) error, ...objstore.IterOption) error {
	return nil
}

func (f *ReadOnlyFile) SupportedIterOptions() []objstore.IterOptionType {
	return nil
}

func (f *ReadOnlyFile) Exists(_ context.Context, name string) (bool, error) {
	return name == f.name, nil
}

func (f *ReadOnlyFile) IsObjNotFoundErr(err error) bool { return os.IsNotExist(err) }

func (f *ReadOnlyFile) IsAccessDeniedErr(err error) bool { return os.IsPermission(err) }

func (f *ReadOnlyFile) Attributes(_ context.Context, name string) (attrs objstore.ObjectAttributes, err error) {
	if name != f.name {
		return attrs, os.ErrNotExist
	}
	return objstore.ObjectAttributes{
		Size:         f.size,
		LastModified: time.Unix(0, 0), // We don't care.
	}, nil
}

func (f *ReadOnlyFile) ReaderAt(_ context.Context, name string) (ReaderAtCloser, error) {
	return f.borrowOrCreateReader(name)
}

func (f *ReadOnlyFile) Get(_ context.Context, name string) (io.ReadCloser, error) {
	r, err := f.borrowOrCreateReader(name)
	if err != nil {
		return nil, err
	}
	if _, err = r.Seek(0, io.SeekStart); err != nil {
		_ = r.Close()
		return nil, err
	}
	return r, nil
}

func (f *ReadOnlyFile) GetRange(_ context.Context, name string, off, length int64) (io.ReadCloser, error) {
	if off < 0 || length < 0 {
		return nil, fmt.Errorf("%w: invalid offset", os.ErrInvalid)
	}
	r, err := f.borrowOrCreateReader(name)
	if err != nil {
		return nil, err
	}
	if _, err = r.Seek(off, io.SeekStart); err != nil {
		_ = r.Close()
		return nil, err
	}
	r.reader = io.LimitReader(r.reader, length)
	return r, nil
}

func (f *ReadOnlyFile) borrowOrCreateReader(name string) (*fileReader, error) {
	if name != f.name {
		return nil, os.ErrNotExist
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.readers) > 0 {
		ff := f.readers[len(f.readers)-1]
		f.readers = f.readers[:len(f.readers)-1]
		ff.reader = ff.File
		return ff, nil
	}
	return f.openReader()
}

func (f *ReadOnlyFile) returnReader(r *fileReader) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.readers = append(f.readers, r)
}

func (f *ReadOnlyFile) openReader() (*fileReader, error) {
	ff, err := os.Open(f.path)
	if err != nil {
		return nil, err
	}
	return &fileReader{
		parent: f,
		File:   ff,
		reader: ff,
	}, nil
}

type fileReader struct {
	parent *ReadOnlyFile
	reader io.Reader
	*os.File
}

func (r *fileReader) Close() error {
	r.reader = nil
	r.parent.returnReader(r)
	return nil
}

func (r *fileReader) Read(p []byte) (int, error) {
	return r.reader.Read(p)
}

func (r *fileReader) Provider(p []byte) objstore.ObjProvider {
	return objstore.ObjProvider("READONLYFILE")
}
