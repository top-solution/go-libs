package fs

import "io/fs"

// FallbackFS is a FS that fallbacks to a default file (useful for SPAs)
type FallbackFs struct {
	fs.FS
	Fallback        string
	GetFallbackFunc func(string) string
}

func (f *FallbackFs) Open(name string) (fs.File, error) {
	file, err := f.FS.Open(name)
	if err != nil {
		if f.GetFallbackFunc != nil {
			file, err = f.FS.Open(f.GetFallbackFunc(name))
		} else {
			file, err = f.FS.Open(f.Fallback)
		}
	}
	if err != nil {
		return nil, err
	}
	return file, err
}
